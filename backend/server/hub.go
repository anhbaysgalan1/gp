package server

import (
	"github.com/anhbaysgalan1/gp/internal/database"
	"github.com/anhbaysgalan1/gp/internal/engine"
	"github.com/anhbaysgalan1/gp/internal/services"
	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

// Hub maintains the set of active clients and broadcasts messages to the
// clients.
type Hub struct {
	rdb            *redis.Client
	clients        map[*Client]bool
	broadcast      chan []byte
	register       chan *Client
	unregister     chan *Client
	tables         map[*table]bool
	pokerEngine    engine.PokerEngine
	tableService   *services.TableService
	sessionService *services.GameSessionService
}

func NewHub(db *gorm.DB) (*Hub, error) {
	return NewHubWithRedis(db, nil)
}

func NewHubWithRedis(db *gorm.DB, redisClient *redis.Client) (*Hub, error) {
	// Use provided Redis client, or create a new one if not provided
	var rdb *redis.Client
	if redisClient != nil {
		rdb = redisClient
	} else {
		var err error
		rdb, err = newRedisClient()
		if err != nil {
			return nil, err
		}
	}

	var pokerEngine engine.PokerEngine
	var tableService *services.TableService
	var sessionService *services.GameSessionService

	// Initialize poker engine and services only if database is provided
	if db != nil {
		var err error
		if redisClient != nil {
			// Use engine with Redis caching
			pokerEngine, err = engine.NewPokerEngineWithRedis(db, redisClient)
		} else {
			// Use engine without Redis
			pokerEngine, err = engine.NewPokerEngineImpl(db)
		}
		if err != nil {
			return nil, err
		}

		// Create table and session services using direct GORM operations
		wrappedDB := &database.DB{DB: db}
		tableService = services.NewTableService(wrappedDB)
		sessionService = services.NewGameSessionService(wrappedDB)
	}

	hub := &Hub{
		rdb:            rdb,
		clients:        make(map[*Client]bool),
		broadcast:      make(chan []byte),
		register:       make(chan *Client),
		unregister:     make(chan *Client),
		tables:         make(map[*table]bool),
		pokerEngine:    pokerEngine,
		tableService:   tableService,
		sessionService: sessionService,
	}
	return hub, nil
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.registerClient(client)
		case client := <-h.unregister:
			h.unregisterClient(client)
		case message := <-h.broadcast:
			h.broadcastToClients(message)
		}
	}
}

func (h *Hub) registerClient(client *Client) {
	h.clients[client] = true
}

func (h *Hub) unregisterClient(client *Client) {
	if _, ok := h.clients[client]; ok {
		delete(h.clients, client)
		close(client.send)
	}
}

func (h *Hub) broadcastToClients(message []byte) {
	for client := range h.clients {
		select {
		case client.send <- message:
		default:
			close(client.send)
			delete(h.clients, client)
		}
	}
}

func (h *Hub) createTable(name string) *table {
	table := newTable(name, h.rdb, h.pokerEngine, h.tableService, h.sessionService)
	go table.run()
	h.tables[table] = true
	return table
}

func (h *Hub) findTableByName(name string) *table {
	var foundTable *table
	for table := range h.tables {
		if table.name == name {
			foundTable = table
		}
	}
	return foundTable
}
