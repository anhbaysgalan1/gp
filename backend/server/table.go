package server

import (
	"context"
	"fmt"

	"github.com/anhbaysgalan1/gp/internal/engine"
	"github.com/anhbaysgalan1/gp/internal/services"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
)

// table is a single table or game of poker
type table struct {
	id             uuid.UUID
	name           string
	rdb            *redis.Client
	clients        map[*Client]bool
	register       chan *Client
	unregister     chan *Client
	broadcast      chan []byte
	engine         engine.PokerEngine
	game           *SimpleGameAdapter           // Simplified compatibility layer using direct GORM operations
	sessionService *services.GameSessionService // Service for managing real money game sessions
}

// newTable creates a new table using the simplified adapter
func newTable(name string, redisClient *redis.Client, pokerEngine engine.PokerEngine, tableService *services.TableService, sessionService *services.GameSessionService) *table {
	return &table{
		id:             uuid.New(),
		name:           name,
		rdb:            redisClient,
		clients:        make(map[*Client]bool),
		register:       make(chan *Client),
		unregister:     make(chan *Client),
		broadcast:      make(chan []byte),
		engine:         pokerEngine,
		game:           NewSimpleGameAdapter(tableService, name),
		sessionService: sessionService,
	}
}

func (t *table) run() {
	go t.subscribeToMessages()

	for {
		select {
		case client := <-t.register:
			t.registerClient(client)
		case client := <-t.unregister:
			t.unregisterClient(client)
		case message := <-t.broadcast:
			t.publishMessages(message)
		}
	}
}

func (t *table) registerClient(client *Client) {
	t.clients[client] = true
}

func (t *table) unregisterClient(client *Client) {
	if _, ok := t.clients[client]; ok {
		delete(t.clients, client)
	}
}

func (t *table) broadcastToClients(message []byte) {
	for client := range t.clients {
		select {
		case client.send <- message:
		default:
			close(client.send)
			delete(t.clients, client)
		}
	}
}

var ctx = context.Background()

func (t *table) publishMessages(message []byte) {
	err := t.rdb.Publish(ctx, t.name, message).Err()
	if err != nil {
		fmt.Println(err)
	}
}

func (t *table) subscribeToMessages() {
	pubsub := t.rdb.Subscribe(ctx, t.name)
	ch := pubsub.Channel()

	for msg := range ch {
		t.broadcastToClients([]byte(msg.Payload))
	}
}
