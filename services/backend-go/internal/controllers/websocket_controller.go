package controllers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

// WebSocketController gère les connexions WebSocket
type WebSocketController struct {
	upgrader websocket.Upgrader
	logger   *logrus.Logger
}

// NewWebSocketController crée un nouveau WebSocketController
func NewWebSocketController(deps *Dependencies) *WebSocketController {
	return &WebSocketController{
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // En production, vérifier l'origine
			},
		},
		logger: deps.Logger,
	}
}

// Handle gère les connexions WebSocket
// @Summary WebSocket endpoint
// @Description Endpoint WebSocket pour les données en temps réel
// @Tags websocket
// @Router /ws/crypto [get]
func (ctrl *WebSocketController) Handle(c *gin.Context) {
	conn, err := ctrl.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		ctrl.logger.Error("Erreur upgrade WebSocket: ", err)
		return
	}
	defer conn.Close()

	ctrl.logger.Info("Nouvelle connexion WebSocket")

	// Boucle de lecture des messages
	for {
		var msg map[string]interface{}
		err := conn.ReadJSON(&msg)
		if err != nil {
			ctrl.logger.Error("Erreur lecture WebSocket: ", err)
			break
		}

		// Traiter le message (subscribe/unsubscribe)
		if action, ok := msg["action"].(string); ok {
			switch action {
			case "subscribe":
				if symbol, ok := msg["symbol"].(string); ok {
					ctrl.logger.Infof("Subscription à %s", symbol)
					// Ici, on pourrait implémenter la logique de subscription
				}
			case "unsubscribe":
				if symbol, ok := msg["symbol"].(string); ok {
					ctrl.logger.Infof("Unsubscription de %s", symbol)
				}
			}
		}

		// Echo pour test
		response := map[string]interface{}{
			"type":      "response",
			"message":   "Message reçu",
			"timestamp": time.Now(),
		}

		if err := conn.WriteJSON(response); err != nil {
			ctrl.logger.Error("Erreur écriture WebSocket: ", err)
			break
		}
	}
}
