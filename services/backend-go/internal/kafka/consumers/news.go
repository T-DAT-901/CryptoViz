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

// NewsMessage représente le message JSON du topic crypto.news
type NewsMessage struct {
	Type           string   `json:"type"`
	Time           int64    `json:"time"` // Unix timestamp ms
	Source         string   `json:"source"`
	URL            string   `json:"url"`
	Title          string   `json:"title"`
	Content        string   `json:"content"`
	SentimentScore *float64 `json:"sentiment_score,omitempty"`
	Symbols        []string `json:"symbols"` // Array of symbols
}

// NewsHandler gère les messages d'actualités depuis Kafka
type NewsHandler struct {
	kafka.BaseHandler
	repo      models.NewsRepository
	redis     *redis.Client
	logger    *logrus.Logger
	broadcast BroadcastFunc
}

// NewNewsHandler crée un nouveau handler pour les actualités
func NewNewsHandler(cfg *kafka.KafkaConfig, repo models.NewsRepository, redisClient *redis.Client, logger *logrus.Logger) *NewsHandler {
	return &NewsHandler{
		BaseHandler: kafka.BaseHandler{
			Name:   "NewsHandler",
			Topics: []string{cfg.TopicNews},
		},
		repo:   repo,
		redis:  redisClient,
		logger: logger,
	}
}

// SetBroadcast sets the broadcast callback for WebSocket streaming
func (h *NewsHandler) SetBroadcast(fn BroadcastFunc) {
	h.broadcast = fn
}

// Handle traite un message d'actualité
func (h *NewsHandler) Handle(ctx context.Context, msg interface{}) error {
	// Type assertion to get the actual message
	handlerMsg, ok := msg.(kafka.HandlerMessage)
	if !ok {
		return fmt.Errorf("invalid message type")
	}

	// Parser le JSON
	var newsMsg NewsMessage
	if err := json.Unmarshal(handlerMsg.Value, &newsMsg); err != nil {
		h.logger.WithFields(logrus.Fields{
			"topic":  handlerMsg.Topic,
			"offset": handlerMsg.Offset,
			"error":  err.Error(),
		}).Error("Failed to unmarshal news message")
		return fmt.Errorf("failed to unmarshal message: %w", err)
	}

	// Valider le type de message
	if newsMsg.Type != "news" {
		return fmt.Errorf("unexpected message type: %s (expected 'news')", newsMsg.Type)
	}

	// Vérifier les duplicates (clé composite: time:source:url)
	dedupKey := utils.GenerateCustomDedupKey(
		"news",
		fmt.Sprintf("%d", newsMsg.Time),
		newsMsg.Source,
		newsMsg.URL,
	)

	isDuplicate, err := utils.CheckAndMark(ctx, h.redis, dedupKey, 48*time.Hour)
	if err != nil {
		h.logger.Warnf("Failed to check duplicate: %v", err)
	} else if isDuplicate {
		h.logger.WithFields(logrus.Fields{
			"source": newsMsg.Source,
			"url":    newsMsg.URL,
		}).Debug("Duplicate news message, skipping")
		return nil // Pas une erreur, juste un skip
	}

	// Convertir en modèle News
	news := h.toNews(&newsMsg)

	// Insérer dans la base de données
	if err := h.repo.Create(news); err != nil {
		return fmt.Errorf("failed to insert news: %w", err)
	}

	h.logger.WithFields(logrus.Fields{
		"source":  news.Source,
		"title":   news.Title,
		"symbols": news.Symbols,
		"time":    news.Time.Format(time.RFC3339),
	}).Debug("News inserted successfully")

	// Broadcast to WebSocket clients if callback is set
	if h.broadcast != nil {
		// Prepare WebSocket data
		wsData := map[string]interface{}{
			"time":            news.Time.UnixMilli(),
			"source":          news.Source,
			"url":             news.URL,
			"title":           news.Title,
			"content":         news.Content,
			"sentiment_score": news.SentimentScore,
			"symbols":         news.Symbols,
		}

		// Broadcast to each symbol the news mentions
		for _, symbol := range newsMsg.Symbols {
			h.broadcast("news", symbol, "", wsData)
		}
	}

	return nil
}

// toNews convertit un NewsMessage en modèle News
func (h *NewsHandler) toNews(msg *NewsMessage) *models.News {
	return &models.News{
		Time:           time.UnixMilli(msg.Time),
		Source:         msg.Source,
		URL:            msg.URL,
		Title:          msg.Title,
		Content:        msg.Content,
		SentimentScore: msg.SentimentScore,
		Symbols:        models.StringArray(msg.Symbols),
	}
}
