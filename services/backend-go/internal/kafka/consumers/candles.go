package consumers

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"cryptoviz-backend/internal/kafka"
	"cryptoviz-backend/internal/kafka/utils"
	"cryptoviz-backend/models"

	"github.com/sirupsen/logrus"
)

// CandleMessage représente le message JSON du topic crypto.aggregated.*
type CandleMessage struct {
	Type         string  `json:"type"`
	Exchange     string  `json:"exchange"`
	Symbol       string  `json:"symbol"`
	Timeframe    string  `json:"timeframe"`
	WindowStart  int64   `json:"window_start"`  // Unix timestamp ms
	WindowEnd    int64   `json:"window_end"`    // Unix timestamp ms
	Open         float64 `json:"open"`
	High         float64 `json:"high"`
	Low          float64 `json:"low"`
	Close        float64 `json:"close"`
	Volume       float64 `json:"volume"`
	TradeCount   *int    `json:"trade_count,omitempty"`
	Closed       bool    `json:"closed"`
	FirstTradeTs *int64  `json:"first_trade_ts,omitempty"` // Unix timestamp ms
	LastTradeTs  *int64  `json:"last_trade_ts,omitempty"`  // Unix timestamp ms
	DurationMs   *int64  `json:"duration_ms,omitempty"`
}

// CandleHandler gère les messages de candles (OHLCV) depuis Kafka
type CandleHandler struct {
	kafka.BaseHandler
	repo        models.CandleRepository
	logger      *logrus.Logger
	broadcast   BroadcastFunc
	batchBuffer []*models.Candle
	batchMutex  sync.Mutex
	batchSize   int
	flushTicker *time.Ticker
	stopChan    chan struct{}
}

// NewCandleHandler crée un nouveau handler pour les candles
func NewCandleHandler(cfg *kafka.KafkaConfig, repo models.CandleRepository, logger *logrus.Logger) *CandleHandler {
	topics := cfg.GetAggregatedTopics()

	h := &CandleHandler{
		BaseHandler: kafka.BaseHandler{
			Name:   "CandleHandler",
			Topics: topics,
		},
		repo:        repo,
		logger:      logger,
		batchBuffer: make([]*models.Candle, 0, 100),
		batchSize:   100,
		flushTicker: time.NewTicker(250 * time.Millisecond),
		stopChan:    make(chan struct{}),
	}

	// Start periodic flush goroutine
	go h.flushLoop()

	return h
}

// SetBroadcast sets the broadcast callback for WebSocket streaming
func (h *CandleHandler) SetBroadcast(fn BroadcastFunc) {
	h.broadcast = fn
}

// flushLoop periodically flushes the batch buffer
func (h *CandleHandler) flushLoop() {
	for {
		select {
		case <-h.stopChan:
			h.flushBatch() // Final flush on shutdown
			return
		case <-h.flushTicker.C:
			h.flushBatch()
		}
	}
}

// flushBatch writes all buffered candles to the database
func (h *CandleHandler) flushBatch() {
	h.batchMutex.Lock()
	if len(h.batchBuffer) == 0 {
		h.batchMutex.Unlock()
		return
	}
	batch := h.batchBuffer
	h.batchBuffer = make([]*models.Candle, 0, h.batchSize)
	h.batchMutex.Unlock()

	if err := h.repo.CreateBatch(batch); err != nil {
		h.logger.Errorf("Failed to flush candle batch (%d items): %v", len(batch), err)
		return
	}

	h.logger.Debugf("Flushed %d candles to database", len(batch))

	// Broadcast closed candles
	for _, candle := range batch {
		if h.broadcast != nil && candle.Closed {
			wsData := map[string]interface{}{
				"type":         "candle",
				"exchange":     candle.Exchange,
				"symbol":       candle.Symbol,
				"timeframe":    candle.Timeframe,
				"window_start": candle.WindowStart.UnixMilli(),
				"window_end":   candle.WindowEnd.UnixMilli(),
				"open":         candle.Open,
				"high":         candle.High,
				"low":          candle.Low,
				"close":        candle.Close,
				"volume":       candle.Volume,
				"closed":       candle.Closed,
			}
			go h.broadcast("candle", candle.Symbol, candle.Timeframe, wsData)
		}
	}
}

// Stop stops the flush goroutine
func (h *CandleHandler) Stop() {
	h.flushTicker.Stop()
	close(h.stopChan)
}

// Handle traite un message de candle
func (h *CandleHandler) Handle(ctx context.Context, msg interface{}) error {
	// Type assertion to get the actual message
	handlerMsg, ok := msg.(kafka.HandlerMessage)
	if !ok {
		return fmt.Errorf("invalid message type")
	}

	// Convert headers from kafka.MessageHeader to utils.MessageHeader
	utilsHeaders := make([]utils.MessageHeader, len(handlerMsg.Headers))
	for i, h := range handlerMsg.Headers {
		utilsHeaders[i] = utils.MessageHeader{Key: h.Key, Value: h.Value}
	}

	// Parser les headers
	metadata, err := utils.ParseHeaders(utilsHeaders)
	if err != nil {
		return fmt.Errorf("failed to parse headers: %w", err)
	}

	// Parser le JSON
	var candleMsg CandleMessage
	if err := json.Unmarshal(handlerMsg.Value, &candleMsg); err != nil {
		h.logger.WithFields(logrus.Fields{
			"topic":  handlerMsg.Topic,
			"offset": handlerMsg.Offset,
			"error":  err.Error(),
		}).Error("Failed to unmarshal candle message")
		return fmt.Errorf("failed to unmarshal message: %w", err)
	}

	// Valider le type de message
	if candleMsg.Type != "bar" {
		return fmt.Errorf("unexpected message type: %s (expected 'bar')", candleMsg.Type)
	}

	// Note: Redis deduplication removed - DB UPSERT with ON CONFLICT handles duplicates

	// Convertir en modèle Candle
	candle := h.toCandle(&candleMsg, metadata.Source)

	// Add to batch buffer
	h.batchMutex.Lock()
	h.batchBuffer = append(h.batchBuffer, candle)
	shouldFlush := len(h.batchBuffer) >= h.batchSize
	h.batchMutex.Unlock()

	// Flush if batch is full
	if shouldFlush {
		h.flushBatch()
	}

	return nil
}

// toCandle convertit un CandleMessage en modèle Candle
func (h *CandleHandler) toCandle(msg *CandleMessage, source string) *models.Candle {
	windowStart := time.UnixMilli(msg.WindowStart)
	windowEnd := time.UnixMilli(msg.WindowEnd)

	candle := &models.Candle{
		WindowStart: windowStart,
		WindowEnd:   windowEnd,
		Exchange:    msg.Exchange,
		Symbol:      msg.Symbol,
		Timeframe:   msg.Timeframe,
		Open:        msg.Open,
		High:        msg.High,
		Low:         msg.Low,
		Close:       msg.Close,
		Volume:      msg.Volume,
		TradeCount:  msg.TradeCount,
		Closed:      msg.Closed,
		DurationMs:  msg.DurationMs,
	}

	// Champs optionnels
	if msg.FirstTradeTs != nil {
		t := time.UnixMilli(*msg.FirstTradeTs)
		candle.FirstTradeTs = &t
	}
	if msg.LastTradeTs != nil {
		t := time.UnixMilli(*msg.LastTradeTs)
		candle.LastTradeTs = &t
	}
	if source != "" {
		candle.Source = &source
	}

	return candle
}
