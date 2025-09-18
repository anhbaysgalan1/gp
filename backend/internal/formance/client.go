package formance

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/evanofslack/go-poker/internal/config"
	"github.com/google/uuid"
)

type Client struct {
	httpClient *http.Client
	baseURL    string
	apiKey     string
	ledgerName string
	currency   string
}

func NewClient(cfg *config.Config) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL:    cfg.FormanceAPIURL,
		apiKey:     cfg.FormanceAPIKey,
		ledgerName: cfg.FormanceLedgerName,
		currency:   cfg.FormanceCurrency,
	}
}

// FormanceError represents an error response from Formance API
type FormanceError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e FormanceError) Error() string {
	return fmt.Sprintf("formance error %s: %s", e.Code, e.Message)
}

// CreateLedgerRequest represents the request to create a ledger
type CreateLedgerRequest struct {
	Name     string                 `json:"name"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// CreateLedgerResponse represents the response from creating a ledger
type CreateLedgerResponse struct {
	Name string `json:"name"`
}

func (c *Client) CreateLedger(ctx context.Context) error {
	// Check if ledger exists using the v2 API _info endpoint
	url := fmt.Sprintf("%s/v2/%s/_info", c.baseURL, c.ledgerName)

	if err := c.makeRequest(ctx, "GET", url, nil, nil); err != nil {
		return fmt.Errorf("ledger %s doesn't exist or is not accessible: %w", c.ledgerName, err)
	}

	slog.Info("Formance ledger exists and is accessible", "ledger", c.ledgerName, "url", c.baseURL)
	return nil
}

// BalanceResponse represents balance response from Formance v2 API
type BalanceResponse struct {
	Cursor struct {
		PageSize int `json:"pageSize"`
		HasMore  bool `json:"hasMore"`
		Data     []struct {
			Address string `json:"address"`
			Volumes map[string]struct {
				Input   int64 `json:"input"`
				Output  int64 `json:"output"`
				Balance int64 `json:"balance"`
			} `json:"volumes"`
		} `json:"data"`
	} `json:"cursor"`
}

func (c *Client) GetBalance(ctx context.Context, account string) (int64, error) {
	// Use v2 API endpoint to get account with volumes expanded
	url := fmt.Sprintf("%s/v2/%s/accounts/%s?expand=volumes", c.baseURL, c.ledgerName, account)

	var response struct {
		Data struct {
			Address string `json:"address"`
			Volumes map[string]struct {
				Input   int64 `json:"input"`
				Output  int64 `json:"output"`
				Balance int64 `json:"balance"`
			} `json:"volumes"`
		} `json:"data"`
	}

	if err := c.makeRequest(ctx, "GET", url, nil, &response); err != nil {
		return 0, fmt.Errorf("failed to get balance from Formance: %w", err)
	}

	// Check if we have balance data for our currency
	if volumeData, exists := response.Data.Volumes[c.currency]; exists {
		return volumeData.Balance, nil
	}

	// Account doesn't have balance in our currency yet, return 0
	return 0, nil
}

// TransactionRequest represents a transaction request to Formance
type TransactionRequest struct {
	Postings []PostingSimple        `json:"postings"`
	Script   *string                `json:"script,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// TransactionResponse represents a transaction response from Formance v2 API
type TransactionResponse struct {
	Data struct {
		ID       int64                  `json:"id"`
		Postings []PostingData          `json:"postings"`
		Metadata map[string]interface{} `json:"metadata"`
		Date     string                 `json:"timestamp"`
	} `json:"data"`
}

func (c *Client) CreateTransaction(ctx context.Context, postings []PostingSimple, metadata map[string]string) (string, error) {
	// Use v2 API endpoint for transactions
	url := fmt.Sprintf("%s/v2/%s/transactions", c.baseURL, c.ledgerName)

	// Convert metadata to interface{} map
	metadataInterface := make(map[string]interface{})
	for k, v := range metadata {
		metadataInterface[k] = v
	}

	reqBody := TransactionRequest{
		Postings: postings,
		Metadata: metadataInterface,
	}

	var response TransactionResponse
	if err := c.makeRequest(ctx, "POST", url, reqBody, &response); err != nil {
		return "", fmt.Errorf("failed to create transaction in Formance: %w", err)
	}

	txID := response.Data.ID
	slog.Info("Created transaction in Formance", "txid", txID, "postings", len(postings))
	return fmt.Sprintf("%d", txID), nil
}

// makeRequest is a helper method to make HTTP requests to Formance API
func (c *Client) makeRequest(ctx context.Context, method, url string, body interface{}, response interface{}) error {
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Check for HTTP errors
	if resp.StatusCode >= 400 {
		var formanceErr FormanceError
		if err := json.Unmarshal(respBody, &formanceErr); err != nil {
			return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
		}
		return formanceErr
	}

	// Parse successful response
	if response != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, response); err != nil {
			return fmt.Errorf("failed to unmarshal response: %w", err)
		}
	}

	return nil
}

type PostingSimple struct {
	Source      string
	Destination string
	Amount      int64
	Asset       string
}

// TransactionHistoryResponse represents the response from transaction history query
type TransactionHistoryResponse struct {
	Data []TransactionData `json:"data"`
}

// TransactionData represents a single transaction from Formance
type TransactionData struct {
	ID       int64                  `json:"id"`
	Postings []PostingData          `json:"postings"`
	Metadata map[string]interface{} `json:"metadata"`
	Date     string                 `json:"timestamp"`
}

// PostingData represents a posting in a transaction
type PostingData struct {
	Source      string `json:"source"`
	Destination string `json:"destination"`
	Amount      int64  `json:"amount"`
	Asset       string `json:"asset"`
}

// GetTransactionHistory fetches transaction history for a user from Formance
func (c *Client) GetTransactionHistory(ctx context.Context, userID string, limit, offset int) ([]TransactionData, error) {
	// Use v2 API endpoint for transactions with cursor-based pagination
	url := fmt.Sprintf("%s/v2/%s/transactions?pageSize=%d", c.baseURL, c.ledgerName, limit)

	// For V2 API, we fetch larger batches and filter client-side for now
	// This is not ideal but works until server-side filtering is implemented
	fetchLimit := limit * 3 // Fetch more to account for filtering
	if fetchLimit > 100 {
		fetchLimit = 100 // Cap at API limit
	}
	url = fmt.Sprintf("%s/v2/%s/transactions?pageSize=%d", c.baseURL, c.ledgerName, fetchLimit)

	var response struct {
		Cursor struct {
			PageSize int               `json:"pageSize"`
			HasMore  bool              `json:"hasMore"`
			Next     string            `json:"next,omitempty"`
			Data     []TransactionData `json:"data"`
		} `json:"cursor"`
	}

	if err := c.makeRequest(ctx, "GET", url, nil, &response); err != nil {
		return nil, fmt.Errorf("failed to get transaction history from Formance: %w", err)
	}

	// Filter transactions that involve the user's accounts
	var userTransactions []TransactionData
	userWalletAccount := PlayerWalletAccount(uuid.MustParse(userID))
	userSessionPrefix := SessionPrefix(uuid.MustParse(userID))

	for _, tx := range response.Cursor.Data {
		hasUserAccount := false
		for _, posting := range tx.Postings {
			// Check wallet account (exact match)
			if posting.Source == userWalletAccount || posting.Destination == userWalletAccount {
				hasUserAccount = true
				break
			}
			// Check session accounts (prefix match)
			if (posting.Source != "" && strings.HasPrefix(posting.Source, userSessionPrefix)) ||
			   (posting.Destination != "" && strings.HasPrefix(posting.Destination, userSessionPrefix)) {
				hasUserAccount = true
				break
			}
		}
		if hasUserAccount {
			userTransactions = append(userTransactions, tx)
		}
	}

	// Apply offset and limit to filtered results
	start := offset
	if start > len(userTransactions) {
		start = len(userTransactions)
	}

	end := start + limit
	if end > len(userTransactions) {
		end = len(userTransactions)
	}

	return userTransactions[start:end], nil
}