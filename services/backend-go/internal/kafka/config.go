package kafka

import (
	"strings"
	"time"

	"cryptoviz-backend/internal/config"
)

// KafkaConfig contient la configuration Kafka spécifique
type KafkaConfig struct {
	Brokers          []string
	GroupID          string
	EnableAutoCommit bool
	CommitInterval   time.Duration
	SessionTimeout   time.Duration
	MinBytes         int
	MaxBytes         int
	MaxWait          time.Duration

	// Topics
	TopicRawTrades        string
	TopicAggregatedPrefix string
	TopicIndicatorsPrefix string
	TopicNews             string
}

// NewKafkaConfig crée une configuration Kafka à partir de la configuration globale
func NewKafkaConfig(cfg *config.Config) *KafkaConfig {
	brokers := strings.Split(cfg.KafkaBrokers, ",")

	return &KafkaConfig{
		Brokers:               brokers,
		GroupID:               cfg.KafkaGroupID,
		EnableAutoCommit:      true,
		CommitInterval:        5 * time.Second,
		SessionTimeout:        30 * time.Second,
		MinBytes:              1024,             // 1KB
		MaxBytes:              10 * 1024 * 1024, // 10MB
		MaxWait:               500 * time.Millisecond,
		TopicRawTrades:        cfg.TopicRawTrades,
		TopicAggregatedPrefix: cfg.TopicAggregatedPrefix,
		TopicIndicatorsPrefix: cfg.TopicIndicatorsPrefix,
		TopicNews:             cfg.TopicNews,
	}
}

// GetAggregatedTopics retourne la liste des topics agrégés à consommer
func (k *KafkaConfig) GetAggregatedTopics() []string {
	// Topics: crypto.aggregated.5s, 1m, 15m, 1h, 4h, 1d
	timeframes := []string{"5s", "1m", "15m", "1h", "4h", "1d"}
	topics := make([]string, len(timeframes))
	for i, tf := range timeframes {
		topics[i] = k.TopicAggregatedPrefix + tf
	}
	return topics
}

// GetIndicatorsTopics retourne la liste des topics d'indicateurs à consommer
func (k *KafkaConfig) GetIndicatorsTopics() []string {
	// Topics: crypto.indicators.rsi, macd, bollinger, momentum
	indicators := []string{"rsi", "macd", "bollinger", "momentum"}
	topics := make([]string, len(indicators))
	for i, ind := range indicators {
		topics[i] = k.TopicIndicatorsPrefix + ind
	}
	return topics
}
