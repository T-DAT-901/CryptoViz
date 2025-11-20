package kafka

import (
	"context"
	"fmt"
	"sync"

	"github.com/sirupsen/logrus"
)

// ConsumerManager gère le cycle de vie de tous les consommateurs Kafka
type ConsumerManager struct {
	consumers []*Consumer
	handlers  []MessageHandler
	logger    *logrus.Logger
	wg        sync.WaitGroup
	mu        sync.RWMutex
	running   bool
}

// NewConsumerManager crée un nouveau gestionnaire de consommateurs
func NewConsumerManager(logger *logrus.Logger) *ConsumerManager {
	return &ConsumerManager{
		consumers: make([]*Consumer, 0),
		handlers:  make([]MessageHandler, 0),
		logger:    logger,
		running:   false,
	}
}

// RegisterHandler enregistre un handler et crée son consommateur
func (m *ConsumerManager) RegisterHandler(cfg *KafkaConfig, handler MessageHandler) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	consumer, err := NewConsumer(cfg, handler, m.logger)
	if err != nil {
		return fmt.Errorf("failed to create consumer for %s: %w", handler.GetName(), err)
	}

	m.consumers = append(m.consumers, consumer)
	m.handlers = append(m.handlers, handler)

	m.logger.Infof("Registered handler: %s for topics: %v", handler.GetName(), handler.GetTopics())
	return nil
}

// StartAll démarre tous les consommateurs enregistrés
func (m *ConsumerManager) StartAll(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		return fmt.Errorf("consumer manager already running")
	}

	if len(m.consumers) == 0 {
		return fmt.Errorf("no consumers registered")
	}

	m.logger.Infof("Starting %d Kafka consumers...", len(m.consumers))

	for _, consumer := range m.consumers {
		c := consumer // Capture for goroutine
		m.wg.Add(1)
		go func() {
			defer m.wg.Done()
			if err := c.Start(); err != nil {
				m.logger.Errorf("Consumer error: %v", err)
			}
		}()
	}

	m.running = true
	m.logger.Info("All Kafka consumers started successfully")

	return nil
}

// StopAll arrête tous les consommateurs proprement
func (m *ConsumerManager) StopAll() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return nil
	}

	m.logger.Info("Stopping all Kafka consumers...")

	var errors []error
	for _, consumer := range m.consumers {
		if err := consumer.Stop(); err != nil {
			m.logger.Errorf("Failed to stop consumer: %v", err)
			errors = append(errors, err)
		}
	}

	// Attendre que toutes les goroutines se terminent
	m.wg.Wait()

	m.running = false

	if len(errors) > 0 {
		return fmt.Errorf("failed to stop %d consumers", len(errors))
	}

	m.logger.Info("All Kafka consumers stopped successfully")
	return nil
}

// Health retourne le statut de santé de tous les consommateurs
func (m *ConsumerManager) Health() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	health := map[string]interface{}{
		"running":         m.running,
		"total_consumers": len(m.consumers),
		"consumers":       make([]map[string]interface{}, 0),
	}

	for i, consumer := range m.consumers {
		stats := consumer.Stats()
		consumerHealth := map[string]interface{}{
			"name":   m.handlers[i].GetName(),
			"topics": m.handlers[i].GetTopics(),
			"stats":  stats,
		}
		health["consumers"] = append(health["consumers"].([]map[string]interface{}), consumerHealth)
	}

	return health
}

// IsRunning retourne true si le manager est en cours d'exécution
func (m *ConsumerManager) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.running
}
