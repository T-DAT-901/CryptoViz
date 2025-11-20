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

// TradeHandler gère les messages de trades depuis Kafka
type TradeHandler struct {
	kafka.BaseHandler
	repo   models.TradeRepository
	redis  *redis.Client
	logger *logrus.Logger
}

// NewTradeHandler crée un nouveau handler pour les trades
func NewTradeHandler(cfg *kafka.KafkaConfig, repo models.TradeRepository, redisClient *redis.Client, logger *logrus.Logger) *TradeHandler {
	return &TradeHandler{
		BaseHandler: kafka.BaseHandler{
			Name:   "TradeHandler",
			Topics: []string{cfg.TopicRawTrades},
		},
		repo:   repo,
		redis:  redisClient,
		logger: logger,
	}
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

	// Vérifier les duplicates (clé: exchange:symbol:trade_id)
	dedupKey := utils.GenerateCustomDedupKey(
		"trade",
		tradeMsg.Exchange,
		tradeMsg.Symbol,
		tradeMsg.TradeID,
	)

	isDuplicate, err := utils.CheckAndMark(ctx, h.redis, dedupKey, 1*time.Hour)
	if err != nil {
		h.logger.Warnf("Failed to check duplicate: %v", err)
	} else if isDuplicate {
		h.logger.WithFields(logrus.Fields{
			"exchange": tradeMsg.Exchange,
			"symbol":   tradeMsg.Symbol,
			"trade_id": tradeMsg.TradeID,
		}).Debug("Duplicate trade message, skipping")
		return nil // Pas une erreur, juste un skip
	}

	// Convertir en modèle Trade
	trade := h.toTrade(&tradeMsg)

	// Insérer dans la base de données
	if err := h.repo.Create(trade); err != nil {
		return fmt.Errorf("failed to insert trade: %w", err)
	}

	h.logger.WithFields(logrus.Fields{
		"exchange": trade.Exchange,
		"symbol":   trade.Symbol,
		"trade_id": trade.TradeID,
		"price":    trade.Price,
		"side":     trade.Side,
	}).Debug("Trade inserted successfully")

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
