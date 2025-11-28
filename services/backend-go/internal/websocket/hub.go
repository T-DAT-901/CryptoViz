package websocket

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

// Hub maintient l'ensemble des clients actifs et diffuse les messages
type Hub struct {
	// Registered clients
	clients map[*Client]bool

	// Inbound messages from Kafka to broadcast
	broadcast chan *BroadcastMessage

	// Register requests from clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Mutex for clients map
	mu sync.RWMutex

	// Logger
	logger *logrus.Logger

	// Stats
	stats HubStats
}

// HubStats contient les statistiques du Hub
type HubStats struct {
	TotalConnections   int64 `json:"total_connections"`
	ActiveConnections  int   `json:"active_connections"`
	MessagesBroadcast  int64 `json:"messages_broadcast"`
	MessagesDelivered  int64 `json:"messages_delivered"`
	LastBroadcastTime  int64 `json:"last_broadcast_time"`
	mu                 sync.RWMutex
}

// BroadcastMessage représente un message à diffuser
type BroadcastMessage struct {
	Type      string      // "trade", "candle", "indicator", "news"
	Symbol    string      // "BTC/USDT", etc.
	Timeframe string      // "5s", "1m", etc. (for candles)
	Data      interface{} // Actual data
}

// NewHub crée un nouveau Hub
func NewHub(logger *logrus.Logger) *Hub {
	return &Hub{
		broadcast:  make(chan *BroadcastMessage, 1000),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
		logger:     logger,
		stats:      HubStats{},
	}
}

// Run démarre la boucle principale du Hub
func (h *Hub) Run() {
	h.logger.Info("WebSocket Hub started")

	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.stats.TotalConnections++
			h.stats.ActiveConnections = len(h.clients)
			h.mu.Unlock()

			h.logger.WithFields(logrus.Fields{
				"client_id":  client.id,
				"total":      h.stats.ActiveConnections,
				"historical": h.stats.TotalConnections,
			}).Info("Client registered")

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				h.stats.ActiveConnections = len(h.clients)
			}
			h.mu.Unlock()

			h.logger.WithFields(logrus.Fields{
				"client_id": client.id,
				"remaining": h.stats.ActiveConnections,
			}).Info("Client unregistered")

		case message := <-h.broadcast:
			h.broadcastMessage(message)
		}
	}
}

// broadcastMessage envoie un message à tous les clients abonnés
func (h *Hub) broadcastMessage(msg *BroadcastMessage) {
	serverMsg := ServerMessage{
		Type:      msg.Type,
		Data:      msg.Data,
		Timestamp: time.Now().UnixMilli(),
	}

	jsonData, err := json.Marshal(serverMsg)
	if err != nil {
		h.logger.Errorf("Failed to marshal broadcast message: %v", err)
		return
	}

	h.mu.RLock()
	clients := make([]*Client, 0, len(h.clients))
	for client := range h.clients {
		clients = append(clients, client)
	}
	h.mu.RUnlock()

	deliveredCount := 0
	for _, client := range clients {
		if client.IsSubscribed(msg.Type, msg.Symbol, msg.Timeframe) {
			select {
			case client.send <- jsonData:
				deliveredCount++
			default:
				// Client buffer is full, close the connection
				h.logger.WithField("client_id", client.id).Warn("Client buffer full, closing connection")
				go func(c *Client) {
					h.unregister <- c
				}(client)
			}
		}
	}

	h.stats.mu.Lock()
	h.stats.MessagesBroadcast++
	h.stats.MessagesDelivered += int64(deliveredCount)
	h.stats.LastBroadcastTime = time.Now().UnixMilli()
	h.stats.mu.Unlock()
}

// Broadcast envoie un message pour diffusion
func (h *Hub) Broadcast(msgType, symbol, timeframe string, data interface{}) {
	msg := &BroadcastMessage{
		Type:      msgType,
		Symbol:    symbol,
		Timeframe: timeframe,
		Data:      data,
	}

	select {
	case h.broadcast <- msg:
	default:
		h.logger.Warn("Broadcast channel full, dropping message")
	}
}

// BroadcastTrade diffuse un trade
func (h *Hub) BroadcastTrade(data interface{}) {
	// Extract symbol from data
	if tradeData, ok := data.(map[string]interface{}); ok {
		if symbol, ok := tradeData["symbol"].(string); ok {
			h.Broadcast("trade", symbol, "", data)
			return
		}
	}
	h.Broadcast("trade", "*", "", data)
}

// BroadcastCandle diffuse une candle
func (h *Hub) BroadcastCandle(symbol, timeframe string, data interface{}) {
	h.Broadcast("candle", symbol, timeframe, data)
}

// BroadcastIndicator diffuse un indicateur
func (h *Hub) BroadcastIndicator(symbol string, data interface{}) {
	h.Broadcast("indicator", symbol, "", data)
}

// BroadcastNews diffuse une actualité
func (h *Hub) BroadcastNews(data interface{}) {
	h.Broadcast("news", "*", "", data)
}

// RegisterClient enregistre un nouveau client
func (h *Hub) RegisterClient(conn *websocket.Conn) *Client {
	clientID := uuid.New().String()[:8]
	client := NewClient(h, conn, h.logger, clientID)
	h.register <- client
	return client
}

// GetStats retourne les statistiques du Hub
func (h *Hub) GetStats() map[string]interface{} {
	h.stats.mu.RLock()
	defer h.stats.mu.RUnlock()

	h.mu.RLock()
	activeCount := len(h.clients)
	h.mu.RUnlock()

	return map[string]interface{}{
		"total_connections":   h.stats.TotalConnections,
		"active_connections":  activeCount,
		"messages_broadcast":  h.stats.MessagesBroadcast,
		"messages_delivered":  h.stats.MessagesDelivered,
		"last_broadcast_time": h.stats.LastBroadcastTime,
	}
}

// GetClientCount retourne le nombre de clients connectés
func (h *Hub) GetClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}
