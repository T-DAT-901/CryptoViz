package utils

import (
	"context"
	"fmt"
	"time"
)

// RetryConfig contient la configuration pour les retries
type RetryConfig struct {
	MaxRetries     int
	InitialBackoff time.Duration
	MaxBackoff     time.Duration
	BackoffFactor  float64
}

// DefaultRetryConfig retourne une configuration de retry par défaut
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxRetries:     3,
		InitialBackoff: 100 * time.Millisecond,
		MaxBackoff:     10 * time.Second,
		BackoffFactor:  2.0,
	}
}

// RetryWithBackoff exécute une fonction avec un backoff exponentiel
func RetryWithBackoff(ctx context.Context, cfg *RetryConfig, fn func() error) error {
	var lastErr error
	backoff := cfg.InitialBackoff

	for attempt := 0; attempt <= cfg.MaxRetries; attempt++ {
		// Première tentative sans délai
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return fmt.Errorf("context cancelled: %w", ctx.Err())
			case <-time.After(backoff):
			}

			// Augmenter le backoff pour la prochaine tentative
			backoff = time.Duration(float64(backoff) * cfg.BackoffFactor)
			if backoff > cfg.MaxBackoff {
				backoff = cfg.MaxBackoff
			}
		}

		// Exécuter la fonction
		err := fn()
		if err == nil {
			return nil // Succès
		}

		lastErr = err

		// Si c'est la dernière tentative, ne pas continuer
		if attempt == cfg.MaxRetries {
			break
		}
	}

	return fmt.Errorf("failed after %d attempts: %w", cfg.MaxRetries+1, lastErr)
}

// IsTransientError détermine si une erreur est transitoire et mérite un retry
func IsTransientError(err error) bool {
	if err == nil {
		return false
	}

	// TODO: Implémenter la logique de détection d'erreurs transitoires
	// Par exemple: erreurs réseau, timeouts, DB connection errors, etc.
	// Pour l'instant, on considère toutes les erreurs comme transitoires
	return true
}

// ShouldRetry détermine si on doit réessayer en fonction de l'erreur
func ShouldRetry(err error, attempt int, maxAttempts int) bool {
	if err == nil {
		return false
	}

	if attempt >= maxAttempts {
		return false
	}

	return IsTransientError(err)
}
