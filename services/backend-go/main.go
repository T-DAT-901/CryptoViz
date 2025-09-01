package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"

	"cryptoviz-backend/database"
	"cryptoviz-backend/models"
)

// Configuration de l'application
type Config struct {
	Port         string
	RedisURL     string
	KafkaBrokers string
}

// Structure de réponse API
type APIResponse struct {
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,omitempty"`
	Error     string      `json:"error,omitempty"`
	Message   string      `json:"message,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

// Application principale avec GORM
type App struct {
	Config                 *Config
	Redis                  *redis.Client
	Router                 *gin.Engine
	Logger                 *logrus.Logger
	CryptoDataRepo         models.CryptoDataRepository
	TechnicalIndicatorRepo models.TechnicalIndicatorRepository
	CryptoNewsRepo         models.CryptoNewsRepository
	upgrader               websocket.Upgrader
}

// Initialisation de la configuration
func NewConfig() *Config {
	return &Config{
		Port:         getEnv("PORT", "8080"),
		RedisURL:     buildRedisURL(),
		KafkaBrokers: getEnv("KAFKA_BROKERS", "kafka:29092"),
	}
}

// Construction de l'URL Redis
func buildRedisURL() string {
	host := getEnv("REDIS_HOST", "redis")
	port := getEnv("REDIS_PORT", "6379")
	return fmt.Sprintf("%s:%s", host, port)
}

// Utilitaire pour les variables d'environnement
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Initialisation de l'application
func NewApp() *App {
	config := NewConfig()

	// Configuration du logger
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetLevel(logrus.InfoLevel)

	// Configuration de Gin
	if os.Getenv("GIN_MODE") == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	app := &App{
		Config: config,
		Logger: logger,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // En production, vérifier l'origine
			},
		},
	}

	return app
}

// Connexion à la base de données avec GORM
func (app *App) ConnectDatabase() error {
	// Connexion GORM
	if err := database.Connect(); err != nil {
		return fmt.Errorf("erreur de connexion GORM: %w", err)
	}

	// Auto-migration
	if err := database.AutoMigrate(); err != nil {
		return fmt.Errorf("erreur de migration GORM: %w", err)
	}

	// Initialisation des repositories
	db := database.GetDB()
	app.CryptoDataRepo = models.NewCryptoDataRepository(db)
	app.TechnicalIndicatorRepo = models.NewTechnicalIndicatorRepository(db)
	app.CryptoNewsRepo = models.NewCryptoNewsRepository(db)

	app.Logger.Info("✅ Connexion GORM et repositories initialisés")
	return nil
}

// Connexion à Redis
func (app *App) ConnectRedis() error {
	app.Redis = redis.NewClient(&redis.Options{
		Addr: app.Config.RedisURL,
	})

	// Test de la connexion
	ctx := context.Background()
	if err := app.Redis.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("impossible de se connecter à Redis: %w", err)
	}

	app.Logger.Info("✅ Connexion à Redis établie")
	return nil
}

// Configuration des routes
func (app *App) SetupRoutes() {
	app.Router = gin.Default()

	// Middleware CORS
	app.Router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// Routes de santé
	app.Router.GET("/health", app.healthCheck)
	app.Router.GET("/ready", app.readinessCheck)

	// API v1
	v1 := app.Router.Group("/api/v1")
	{
		// Routes crypto data
		v1.GET("/crypto/:symbol/data", app.getCryptoData)
		v1.GET("/crypto/:symbol/latest", app.getLatestPrice)

		// Routes indicateurs
		v1.GET("/indicators/:symbol/:type", app.getIndicators)
		v1.GET("/indicators/:symbol", app.getAllIndicators)

		// Routes actualités
		v1.GET("/news", app.getNews)
		v1.GET("/news/:symbol", app.getNewsBySymbol)

		// Routes statistiques
		v1.GET("/stats/:symbol", app.getCryptoStats)
	}

	// WebSocket
	app.Router.GET("/ws/crypto", app.handleWebSocket)
}

// Health check
func (app *App) healthCheck(c *gin.Context) {
	c.JSON(200, APIResponse{
		Success:   true,
		Message:   "Service healthy",
		Timestamp: time.Now(),
	})
}

// Readiness check
func (app *App) readinessCheck(c *gin.Context) {
	// Vérifier la connexion GORM
	if err := database.Health(); err != nil {
		c.JSON(503, APIResponse{
			Success:   false,
			Error:     "Database not ready",
			Timestamp: time.Now(),
		})
		return
	}

	// Vérifier Redis
	ctx := context.Background()
	if err := app.Redis.Ping(ctx).Err(); err != nil {
		c.JSON(503, APIResponse{
			Success:   false,
			Error:     "Redis not ready",
			Timestamp: time.Now(),
		})
		return
	}

	c.JSON(200, APIResponse{
		Success:   true,
		Message:   "Service ready",
		Timestamp: time.Now(),
	})
}

// Récupérer les données crypto avec GORM
func (app *App) getCryptoData(c *gin.Context) {
	symbol := c.Param("symbol")
	interval := c.DefaultQuery("interval", "1m")
	limit := c.DefaultQuery("limit", "100")

	limitInt, err := strconv.Atoi(limit)
	if err != nil || limitInt > 1000 {
		limitInt = 100
	}

	data, err := app.CryptoDataRepo.GetBySymbol(symbol, interval, limitInt)
	if err != nil {
		app.Logger.Error("Erreur requête crypto data: ", err)
		c.JSON(500, APIResponse{
			Success:   false,
			Error:     "Database error",
			Timestamp: time.Now(),
		})
		return
	}

	c.JSON(200, APIResponse{
		Success:   true,
		Data:      data,
		Timestamp: time.Now(),
	})
}

// Récupérer le dernier prix avec GORM
func (app *App) getLatestPrice(c *gin.Context) {
	symbol := c.Param("symbol")

	data, err := app.CryptoDataRepo.GetLatest(symbol)
	if err != nil {
		c.JSON(404, APIResponse{
			Success:   false,
			Error:     "No data found",
			Timestamp: time.Now(),
		})
		return
	}

	c.JSON(200, APIResponse{
		Success:   true,
		Data:      data,
		Timestamp: time.Now(),
	})
}

// Récupérer les indicateurs techniques avec GORM
func (app *App) getIndicators(c *gin.Context) {
	symbol := c.Param("symbol")
	indicatorType := c.Param("type")
	interval := c.DefaultQuery("interval", "1m")
	limit := c.DefaultQuery("limit", "100")

	limitInt, err := strconv.Atoi(limit)
	if err != nil || limitInt > 1000 {
		limitInt = 100
	}

	data, err := app.TechnicalIndicatorRepo.GetBySymbolAndType(symbol, indicatorType, interval, limitInt)
	if err != nil {
		app.Logger.Error("Erreur requête indicateurs: ", err)
		c.JSON(500, APIResponse{
			Success:   false,
			Error:     "Database error",
			Timestamp: time.Now(),
		})
		return
	}

	c.JSON(200, APIResponse{
		Success:   true,
		Data:      data,
		Timestamp: time.Now(),
	})
}

// Récupérer tous les indicateurs avec GORM
func (app *App) getAllIndicators(c *gin.Context) {
	symbol := c.Param("symbol")
	interval := c.DefaultQuery("interval", "1m")

	data, err := app.TechnicalIndicatorRepo.GetAllBySymbol(symbol, interval)
	if err != nil {
		app.Logger.Error("Erreur requête tous indicateurs: ", err)
		c.JSON(500, APIResponse{
			Success:   false,
			Error:     "Database error",
			Timestamp: time.Now(),
		})
		return
	}

	c.JSON(200, APIResponse{
		Success:   true,
		Data:      data,
		Timestamp: time.Now(),
	})
}

// Récupérer les actualités avec GORM
func (app *App) getNews(c *gin.Context) {
	limit := c.DefaultQuery("limit", "50")

	limitInt, err := strconv.Atoi(limit)
	if err != nil || limitInt > 100 {
		limitInt = 50
	}

	data, err := app.CryptoNewsRepo.GetAll(limitInt)
	if err != nil {
		app.Logger.Error("Erreur requête news: ", err)
		c.JSON(500, APIResponse{
			Success:   false,
			Error:     "Database error",
			Timestamp: time.Now(),
		})
		return
	}

	c.JSON(200, APIResponse{
		Success:   true,
		Data:      data,
		Timestamp: time.Now(),
	})
}

// Récupérer les actualités par symbole avec GORM
func (app *App) getNewsBySymbol(c *gin.Context) {
	symbol := c.Param("symbol")
	limit := c.DefaultQuery("limit", "20")

	limitInt, err := strconv.Atoi(limit)
	if err != nil || limitInt > 100 {
		limitInt = 20
	}

	data, err := app.CryptoNewsRepo.GetBySymbol(symbol, limitInt)
	if err != nil {
		app.Logger.Error("Erreur requête news par symbole: ", err)
		c.JSON(500, APIResponse{
			Success:   false,
			Error:     "Database error",
			Timestamp: time.Now(),
		})
		return
	}

	c.JSON(200, APIResponse{
		Success:   true,
		Data:      data,
		Timestamp: time.Now(),
	})
}

// Récupérer les statistiques crypto avec GORM
func (app *App) getCryptoStats(c *gin.Context) {
	symbol := c.Param("symbol")
	interval := c.DefaultQuery("interval", "1m")

	stats, err := app.CryptoDataRepo.GetStats(symbol, interval)
	if err != nil {
		app.Logger.Error("Erreur requête stats: ", err)
		c.JSON(404, APIResponse{
			Success:   false,
			Error:     "No stats found",
			Timestamp: time.Now(),
		})
		return
	}

	c.JSON(200, APIResponse{
		Success:   true,
		Data:      stats,
		Timestamp: time.Now(),
	})
}

// Gestionnaire WebSocket
func (app *App) handleWebSocket(c *gin.Context) {
	conn, err := app.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		app.Logger.Error("Erreur upgrade WebSocket: ", err)
		return
	}
	defer conn.Close()

	app.Logger.Info("Nouvelle connexion WebSocket")

	// Boucle de lecture des messages
	for {
		var msg map[string]interface{}
		err := conn.ReadJSON(&msg)
		if err != nil {
			app.Logger.Error("Erreur lecture WebSocket: ", err)
			break
		}

		// Traiter le message (subscribe/unsubscribe)
		if action, ok := msg["action"].(string); ok {
			switch action {
			case "subscribe":
				if symbol, ok := msg["symbol"].(string); ok {
					app.Logger.Infof("Subscription à %s", symbol)
					// Ici, on pourrait implémenter la logique de subscription
				}
			case "unsubscribe":
				if symbol, ok := msg["symbol"].(string); ok {
					app.Logger.Infof("Unsubscription de %s", symbol)
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
			app.Logger.Error("Erreur écriture WebSocket: ", err)
			break
		}
	}
}

// Démarrage du serveur
func (app *App) Start() error {
	// Connexion à la base de données avec GORM
	if err := app.ConnectDatabase(); err != nil {
		return err
	}

	// Connexion à Redis
	if err := app.ConnectRedis(); err != nil {
		return err
	}

	// Configuration des routes
	app.SetupRoutes()

	// Démarrage du serveur
	app.Logger.Infof("🚀 Démarrage du serveur sur le port %s", app.Config.Port)

	server := &http.Server{
		Addr:    ":" + app.Config.Port,
		Handler: app.Router,
	}

	// Gestion gracieuse de l'arrêt
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)
		<-sigint

		app.Logger.Info("🛑 Arrêt du serveur...")

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			app.Logger.Error("Erreur arrêt serveur: ", err)
		}

		// Fermeture des connexions
		if app.Redis != nil {
			app.Redis.Close()
		}
		database.Close()
	}()

	app.Logger.Info("✅ Serveur démarré avec succès")
	return server.ListenAndServe()
}

// Point d'entrée principal
func main() {
	app := NewApp()

	if err := app.Start(); err != nil && err != http.ErrServerClosed {
		log.Fatal("❌ Erreur démarrage serveur:", err)
	}

	app.Logger.Info("✅ Serveur arrêté proprement")
}
