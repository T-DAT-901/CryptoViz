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
	repo        models.NewsRepository
	logger      *logrus.Logger
	broadcast   BroadcastFunc
	batchBuffer []*models.News
	batchMutex  sync.Mutex
	batchSize   int
	flushTicker *time.Ticker
	stopChan    chan struct{}
}

// NewNewsHandler crée un nouveau handler pour les actualités
func NewNewsHandler(cfg *kafka.KafkaConfig, repo models.NewsRepository, logger *logrus.Logger) *NewsHandler {
	h := &NewsHandler{
		BaseHandler: kafka.BaseHandler{
			Name:   "NewsHandler",
			Topics: []string{cfg.TopicNews},
		},
		repo:        repo,
		logger:      logger,
		batchBuffer: make([]*models.News, 0, 20),
		batchSize:   20,
		flushTicker: time.NewTicker(1 * time.Second),
		stopChan:    make(chan struct{}),
	}

	go h.flushLoop()
	return h
}

// SetBroadcast sets the broadcast callback for WebSocket streaming
func (h *NewsHandler) SetBroadcast(fn BroadcastFunc) {
	h.broadcast = fn
}

// flushLoop periodically flushes the batch buffer
func (h *NewsHandler) flushLoop() {
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

// flushBatch writes all buffered news to the database
func (h *NewsHandler) flushBatch() {
	h.batchMutex.Lock()
	if len(h.batchBuffer) == 0 {
		h.batchMutex.Unlock()
		return
	}
	batch := h.batchBuffer
	h.batchBuffer = make([]*models.News, 0, h.batchSize)
	h.batchMutex.Unlock()

	if err := h.repo.CreateBatch(batch); err != nil {
		h.logger.Errorf("Failed to flush news batch (%d items): %v", len(batch), err)
		return
	}

	h.logger.Debugf("Flushed %d news to database", len(batch))

	// Broadcast news
	for _, news := range batch {
		if h.broadcast != nil {
			wsData := map[string]interface{}{
				"time":            news.Time.UnixMilli(),
				"source":          news.Source,
				"url":             news.URL,
				"title":           news.Title,
				"content":         news.Content,
				"sentiment_score": news.SentimentScore,
				"symbols":         news.Symbols,
			}
			for _, symbol := range news.Symbols {
				go h.broadcast("news", symbol, "", wsData)
			}
		}
	}
}

// Stop stops the flush goroutine
func (h *NewsHandler) Stop() {
	h.flushTicker.Stop()
	close(h.stopChan)
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

	// Note: Redis deduplication removed - DB UPSERT with ON CONFLICT handles duplicates

	// Convertir en modèle News
	news := h.toNews(&newsMsg)

	// Add to batch buffer
	h.batchMutex.Lock()
	h.batchBuffer = append(h.batchBuffer, news)
	shouldFlush := len(h.batchBuffer) >= h.batchSize
	h.batchMutex.Unlock()

	// Flush if batch is full
	if shouldFlush {
		h.flushBatch()
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
