package consumers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"cryptoviz-backend/internal/kafka"
	"cryptoviz-backend/internal/kafka/utils"
	"cryptoviz-backend/models"

	"github.com/go-redis/redis/v8"
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
	repo      models.CandleRepository
	redis     *redis.Client
	logger    *logrus.Logger
	broadcast BroadcastFunc
}

// NewCandleHandler crée un nouveau handler pour les candles
func NewCandleHandler(cfg *kafka.KafkaConfig, repo models.CandleRepository, redisClient *redis.Client, logger *logrus.Logger) *CandleHandler {
	topics := cfg.GetAggregatedTopics()

	return &CandleHandler{
		BaseHandler: kafka.BaseHandler{
			Name:   "CandleHandler",
			Topics: topics,
		},
		repo:   repo,
		redis:  redisClient,
		logger: logger,
	}
}

// SetBroadcast sets the broadcast callback for WebSocket streaming
func (h *CandleHandler) SetBroadcast(fn BroadcastFunc) {
	h.broadcast = fn
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

	// Vérifier les duplicates (clé: exchange:symbol:timeframe:window_start)
	dedupKey := utils.GenerateCustomDedupKey(
		"candle",
		candleMsg.Exchange,
		candleMsg.Symbol,
		candleMsg.Timeframe,
		fmt.Sprintf("%d", candleMsg.WindowStart),
	)

	isDuplicate, err := utils.CheckAndMark(ctx, h.redis, dedupKey, 24*time.Hour)
	if err != nil {
		h.logger.Warnf("Failed to check duplicate: %v", err)
	} else if isDuplicate {
		h.logger.WithFields(logrus.Fields{
			"exchange":  candleMsg.Exchange,
			"symbol":    candleMsg.Symbol,
			"timeframe": candleMsg.Timeframe,
		}).Debug("Duplicate candle message, skipping")
		return nil // Pas une erreur, juste un skip
	}

	// Convertir en modèle Candle
	candle := h.toCandle(&candleMsg, metadata.Source)

	// Insérer dans la base de données
	if err := h.repo.Create(candle); err != nil {
		return fmt.Errorf("failed to insert candle: %w", err)
	}

	// Broadcast to WebSocket clients (only closed candles to reduce noise)
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
		h.broadcast("candle", candle.Symbol, candle.Timeframe, wsData)
	}

	h.logger.WithFields(logrus.Fields{
		"exchange":  candle.Exchange,
		"symbol":    candle.Symbol,
		"timeframe": candle.Timeframe,
		"window":    candle.WindowStart.Format(time.RFC3339),
		"closed":    candle.Closed,
	}).Debug("Candle inserted successfully")

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
