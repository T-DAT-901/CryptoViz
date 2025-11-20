package routes

import (
	"cryptoviz-backend/internal/controllers"
	"cryptoviz-backend/internal/middleware"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// Setup configure toutes les routes de l'application
func Setup(deps *controllers.Dependencies, logger *logrus.Logger) *gin.Engine {
	router := gin.New()

	// Middleware globaux
	router.Use(gin.Recovery())
	router.Use(middleware.CORS())
	router.Use(middleware.Logger(logger))

	// Contrôleurs
	healthCtrl := controllers.NewHealthController(deps)
	candleCtrl := controllers.NewCandleController(deps)
	indicatorCtrl := controllers.NewIndicatorController(deps)
	newsCtrl := controllers.NewNewsController(deps)
	wsCtrl := controllers.NewWebSocketController(deps)

	// Routes de santé
	router.GET("/health", healthCtrl.HealthCheck)
	router.GET("/ready", healthCtrl.ReadinessCheck)

	// API v1
	v1 := router.Group("/api/v1")
	{
		// Routes candles (using query parameters for symbols with slashes)
		v1.GET("/crypto/data", middleware.ValidateSymbolQuery(), candleCtrl.GetCandleData)
		v1.GET("/crypto/latest", middleware.ValidateSymbolQuery(), candleCtrl.GetLatestPrice)
		v1.GET("/stats", middleware.ValidateSymbolQuery(), candleCtrl.GetStats)

		// Routes indicateurs (using query parameters for symbols with slashes)
		v1.GET("/indicators/:type", middleware.ValidateSymbolQuery(), indicatorCtrl.GetByType)
		v1.GET("/indicators", middleware.ValidateSymbolQuery(), indicatorCtrl.GetAll)

		// Routes news (symbols are single tokens like BTC, ETH, not pairs)
		v1.GET("/news", newsCtrl.GetAll)
		v1.GET("/news/:symbol", newsCtrl.GetBySymbol)
	}

	// WebSocket
	router.GET("/ws/crypto", wsCtrl.Handle)
	router.GET("/ws/stats", wsCtrl.Stats)

	return router
}
