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
	repo   models.IndicatorRepository
	redis  *redis.Client
	logger *logrus.Logger
}

// NewIndicatorHandler crée un nouveau handler pour les indicateurs
func NewIndicatorHandler(cfg *kafka.KafkaConfig, repo models.IndicatorRepository, redisClient *redis.Client, logger *logrus.Logger) *IndicatorHandler {
	topics := cfg.GetIndicatorsTopics()

	return &IndicatorHandler{
		BaseHandler: kafka.BaseHandler{
			Name:   "IndicatorHandler",
			Topics: topics,
		},
		repo:   repo,
		redis:  redisClient,
		logger: logger,
	}
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

	// Vérifier les duplicates (clé: symbol:indicator_type:timeframe:time)
	dedupKey := utils.GenerateCustomDedupKey(
		"indicator",
		indicatorMsg.Symbol,
		indicatorMsg.IndicatorType,
		indicatorMsg.Timeframe,
		fmt.Sprintf("%d", indicatorMsg.Time),
	)

	isDuplicate, err := utils.CheckAndMark(ctx, h.redis, dedupKey, 24*time.Hour)
	if err != nil {
		h.logger.Warnf("Failed to check duplicate: %v", err)
	} else if isDuplicate {
		h.logger.WithFields(logrus.Fields{
			"symbol":         indicatorMsg.Symbol,
			"indicator_type": indicatorMsg.IndicatorType,
			"timeframe":      indicatorMsg.Timeframe,
		}).Debug("Duplicate indicator message, skipping")
		return nil // Pas une erreur, juste un skip
	}

	// Convertir en modèle Indicator
	indicator := h.toIndicator(&indicatorMsg)

	// Insérer dans la base de données
	if err := h.repo.Create(indicator); err != nil {
		return fmt.Errorf("failed to insert indicator: %w", err)
	}

	h.logger.WithFields(logrus.Fields{
		"symbol":         indicator.Symbol,
		"indicator_type": indicator.IndicatorType,
		"timeframe":      indicator.Timeframe,
		"time":           indicator.Time.Format(time.RFC3339),
	}).Debug("Indicator inserted successfully")

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
