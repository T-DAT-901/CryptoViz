package consumers

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"cryptoviz-backend/internal/kafka"
	"cryptoviz-backend/models"

	"github.com/sirupsen/logrus"
)

// TradeMessage représente le message JSON du topic crypto.raw.trades
type TradeMessage struct {
	Type     string  `json:"type"`
	EventTs  int64   `json:"event_ts"` // Unix timestamp ms
	Exchange string  `json:"exchange"`
	Symbol   string  `json:"symbol"`
	TradeID  string  `json:"trade_id"`
	Price    float64 `json:"price"`
	Amount   float64 `json:"amount"`
	Side     string  `json:"side"` // 'buy' or 'sell'
}

// BroadcastFunc is a callback for broadcasting messages to WebSocket
type BroadcastFunc func(msgType, symbol, timeframe string, data interface{})

// TradeHandler gère les messages de trades depuis Kafka
type TradeHandler struct {
	kafka.BaseHandler
	repo        models.TradeRepository
	logger      *logrus.Logger
	broadcast   BroadcastFunc
	batchBuffer []*models.Trade
	batchMutex  sync.Mutex
	batchSize   int
	flushTicker *time.Ticker
	stopChan    chan struct{}
}

// NewTradeHandler crée un nouveau handler pour les trades
func NewTradeHandler(cfg *kafka.KafkaConfig, repo models.TradeRepository, logger *logrus.Logger) *TradeHandler {
	h := &TradeHandler{
		BaseHandler: kafka.BaseHandler{
			Name:   "TradeHandler",
			Topics: []string{cfg.TopicRawTrades},
		},
		repo:        repo,
		logger:      logger,
		batchBuffer: make([]*models.Trade, 0, 500),
		batchSize:   500, // Higher batch size for high-frequency trades
		flushTicker: time.NewTicker(100 * time.Millisecond),
		stopChan:    make(chan struct{}),
	}

	go h.flushLoop()
	return h
}

// SetBroadcast sets the broadcast callback for WebSocket streaming
func (h *TradeHandler) SetBroadcast(fn BroadcastFunc) {
	h.broadcast = fn
}

// flushLoop periodically flushes the batch buffer
func (h *TradeHandler) flushLoop() {
	for {
		select {
		case <-h.stopChan:
			h.flushBatch()
			return
		case <-h.flushTicker.C:
			h.flushBatch()
		}
	}
}

// flushBatch writes all buffered trades to the database
func (h *TradeHandler) flushBatch() {
	h.batchMutex.Lock()
	if len(h.batchBuffer) == 0 {
		h.batchMutex.Unlock()
		return
	}
	batch := h.batchBuffer
	h.batchBuffer = make([]*models.Trade, 0, h.batchSize)
	h.batchMutex.Unlock()

	if err := h.repo.CreateBatch(batch); err != nil {
		h.logger.Errorf("Failed to flush trade batch (%d items): %v", len(batch), err)
		return
	}

	h.logger.Debugf("Flushed %d trades to database", len(batch))

	// Broadcast trades (sample to avoid flooding)
	for i, trade := range batch {
		if h.broadcast != nil && i%10 == 0 { // Only broadcast every 10th trade
			wsData := map[string]interface{}{
				"type":      "trade",
				"exchange":  trade.Exchange,
				"symbol":    trade.Symbol,
				"trade_id":  trade.TradeID,
				"price":     trade.Price,
				"amount":    trade.Amount,
				"side":      trade.Side,
				"timestamp": trade.EventTs.UnixMilli(),
			}
			go h.broadcast("trade", trade.Symbol, "", wsData)
		}
	}
}

// Stop stops the flush goroutine
func (h *TradeHandler) Stop() {
	h.flushTicker.Stop()
	close(h.stopChan)
}

// Handle traite un message de trade
func (h *TradeHandler) Handle(ctx context.Context, msg interface{}) error {
	// Type assertion to get the actual message
	handlerMsg, ok := msg.(kafka.HandlerMessage)
	if !ok {
		return fmt.Errorf("invalid message type")
	}

	// Parser le JSON
	var tradeMsg TradeMessage
	if err := json.Unmarshal(handlerMsg.Value, &tradeMsg); err != nil {
		h.logger.WithFields(logrus.Fields{
			"topic":  handlerMsg.Topic,
			"offset": handlerMsg.Offset,
			"error":  err.Error(),
		}).Error("Failed to unmarshal trade message")
		return fmt.Errorf("failed to unmarshal message: %w", err)
	}

	// Valider le type de message
	if tradeMsg.Type != "trade" {
		return fmt.Errorf("unexpected message type: %s (expected 'trade')", tradeMsg.Type)
	}

	// Note: Redis deduplication removed - DB INSERT with ON CONFLICT DO NOTHING handles duplicates

	// Convertir en modèle Trade
	trade := h.toTrade(&tradeMsg)

	// Add to batch buffer
	h.batchMutex.Lock()
	h.batchBuffer = append(h.batchBuffer, trade)
	shouldFlush := len(h.batchBuffer) >= h.batchSize
	h.batchMutex.Unlock()

	// Flush if batch is full
	if shouldFlush {
		h.flushBatch()
	}

	return nil
}

// toTrade convertit un TradeMessage en modèle Trade
func (h *TradeHandler) toTrade(msg *TradeMessage) *models.Trade {
	return &models.Trade{
		EventTs:  time.UnixMilli(msg.EventTs),
		Exchange: msg.Exchange,
		Symbol:   msg.Symbol,
		TradeID:  msg.TradeID,
		Price:    msg.Price,
		Amount:   msg.Amount,
		Side:     msg.Side,
	}
}
