package utils

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

const (
	// DeduplicationKeyPrefix est le préfixe des clés Redis pour la déduplication
	DeduplicationKeyPrefix = "kafka:dedup:"
)

// IsDuplicate vérifie si un message a déjà été traité
func IsDuplicate(ctx context.Context, redisClient *redis.Client, key string, ttl time.Duration) (bool, error) {
	fullKey := DeduplicationKeyPrefix + key

	// Vérifier si la clé existe
	exists, err := redisClient.Exists(ctx, fullKey).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check duplicate: %w", err)
	}

	return exists > 0, nil
}

// MarkProcessed marque un message comme traité dans Redis
func MarkProcessed(ctx context.Context, redisClient *redis.Client, key string, ttl time.Duration) error {
	fullKey := DeduplicationKeyPrefix + key

	// Définir la clé avec un TTL
	err := redisClient.Set(ctx, fullKey, "1", ttl).Err()
	if err != nil {
		return fmt.Errorf("failed to mark as processed: %w", err)
	}

	return nil
}

// GenerateDedupKey génère une clé de déduplication unique pour un message
// Format: topic:partition:offset ou custom key
func GenerateDedupKey(topic string, partition int, offset int64) string {
	return fmt.Sprintf("%s:%d:%d", topic, partition, offset)
}

// GenerateCustomDedupKey génère une clé de déduplication personnalisée
// Utilisé pour les messages avec un identifiant unique (ex: trade_id)
func GenerateCustomDedupKey(prefix string, values ...string) string {
	key := prefix
	for _, value := range values {
		key += ":" + value
	}
	return key
}

// CheckAndMark vérifie et marque un message en une seule opération atomique
func CheckAndMark(ctx context.Context, redisClient *redis.Client, key string, ttl time.Duration) (bool, error) {
	fullKey := DeduplicationKeyPrefix + key

	// Utiliser SETNX pour une opération atomique (set if not exists)
	success, err := redisClient.SetNX(ctx, fullKey, "1", ttl).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check and mark: %w", err)
	}

	// Si success = true, le message n'était pas un duplicate (nouvelle clé créée)
	// Si success = false, le message est un duplicate (clé existait déjà)
	return !success, nil
}
