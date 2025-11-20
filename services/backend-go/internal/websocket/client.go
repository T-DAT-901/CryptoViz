package websocket

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

// Subscription représente un abonnement à des données spécifiques
type Subscription struct {
	Type      string `json:"type"`      // "trades", "candles", "indicators", "news"
	Symbol    string `json:"symbol"`    // "BTC/USDT", "ETH/USDT", "*" for all
	Timeframe string `json:"timeframe"` // "5s", "1m", "15m", "1h" (for candles)
}

// Client représente une connexion WebSocket
type Client struct {
	hub           *Hub
	conn          *websocket.Conn
	send          chan []byte
	subscriptions map[string]Subscription // key = "type:symbol:timeframe"
	mu            sync.RWMutex
	logger        *logrus.Logger
	id            string
}

// NewClient crée un nouveau client
func NewClient(hub *Hub, conn *websocket.Conn, logger *logrus.Logger, id string) *Client {
	return &Client{
		hub:           hub,
		conn:          conn,
		send:          make(chan []byte, 256),
		subscriptions: make(map[string]Subscription),
		logger:        logger,
		id:            id,
	}
}

// Subscribe ajoute un abonnement
func (c *Client) Subscribe(sub Subscription) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := sub.Type + ":" + sub.Symbol + ":" + sub.Timeframe
	c.subscriptions[key] = sub

	c.logger.WithFields(logrus.Fields{
		"client_id": c.id,
		"type":      sub.Type,
		"symbol":    sub.Symbol,
		"timeframe": sub.Timeframe,
	}).Info("Client subscribed")
}

// Unsubscribe supprime un abonnement
func (c *Client) Unsubscribe(sub Subscription) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := sub.Type + ":" + sub.Symbol + ":" + sub.Timeframe
	delete(c.subscriptions, key)

	c.logger.WithFields(logrus.Fields{
		"client_id": c.id,
		"type":      sub.Type,
		"symbol":    sub.Symbol,
		"timeframe": sub.Timeframe,
	}).Info("Client unsubscribed")
}

// IsSubscribed vérifie si le client est abonné à un type de données
func (c *Client) IsSubscribed(dataType, symbol, timeframe string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Check exact match
	key := dataType + ":" + symbol + ":" + timeframe
	if _, ok := c.subscriptions[key]; ok {
		return true
	}

	// Check wildcard symbol match
	wildcardKey := dataType + ":*:" + timeframe
	if _, ok := c.subscriptions[wildcardKey]; ok {
		return true
	}

	// Check without timeframe (for trades/news)
	noTimeframeKey := dataType + ":" + symbol + ":"
	if _, ok := c.subscriptions[noTimeframeKey]; ok {
		return true
	}

	// Check wildcard without timeframe
	wildcardNoTf := dataType + ":*:"
	if _, ok := c.subscriptions[wildcardNoTf]; ok {
		return true
	}

	return false
}

// GetSubscriptions retourne les abonnements actuels
func (c *Client) GetSubscriptions() []Subscription {
	c.mu.RLock()
	defer c.mu.RUnlock()

	subs := make([]Subscription, 0, len(c.subscriptions))
	for _, sub := range c.subscriptions {
		subs = append(subs, sub)
	}
	return subs
}

// ReadPump lit les messages du client WebSocket
func (c *Client) ReadPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.logger.WithField("client_id", c.id).Errorf("WebSocket error: %v", err)
			}
			break
		}

		// Parse client message
		var clientMsg ClientMessage
		if err := json.Unmarshal(message, &clientMsg); err != nil {
			c.logger.WithField("client_id", c.id).Errorf("Failed to parse message: %v", err)
			continue
		}

		// Handle client message
		c.handleMessage(clientMsg)
	}
}

// ClientMessage représente un message reçu du client
type ClientMessage struct {
	Action    string `json:"action"`    // "subscribe", "unsubscribe", "ping"
	Type      string `json:"type"`      // "trades", "candles", "indicators", "news"
	Symbol    string `json:"symbol"`    // "BTC/USDT", "*"
	Timeframe string `json:"timeframe"` // "5s", "1m", "15m", "1h"
}

// ServerMessage représente un message envoyé au client
type ServerMessage struct {
	Type      string      `json:"type"`      // "trade", "candle", "indicator", "news", "ack", "error"
	Data      interface{} `json:"data"`      // Données spécifiques
	Timestamp int64       `json:"timestamp"` // Unix timestamp ms
}

func (c *Client) handleMessage(msg ClientMessage) {
	switch msg.Action {
	case "subscribe":
		sub := Subscription{
			Type:      msg.Type,
			Symbol:    msg.Symbol,
			Timeframe: msg.Timeframe,
		}
		c.Subscribe(sub)
		c.sendAck("subscribed", sub)

	case "unsubscribe":
		sub := Subscription{
			Type:      msg.Type,
			Symbol:    msg.Symbol,
			Timeframe: msg.Timeframe,
		}
		c.Unsubscribe(sub)
		c.sendAck("unsubscribed", sub)

	case "ping":
		c.sendAck("pong", nil)

	case "list_subscriptions":
		c.sendAck("subscriptions", c.GetSubscriptions())

	default:
		c.sendError("Unknown action: " + msg.Action)
	}
}

func (c *Client) sendAck(ackType string, data interface{}) {
	response := ServerMessage{
		Type:      ackType,
		Data:      data,
		Timestamp: time.Now().UnixMilli(),
	}

	jsonData, err := json.Marshal(response)
	if err != nil {
		c.logger.Errorf("Failed to marshal ack: %v", err)
		return
	}

	select {
	case c.send <- jsonData:
	default:
		c.logger.WithField("client_id", c.id).Warn("Client send buffer full")
	}
}

func (c *Client) sendError(errMsg string) {
	response := ServerMessage{
		Type: "error",
		Data: map[string]string{
			"message": errMsg,
		},
		Timestamp: time.Now().UnixMilli(),
	}

	jsonData, err := json.Marshal(response)
	if err != nil {
		return
	}

	select {
	case c.send <- jsonData:
	default:
	}
}

// WritePump envoie les messages au client WebSocket
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Hub closed the channel
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages to the current websocket message
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
