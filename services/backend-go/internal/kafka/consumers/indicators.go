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

// IndicatorMessage représente le message JSON des topics crypto.indicators.*
type IndicatorMessage struct {
	Type           string   `json:"type"`
	Time           int64    `json:"time"` // Unix timestamp ms
	Symbol         string   `json:"symbol"`
	Timeframe      string   `json:"timeframe"`
	IndicatorType  string   `json:"indicator_type"`
	Value          *float64 `json:"value,omitempty"`           // For RSI, momentum
	ValueSignal    *float64 `json:"value_signal,omitempty"`    // For MACD
	ValueHistogram *float64 `json:"value_histogram,omitempty"` // For MACD
	UpperBand      *float64 `json:"upper_band,omitempty"`      // For Bollinger
	LowerBand      *float64 `json:"lower_band,omitempty"`      // For Bollinger
	MiddleBand     *float64 `json:"middle_band,omitempty"`     // For Bollinger
}

// IndicatorHandler gère les messages d'indicateurs depuis Kafka
type IndicatorHandler struct {
	kafka.BaseHandler
	repo        models.IndicatorRepository
	logger      *logrus.Logger
	batchBuffer []*models.Indicator
	batchMutex  sync.Mutex
	batchSize   int
	flushTicker *time.Ticker
	stopChan    chan struct{}
}

// NewIndicatorHandler crée un nouveau handler pour les indicateurs
func NewIndicatorHandler(cfg *kafka.KafkaConfig, repo models.IndicatorRepository, logger *logrus.Logger) *IndicatorHandler {
	topics := cfg.GetIndicatorsTopics()

	h := &IndicatorHandler{
		BaseHandler: kafka.BaseHandler{
			Name:   "IndicatorHandler",
			Topics: topics,
		},
		repo:        repo,
		logger:      logger,
		batchBuffer: make([]*models.Indicator, 0, 50),
		batchSize:   50,
		flushTicker: time.NewTicker(500 * time.Millisecond),
		stopChan:    make(chan struct{}),
	}

	go h.flushLoop()
	return h
}

// flushLoop periodically flushes the batch buffer
func (h *IndicatorHandler) flushLoop() {
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

// flushBatch writes all buffered indicators to the database
func (h *IndicatorHandler) flushBatch() {
	h.batchMutex.Lock()
	if len(h.batchBuffer) == 0 {
		h.batchMutex.Unlock()
		return
	}
	batch := h.batchBuffer
	h.batchBuffer = make([]*models.Indicator, 0, h.batchSize)
	h.batchMutex.Unlock()

	if err := h.repo.CreateBatch(batch); err != nil {
		h.logger.Errorf("Failed to flush indicator batch (%d items): %v", len(batch), err)
		return
	}

	h.logger.Debugf("Flushed %d indicators to database", len(batch))
}

// Stop stops the flush goroutine
func (h *IndicatorHandler) Stop() {
	h.flushTicker.Stop()
	close(h.stopChan)
}

// Handle traite un message d'indicateur
func (h *IndicatorHandler) Handle(ctx context.Context, msg interface{}) error {
	// Type assertion to get the actual message
	handlerMsg, ok := msg.(kafka.HandlerMessage)
	if !ok {
		return fmt.Errorf("invalid message type")
	}

	// Parser le JSON
	var indicatorMsg IndicatorMessage
	if err := json.Unmarshal(handlerMsg.Value, &indicatorMsg); err != nil {
		h.logger.WithFields(logrus.Fields{
			"topic":  handlerMsg.Topic,
			"offset": handlerMsg.Offset,
			"error":  err.Error(),
		}).Error("Failed to unmarshal indicator message")
		return fmt.Errorf("failed to unmarshal message: %w", err)
	}

	// Valider le type de message
	if indicatorMsg.Type != "indicator" {
		return fmt.Errorf("unexpected message type: %s (expected 'indicator')", indicatorMsg.Type)
	}

	// Note: Redis deduplication removed - DB UPSERT with ON CONFLICT handles duplicates

	// Convertir en modèle Indicator
	indicator := h.toIndicator(&indicatorMsg)

	// Add to batch buffer
	h.batchMutex.Lock()
	h.batchBuffer = append(h.batchBuffer, indicator)
	shouldFlush := len(h.batchBuffer) >= h.batchSize
	h.batchMutex.Unlock()

	// Flush if batch is full
	if shouldFlush {
		h.flushBatch()
	}

	return nil
}

// toIndicator convertit un IndicatorMessage en modèle Indicator
func (h *IndicatorHandler) toIndicator(msg *IndicatorMessage) *models.Indicator {
	return &models.Indicator{
		Time:           time.UnixMilli(msg.Time),
		Symbol:         msg.Symbol,
		Timeframe:      msg.Timeframe,
		IndicatorType:  msg.IndicatorType,
		Value:          msg.Value,
		ValueSignal:    msg.ValueSignal,
		ValueHistogram: msg.ValueHistogram,
		UpperBand:      msg.UpperBand,
		LowerBand:      msg.LowerBand,
		MiddleBand:     msg.MiddleBand,
	}
}
