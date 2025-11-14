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
}

// Load charge la configuration depuis les variables d'environnement
func Load() *Config {
	return &Config{
		Port:         getEnv("PORT", "8080"),
		RedisHost:    getEnv("REDIS_HOST", "redis"),
		RedisPort:    getEnv("REDIS_PORT", "6379"),
		KafkaBrokers: getEnv("KAFKA_BROKERS", "kafka:29092"),
		GinMode:      getEnv("GIN_MODE", "debug"),
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
