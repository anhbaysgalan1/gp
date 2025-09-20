/**
 * Comprehensive API client for the Go Poker Platform
 * Handles authentication, user management, balance operations, and more
 */

import {
  User,
  UserBalance,
  TransactionHistory,
  CreateUserRequest,
  LoginRequest,
  PokerTable,
  CreateTableRequest,
  UpdateTableRequest,
  JoinTableRequest,
  JoinTableResponse,
  TableListResponse
} from '../types/api';

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

export class ApiError extends Error {
  constructor(
    message: string,
    public status: number,
    public response?: any
  ) {
    super(message);
    this.name = 'ApiError';
  }
}

class ApiClient {
  private baseUrl: string;
  private token: string | null = null;

  constructor(baseUrl: string = API_BASE_URL) {
    this.baseUrl = baseUrl;

    // Initialize token from localStorage if available
    this.token = this.getCurrentToken();
  }

  /**
   * Get current token from localStorage
   */
  private getCurrentToken(): string | null {
    if (typeof window !== 'undefined') {
      return localStorage.getItem('auth_token');
    }
    return this.token;
  }

  /**
   * Set authentication token
   */
  setAuthToken(token: string) {
    this.token = token;
    if (typeof window !== 'undefined') {
      localStorage.setItem('auth_token', token);
    }
  }

  /**
   * Clear authentication token
   */
  clearAuthToken() {
    this.token = null;
    if (typeof window !== 'undefined') {
      localStorage.removeItem('auth_token');
    }
  }

  /**
   * Get current auth token
   */
  getAuthToken(): string | null {
    return this.token;
  }

  /**
   * Make HTTP request with proper error handling
   */
  private async request<T>(
    endpoint: string,
    options: RequestInit = {}
  ): Promise<T> {
    const url = `${this.baseUrl}${endpoint}`;

    const currentToken = this.getCurrentToken();
    const config: RequestInit = {
      ...options,
      headers: {
        'Content-Type': 'application/json',
        ...(currentToken && { Authorization: `Bearer ${currentToken}` }),
        ...options.headers,
      },
    };

    try {
      const response = await fetch(url, config);

      // Handle different response types
      let data: any;
      const contentType = response.headers.get('content-type');

      if (contentType && contentType.includes('application/json')) {
        data = await response.json();
      } else {
        data = await response.text();
      }

      if (!response.ok) {
        // Extract error message from various possible formats
        let errorMessage = `HTTP ${response.status}`;

        if (typeof data === 'string') {
          errorMessage = data;
        } else if (data?.error) {
          errorMessage = data.error;
        } else if (data?.message) {
          errorMessage = data.message;
        } else if (data && typeof data === 'object') {
          // If data is an object but has no message/error field, stringify it
          errorMessage = JSON.stringify(data);
        }

        throw new ApiError(
          errorMessage,
          response.status,
          data
        );
      }

      return data;
    } catch (error) {
      if (error instanceof ApiError) {
        throw error;
      }
      throw new ApiError(
        error instanceof Error ? error.message : 'Network error',
        0
      );
    }
  }

  // ============= AUTHENTICATION ENDPOINTS =============

  /**
   * Register a new user
   */
  async register(userData: CreateUserRequest): Promise<{ message: string; user_id: string }> {
    return this.request('/api/v1/auth/register', {
      method: 'POST',
      body: JSON.stringify(userData),
    });
  }

  /**
   * Login user
   */
  async login(credentials: LoginRequest): Promise<{ token: string; user: User }> {
    const response = await this.request<{ token: string; user: User }>('/api/v1/auth/login', {
      method: 'POST',
      body: JSON.stringify(credentials),
    });

    // Automatically set the token
    this.setAuthToken(response.token);

    return response;
  }

  /**
   * Logout user
   */
  async logout(): Promise<void> {
    // Note: Backend doesn't have logout endpoint yet
    // Just clear the token locally
    this.clearAuthToken();
  }

  /**
   * Verify email with token
   */
  async verifyEmail(token: string): Promise<{ message: string }> {
    return this.request('/api/v1/auth/verify-email', {
      method: 'POST',
      body: JSON.stringify({ token }),
    });
  }

  /**
   * Request password reset
   * TODO: Implement in backend
   */
  async requestPasswordReset(email: string): Promise<{ message: string }> {
    throw new Error('Password reset not implemented yet');
    // return this.request('/api/v1/auth/forgot-password', {
    //   method: 'POST',
    //   body: JSON.stringify({ email }),
    // });
  }

  /**
   * Reset password with token
   * TODO: Implement in backend
   */
  async resetPassword(token: string, newPassword: string): Promise<{ message: string }> {
    throw new Error('Password reset not implemented yet');
    // return this.request('/api/v1/auth/reset-password', {
    //   method: 'POST',
    //   body: JSON.stringify({ token, new_password: newPassword }),
    // });
  }

  // ============= USER MANAGEMENT ENDPOINTS =============

  /**
   * Get current user profile
   */
  async getCurrentUser(): Promise<User> {
    return this.request('/api/v1/user/me');
  }

  /**
   * Update user profile
   */
  async updateProfile(updates: Partial<User>): Promise<{ message: string }> {
    return this.request('/api/v1/user/profile', {
      method: 'PUT',
      body: JSON.stringify(updates),
    });
  }

  /**
   * Change password
   * TODO: Implement in backend
   */
  async changePassword(currentPassword: string, newPassword: string): Promise<{ message: string }> {
    throw new Error('Change password not implemented yet');
    // return this.request('/api/v1/user/change-password', {
    //   method: 'POST',
    //   body: JSON.stringify({
    //     current_password: currentPassword,
    //     new_password: newPassword,
    //   }),
    // });
  }

  // ============= BALANCE ENDPOINTS =============

  /**
   * Get user balance
   */
  async getBalance(): Promise<UserBalance> {
    return this.request('/api/v1/balance');
  }

  /**
   * Transfer money to game account
   */
  async transferToGame(amount: number, sessionId: string): Promise<{
    message: string;
    transaction_id: string;
    amount: number;
    session_id: string;
  }> {
    return this.request('/api/v1/balance/transfer-to-game', {
      method: 'POST',
      body: JSON.stringify({
        amount,
        session_id: sessionId,
      }),
    });
  }

  /**
   * Transfer money from game account back to main
   */
  async transferFromGame(amount: number, sessionId: string): Promise<{
    message: string;
    transaction_id: string;
    amount: number;
    session_id: string;
  }> {
    return this.request('/api/v1/balance/transfer-from-game', {
      method: 'POST',
      body: JSON.stringify({
        amount,
        session_id: sessionId,
      }),
    });
  }

  /**
   * Withdraw money from main account
   */
  async withdraw(amount: number): Promise<{
    message: string;
    transaction_id: string;
    amount: number;
    status: string;
  }> {
    return this.request('/api/v1/balance/withdraw', {
      method: 'POST',
      body: JSON.stringify({
        amount,
      }),
    });
  }

  /**
   * Get transaction history
   */
  async getTransactionHistory(limit = 10, offset = 0): Promise<TransactionHistory> {
    const params = new URLSearchParams({
      limit: limit.toString(),
      offset: offset.toString(),
    });

    return this.request(`/api/v1/balance/transactions?${params}`);
  }

  // ============= ADMIN ENDPOINTS =============

  /**
   * Get all users (admin only)
   */
  async getAllUsers(page = 1, limit = 20): Promise<{
    users: User[];
    pagination: {
      page: number;
      limit: number;
      total: number;
      pages: number;
    };
  }> {
    const params = new URLSearchParams({
      page: page.toString(),
      limit: limit.toString(),
    });

    return this.request(`/api/v1/admin/users?${params}`);
  }

  /**
   * Update user role (admin only)
   */
  async updateUserRole(userId: string, role: string): Promise<{ message: string }> {
    return this.request(`/api/v1/admin/users/${userId}/role`, {
      method: 'PUT',
      body: JSON.stringify({ role }),
    });
  }

  /**
   * Delete user (admin only)
   */
  async deleteUser(userId: string): Promise<{ message: string }> {
    return this.request(`/api/v1/admin/users/${userId}`, {
      method: 'DELETE',
    });
  }

  /**
   * Deposit money to user account (admin only)
   */
  async depositMoney(userId: string, amount: number): Promise<{
    message: string;
    user_id: string;
    amount: number;
    transaction_id: string;
  }> {
    return this.request(`/api/v1/admin/users/${userId}/deposit`, {
      method: 'POST',
      body: JSON.stringify({ amount }),
    });
  }

  /**
   * Withdraw money from user account (admin only)
   */
  async withdrawMoney(userId: string, amount: number): Promise<{
    message: string;
    user_id: string;
    amount: number;
    transaction_id: string;
  }> {
    return this.request(`/api/v1/admin/users/${userId}/withdraw`, {
      method: 'POST',
      body: JSON.stringify({ amount }),
    });
  }

  // ============= TABLE MANAGEMENT ENDPOINTS =============

  /**
   * Get list of available tables
   */
  async getTables(params?: {
    limit?: number;
    offset?: number;
    type?: 'cash' | 'tournament';
    status?: 'waiting' | 'active' | 'full' | 'closed';
  }): Promise<TableListResponse> {
    const searchParams = new URLSearchParams();
    if (params?.limit) searchParams.set('limit', params.limit.toString());
    if (params?.offset) searchParams.set('offset', params.offset.toString());
    if (params?.type) searchParams.set('type', params.type);
    if (params?.status) searchParams.set('status', params.status);

    const queryString = searchParams.toString();
    const endpoint = queryString ? `/api/v1/tables?${queryString}` : '/api/v1/tables';

    return this.request(endpoint);
  }

  /**
   * Create a new poker table
   */
  async createTable(tableData: CreateTableRequest): Promise<PokerTable> {
    return this.request('/api/v1/tables', {
      method: 'POST',
      body: JSON.stringify(tableData),
    });
  }

  /**
   * Get details of a specific table
   */
  async getTable(tableId: string): Promise<PokerTable> {
    return this.request(`/api/v1/tables/${tableId}`);
  }

  /**
   * Update table settings (only by creator)
   */
  async updateTable(tableId: string, updates: UpdateTableRequest): Promise<PokerTable> {
    return this.request(`/api/v1/tables/${tableId}`, {
      method: 'PUT',
      body: JSON.stringify(updates),
    });
  }

  /**
   * Delete a table (only by creator)
   */
  async deleteTable(tableId: string): Promise<{ message: string }> {
    return this.request(`/api/v1/tables/${tableId}`, {
      method: 'DELETE',
    });
  }

  /**
   * Join a poker table
   */
  async joinTable(tableId: string, joinData: JoinTableRequest): Promise<JoinTableResponse> {
    return this.request(`/api/v1/tables/${tableId}/join`, {
      method: 'POST',
      body: JSON.stringify(joinData),
    });
  }

  /**
   * Leave a poker table
   */
  async leaveTable(tableId: string): Promise<{ message: string; table_id: string; user_id: string }> {
    return this.request(`/api/v1/tables/${tableId}/leave`, {
      method: 'POST',
    });
  }

  // ============= UTILITY METHODS =============

  /**
   * Check if user is authenticated
   */
  isAuthenticated(): boolean {
    return !!this.getCurrentToken();
  }

  /**
   * Health check endpoint
   */
  async healthCheck(): Promise<{ status: string; timestamp: string }> {
    return this.request('/health');
  }
}

// Export singleton instance
export const apiClient = new ApiClient();

// Export the class for testing
export default ApiClient;