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
		// Routes candles
		v1.GET("/crypto/:symbol/data", candleCtrl.GetCandleData)
		v1.GET("/crypto/:symbol/latest", candleCtrl.GetLatestPrice)
		v1.GET("/stats/:symbol", candleCtrl.GetStats)

		// Routes indicateurs
		v1.GET("/indicators/:symbol/:type", indicatorCtrl.GetByType)
		v1.GET("/indicators/:symbol", indicatorCtrl.GetAll)

		// Routes news
		v1.GET("/news", newsCtrl.GetAll)
		v1.GET("/news/:symbol", newsCtrl.GetBySymbol)
	}

	// WebSocket
	router.GET("/ws/crypto", wsCtrl.Handle)

	return router
}
