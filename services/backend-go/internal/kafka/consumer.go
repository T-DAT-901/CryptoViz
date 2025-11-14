package kafka

import (
	"context"
	"fmt"
	"time"

	"cryptoviz-backend/internal/kafka/utils"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/sirupsen/logrus"
)

// Consumer représente un consommateur Kafka générique
type Consumer struct {
	consumer *kafka.Consumer
	handler  MessageHandler
	logger   *logrus.Logger
	ctx      context.Context
	cancel   context.CancelFunc
}

// NewConsumer crée un nouveau consommateur Kafka avec confluent-kafka-go
func NewConsumer(cfg *KafkaConfig, handler MessageHandler, logger *logrus.Logger) (*Consumer, error) {
	topics := handler.GetTopics()

	// Configuration du consommateur confluent-kafka-go
	config := &kafka.ConfigMap{
		"bootstrap.servers":        cfg.Brokers[0], // Join brokers with comma if multiple
		"group.id":                 cfg.GroupID,
		"auto.offset.reset":        "earliest", // Start from beginning if no offset
		"enable.auto.commit":       false,      // Manual commit for better control
		"session.timeout.ms":       int(cfg.SessionTimeout.Milliseconds()),
		"max.poll.interval.ms":     300000, // 5 minutes
		"go.logs.channel.enable":   true,
		"go.application.rebalance.enable": true,
	}

	consumer, err := kafka.NewConsumer(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create consumer: %w", err)
	}

	// Subscribe to topics
	err = consumer.SubscribeTopics(topics, nil)
	if err != nil {
		consumer.Close()
		return nil, fmt.Errorf("failed to subscribe to topics: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Consumer{
		consumer: consumer,
		handler:  handler,
		logger:   logger,
		ctx:      ctx,
		cancel:   cancel,
	}, nil
}

// Start démarre la consommation des messages
func (c *Consumer) Start() error {
	c.logger.Infof("[%s] Starting Kafka consumer for topics: %v", c.handler.GetName(), c.handler.GetTopics())
	c.logger.Infof("[%s] Consumer started, polling for messages...", c.handler.GetName())

	messageCount := 0
	for {
		select {
		case <-c.ctx.Done():
			c.logger.Infof("[%s] Context canceled, stopping consumer", c.handler.GetName())
			return nil
		default:
			// Poll for messages with 1 second timeout
			ev := c.consumer.Poll(1000)
			if ev == nil {
				continue
			}

			switch e := ev.(type) {
			case *kafka.Message:
				messageCount++
				if messageCount%100 == 1 {
					c.logger.Infof("[%s] Processed %d messages so far", c.handler.GetName(), messageCount)
				}

				c.logger.Infof("[%s] Received message from topic %s[%d]@%d",
					c.handler.GetName(), *e.TopicPartition.Topic, e.TopicPartition.Partition, e.TopicPartition.Offset)

				// Convert confluent-kafka-go message to our format
				msg := ConvertMessage(e)

				// Process message with retry
				if err := c.processMessage(msg); err != nil {
					c.logger.WithFields(logrus.Fields{
						"topic":     *e.TopicPartition.Topic,
						"partition": e.TopicPartition.Partition,
						"offset":    e.TopicPartition.Offset,
						"error":     err.Error(),
					}).Error("Failed to process message")
					continue
				}

				// Commit offset after successful processing
				_, err := c.consumer.CommitMessage(e)
				if err != nil {
					c.logger.Errorf("[%s] Failed to commit message: %v", c.handler.GetName(), err)
				}

			case kafka.Error:
				c.logger.Errorf("[%s] Kafka error: %v", c.handler.GetName(), e)
				if e.Code() == kafka.ErrAllBrokersDown {
					return fmt.Errorf("all brokers down")
				}

			case *kafka.Stats:
				// Consumer statistics (optional, can be logged)
				c.logger.Debugf("[%s] Consumer stats: %s", c.handler.GetName(), e.String())

			default:
				c.logger.Debugf("[%s] Ignored event: %v", c.handler.GetName(), e)
			}
		}
	}
}

// ConvertMessage converts confluent-kafka-go message to our Message format
func ConvertMessage(m *kafka.Message) Message {
	return Message{
		Topic:     *m.TopicPartition.Topic,
		Partition: int(m.TopicPartition.Partition),
		Offset:    int64(m.TopicPartition.Offset),
		Key:       m.Key,
		Value:     m.Value,
		Headers:   convertHeaders(m.Headers),
		Timestamp: m.Timestamp,
	}
}

// convertHeaders converts Kafka headers
func convertHeaders(headers []kafka.Header) []MessageHeader {
	result := make([]MessageHeader, len(headers))
	for i, h := range headers {
		result[i] = MessageHeader{
			Key:   h.Key,
			Value: h.Value,
		}
	}
	return result
}

// Message represents a Kafka message
type Message struct {
	Topic     string
	Partition int
	Offset    int64
	Key       []byte
	Value     []byte
	Headers   []MessageHeader
	Timestamp time.Time
}

// MessageHeader represents a message header
type MessageHeader struct {
	Key   string
	Value []byte
}

// HandlerMessage is the generic message format passed to handlers
type HandlerMessage struct {
	Topic     string
	Partition int
	Offset    int64
	Key       []byte
	Value     []byte
	Headers   []MessageHeader
}

// processMessage traite un message avec retry
func (c *Consumer) processMessage(msg Message) error {
	// Configuration de retry
	retryConfig := utils.DefaultRetryConfig()

	// Traiter avec retry
	err := utils.RetryWithBackoff(c.ctx, retryConfig, func() error {
		// Convert our Message to the handler's expected format
		handlerMsg := HandlerMessage{
			Topic:     msg.Topic,
			Partition: msg.Partition,
			Offset:    msg.Offset,
			Key:       msg.Key,
			Value:     msg.Value,
			Headers:   msg.Headers,
		}
		return c.handler.Handle(c.ctx, handlerMsg)
	})

	if err != nil {
		return fmt.Errorf("failed to handle message after retries: %w", err)
	}

	c.logger.WithFields(logrus.Fields{
		"topic":     msg.Topic,
		"partition": msg.Partition,
		"offset":    msg.Offset,
		"handler":   c.handler.GetName(),
	}).Debug("Message processed successfully")

	return nil
}

// Stop arrête le consommateur proprement
func (c *Consumer) Stop() error {
	c.logger.Infof("[%s] Stopping consumer...", c.handler.GetName())
	c.cancel() // Cancel context

	if err := c.consumer.Close(); err != nil {
		return fmt.Errorf("failed to close consumer: %w", err)
	}

	c.logger.Infof("[%s] Consumer stopped successfully", c.handler.GetName())
	return nil
}

// Stats retourne les statistiques du consommateur
func (c *Consumer) Stats() map[string]interface{} {
	// confluent-kafka-go doesn't have a simple Stats() method
	// We'd need to enable statistics and parse them
	return map[string]interface{}{
		"handler": c.handler.GetName(),
		"topics":  c.handler.GetTopics(),
	}
}
