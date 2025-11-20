package controllers

import (
	"net/http"

	ws "cryptoviz-backend/internal/websocket"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

// WebSocketController gère les connexions WebSocket
type WebSocketController struct {
	upgrader websocket.Upgrader
	logger   *logrus.Logger
	hub      *ws.Hub
}

// NewWebSocketController crée un nouveau WebSocketController
func NewWebSocketController(deps *Dependencies) *WebSocketController {
	return &WebSocketController{
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // En production, vérifier l'origine
			},
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
		logger: deps.Logger,
		hub:    deps.WSHub,
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

	// Register client with Hub
	client := ctrl.hub.RegisterClient(conn)

	ctrl.logger.WithField("client_id", client.GetSubscriptions()).Info("Nouvelle connexion WebSocket")

	// Start client pumps
	go client.WritePump()
	go client.ReadPump()
}

// Stats retourne les statistiques du Hub WebSocket
// @Summary WebSocket stats
// @Description Statistiques des connexions WebSocket
// @Tags websocket
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /ws/stats [get]
func (ctrl *WebSocketController) Stats(c *gin.Context) {
	stats := ctrl.hub.GetStats()
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"data":   stats,
	})
}
