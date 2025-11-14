package kafka

import (
	"context"
)

// MessageHandler définit l'interface pour traiter les messages Kafka
type MessageHandler interface {
	// Handle traite un message Kafka (generic message structure)
	Handle(ctx context.Context, msg interface{}) error

	// GetTopics retourne les topics que ce handler gère
	GetTopics() []string

	// GetName retourne le nom du handler pour le logging
	GetName() string
}

// BaseHandler contient les champs communs à tous les handlers
type BaseHandler struct {
	Name   string
	Topics []string
}

// GetTopics implémente MessageHandler.GetTopics
func (h *BaseHandler) GetTopics() []string {
	return h.Topics
}

// GetName implémente MessageHandler.GetName
func (h *BaseHandler) GetName() string {
	return h.Name
}
