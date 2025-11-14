package config

import (
	"fmt"
	"os"
)

// Config contient toute la configuration de l'application
type Config struct {
	Port         string
	RedisHost    string
	RedisPort    string
	KafkaBrokers string
	GinMode      string

	// Kafka configuration
	KafkaGroupID           string
	TopicRawTrades         string
	TopicAggregatedPrefix  string
	TopicIndicatorsPrefix  string
	TopicNews              string
}

// Load charge la configuration depuis les variables d'environnement
func Load() *Config {
	return &Config{
		Port:         getEnv("PORT", "8080"),
		RedisHost:    getEnv("REDIS_HOST", "redis"),
		RedisPort:    getEnv("REDIS_PORT", "6379"),
		KafkaBrokers: getEnv("KAFKA_BROKERS", "kafka:29092"),
		GinMode:      getEnv("GIN_MODE", "debug"),

		// Kafka topics
		KafkaGroupID:          getEnv("KAFKA_GROUP_ID", "backend-go-consumers"),
		TopicRawTrades:        getEnv("TOPIC_RAW_TRADES", "crypto.raw.trades"),
		TopicAggregatedPrefix: getEnv("TOPIC_AGGREGATED_PREFIX", "crypto.aggregated."),
		TopicIndicatorsPrefix: getEnv("TOPIC_INDICATORS_PREFIX", "crypto.indicators."),
		TopicNews:             getEnv("TOPIC_NEWS", "crypto.news"),
	}
}

// RedisURL retourne l'URL complète de Redis
func (c *Config) RedisURL() string {
	return fmt.Sprintf("%s:%s", c.RedisHost, c.RedisPort)
}

// getEnv récupère une variable d'environnement avec une valeur par défaut
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
