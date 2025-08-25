package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/go-redis/redis/v8"
	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"
)

// Configuration de l'application
type Config struct {
	Port         string
	DatabaseURL  string
	RedisURL     string
	KafkaBrokers string
}

// Structure pour les données crypto
type CryptoData struct {
	Time         time.Time `json:"time" db:"time"`
	Symbol       string    `json:"symbol" db:"symbol"`
	IntervalType string    `json:"interval_type" db:"interval_type"`
	Open         float64   `json:"open" db:"open"`
	High         float64   `json:"high" db:"high"`
	Low          float64   `json:"low" db:"low"`
	Close        float64   `json:"close" db:"close"`
	Volume       float64   `json:"volume" db:"volume"`
}

// Structure pour les indicateurs techniques
type TechnicalIndicator struct {
	Time          time.Time `json:"time" db:"time"`
	Symbol        string    `json:"symbol" db:"symbol"`
	IntervalType  string    `json:"interval_type" db:"interval_type"`
	IndicatorType string    `json:"indicator_type" db:"indicator_type"`
	Value         *float64  `json:"value" db:"value"`
	ValueSignal   *float64  `json:"value_signal" db:"value_signal"`
	ValueHisto    *float64  `json:"value_histogram" db:"value_histogram"`
	UpperBand     *float64  `json:"upper_band" db:"upper_band"`
	LowerBand     *float64  `json:"lower_band" db:"lower_band"`
	MiddleBand    *float64  `json:"middle_band" db:"middle_band"`
}

// Structure pour les actualités
type CryptoNews struct {
	ID             int       `json:"id" db:"id"`
	Time           time.Time `json:"time" db:"time"`
	Title          string    `json:"title" db:"title"`
	Content        string    `json:"content" db:"content"`
	Source         string    `json:"source" db:"source"`
	URL            string    `json:"url" db:"url"`
	SentimentScore *float64  `json:"sentiment_score" db:"sentiment_score"`
	Symbols        []string  `json:"symbols" db:"symbols"`
}

// Structure de réponse API
type APIResponse struct {
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,omitempty"`
	Error     string      `json:"error,omitempty"`
	Message   string      `json:"message,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

// Application principale
type App struct {
	Config   *Config
	DB       *sql.DB
	Redis    *redis.Client
	Router   *gin.Engine
	Logger   *logrus.Logger
	upgrader websocket.Upgrader
}

// Initialisation de la configuration
func NewConfig() *Config {
	return &Config{
		Port:         getEnv("PORT", "8080"),
		DatabaseURL:  buildDatabaseURL(),
		RedisURL:     buildRedisURL(),
		KafkaBrokers: getEnv("KAFKA_BROKERS", "kafka:29092"),
	}
}

// Construction de l'URL de base de données
func buildDatabaseURL() string {
	host := getEnv("TIMESCALE_HOST", "timescaledb")
	port := getEnv("TIMESCALE_PORT", "5432")
	dbname := getEnv("TIMESCALE_DB", "cryptoviz")
	user := getEnv("TIMESCALE_USER", "postgres")
	password := getEnv("TIMESCALE_PASSWORD", "password")

	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		user, password, host, port, dbname)
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

// Connexion à la base de données
func (app *App) ConnectDatabase() error {
	var err error
	app.DB, err = sql.Open("postgres", app.Config.DatabaseURL)
	if err != nil {
		return fmt.Errorf("erreur de connexion à la base de données: %w", err)
	}

	// Test de la connexion
	if err = app.DB.Ping(); err != nil {
		return fmt.Errorf("impossible de ping la base de données: %w", err)
	}

	app.Logger.Info("Connexion à TimescaleDB établie")
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

	app.Logger.Info("Connexion à Redis établie")
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
	// Vérifier les connexions
	if err := app.DB.Ping(); err != nil {
		c.JSON(503, APIResponse{
			Success:   false,
			Error:     "Database not ready",
			Timestamp: time.Now(),
		})
		return
	}

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

// Récupérer les données crypto
func (app *App) getCryptoData(c *gin.Context) {
	symbol := c.Param("symbol")
	interval := c.DefaultQuery("interval", "1m")
	limit := c.DefaultQuery("limit", "100")

	limitInt, err := strconv.Atoi(limit)
	if err != nil || limitInt > 1000 {
		limitInt = 100
	}

	query := `
		SELECT time, symbol, interval_type, open, high, low, close, volume
		FROM crypto_data
		WHERE symbol = $1 AND interval_type = $2
		ORDER BY time DESC
		LIMIT $3
	`

	rows, err := app.DB.Query(query, symbol, interval, limitInt)
	if err != nil {
		app.Logger.Error("Erreur requête crypto data", err)
		c.JSON(500, APIResponse{
			Success:   false,
			Error:     "Database error",
			Timestamp: time.Now(),
		})
		return
	}
	defer rows.Close()

	var data []CryptoData
	for rows.Next() {
		var item CryptoData
		err := rows.Scan(&item.Time, &item.Symbol, &item.IntervalType,
			&item.Open, &item.High, &item.Low, &item.Close, &item.Volume)
		if err != nil {
			continue
		}
		data = append(data, item)
	}

	c.JSON(200, APIResponse{
		Success:   true,
		Data:      data,
		Timestamp: time.Now(),
	})
}

// Récupérer le dernier prix
func (app *App) getLatestPrice(c *gin.Context) {
	symbol := c.Param("symbol")

	query := `
		SELECT time, symbol, interval_type, open, high, low, close, volume
		FROM crypto_data
		WHERE symbol = $1
		ORDER BY time DESC
		LIMIT 1
	`

	var data CryptoData
	err := app.DB.QueryRow(query, symbol).Scan(
		&data.Time, &data.Symbol, &data.IntervalType,
		&data.Open, &data.High, &data.Low, &data.Close, &data.Volume)

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

// Récupérer les indicateurs techniques
func (app *App) getIndicators(c *gin.Context) {
	symbol := c.Param("symbol")
	indicatorType := c.Param("type")
	interval := c.DefaultQuery("interval", "1m")
	limit := c.DefaultQuery("limit", "100")

	limitInt, err := strconv.Atoi(limit)
	if err != nil || limitInt > 1000 {
		limitInt = 100
	}

	query := `
		SELECT time, symbol, interval_type, indicator_type, value,
		       value_signal, value_histogram, upper_band, lower_band, middle_band
		FROM crypto_indicators
		WHERE symbol = $1 AND indicator_type = $2 AND interval_type = $3
		ORDER BY time DESC
		LIMIT $4
	`

	rows, err := app.DB.Query(query, symbol, indicatorType, interval, limitInt)
	if err != nil {
		c.JSON(500, APIResponse{
			Success:   false,
			Error:     "Database error",
			Timestamp: time.Now(),
		})
		return
	}
	defer rows.Close()

	var data []TechnicalIndicator
	for rows.Next() {
		var item TechnicalIndicator
		err := rows.Scan(&item.Time, &item.Symbol, &item.IntervalType,
			&item.IndicatorType, &item.Value, &item.ValueSignal,
			&item.ValueHisto, &item.UpperBand, &item.LowerBand, &item.MiddleBand)
		if err != nil {
			continue
		}
		data = append(data, item)
	}

	c.JSON(200, APIResponse{
		Success:   true,
		Data:      data,
		Timestamp: time.Now(),
	})
}

// Récupérer tous les indicateurs
func (app *App) getAllIndicators(c *gin.Context) {
	symbol := c.Param("symbol")
	interval := c.DefaultQuery("interval", "1m")

	query := `
		SELECT DISTINCT ON (indicator_type)
		       time, symbol, interval_type, indicator_type, value,
		       value_signal, value_histogram, upper_band, lower_band, middle_band
		FROM crypto_indicators
		WHERE symbol = $1 AND interval_type = $2
		ORDER BY indicator_type, time DESC
	`

	rows, err := app.DB.Query(query, symbol, interval)
	if err != nil {
		c.JSON(500, APIResponse{
			Success:   false,
			Error:     "Database error",
			Timestamp: time.Now(),
		})
		return
	}
	defer rows.Close()

	var data []TechnicalIndicator
	for rows.Next() {
		var item TechnicalIndicator
		err := rows.Scan(&item.Time, &item.Symbol, &item.IntervalType,
			&item.IndicatorType, &item.Value, &item.ValueSignal,
			&item.ValueHisto, &item.UpperBand, &item.LowerBand, &item.MiddleBand)
		if err != nil {
			continue
		}
		data = append(data, item)
	}

	c.JSON(200, APIResponse{
		Success:   true,
		Data:      data,
		Timestamp: time.Now(),
	})
}

// Récupérer les actualités
func (app *App) getNews(c *gin.Context) {
	limit := c.DefaultQuery("limit", "50")

	limitInt, err := strconv.Atoi(limit)
	if err != nil || limitInt > 100 {
		limitInt = 50
	}

	query := `
		SELECT id, time, title, content, source, url, sentiment_score, symbols
		FROM crypto_news
		ORDER BY time DESC
		LIMIT $1
	`

	rows, err := app.DB.Query(query, limitInt)
	if err != nil {
		c.JSON(500, APIResponse{
			Success:   false,
			Error:     "Database error",
			Timestamp: time.Now(),
		})
		return
	}
	defer rows.Close()

	var data []CryptoNews
	for rows.Next() {
		var item CryptoNews
		var symbolsJSON []byte
		err := rows.Scan(&item.ID, &item.Time, &item.Title, &item.Content,
			&item.Source, &item.URL, &item.SentimentScore, &symbolsJSON)
		if err != nil {
			continue
		}

		// Décoder le JSON des symboles
		if len(symbolsJSON) > 0 {
			json.Unmarshal(symbolsJSON, &item.Symbols)
		}

		data = append(data, item)
	}

	c.JSON(200, APIResponse{
		Success:   true,
		Data:      data,
		Timestamp: time.Now(),
	})
}

// Récupérer les actualités par symbole
func (app *App) getNewsBySymbol(c *gin.Context) {
	symbol := c.Param("symbol")
	limit := c.DefaultQuery("limit", "20")

	limitInt, err := strconv.Atoi(limit)
	if err != nil || limitInt > 100 {
		limitInt = 20
	}

	query := `
		SELECT id, time, title, content, source, url, sentiment_score, symbols
		FROM crypto_news
		WHERE $1 = ANY(symbols)
		ORDER BY time DESC
		LIMIT $2
	`

	rows, err := app.DB.Query(query, symbol, limitInt)
	if err != nil {
		c.JSON(500, APIResponse{
			Success:   false,
			Error:     "Database error",
			Timestamp: time.Now(),
		})
		return
	}
	defer rows.Close()

	var data []CryptoNews
	for rows.Next() {
		var item CryptoNews
		var symbolsJSON []byte
		err := rows.Scan(&item.ID, &item.Time, &item.Title, &item.Content,
			&item.Source, &item.URL, &item.SentimentScore, &symbolsJSON)
		if err != nil {
			continue
		}

		if len(symbolsJSON) > 0 {
			json.Unmarshal(symbolsJSON, &item.Symbols)
		}

		data = append(data, item)
	}

	c.JSON(200, APIResponse{
		Success:   true,
		Data:      data,
		Timestamp: time.Now(),
	})
}

// Récupérer les statistiques crypto
func (app *App) getCryptoStats(c *gin.Context) {
	symbol := c.Param("symbol")
	interval := c.DefaultQuery("interval", "1m")

	query := `SELECT * FROM get_crypto_stats($1, $2)`

	var stats struct {
		Symbol           string   `json:"symbol"`
		IntervalType     string   `json:"interval_type"`
		LatestPrice      *float64 `json:"latest_price"`
		PriceChange24h   *float64 `json:"price_change_24h"`
		PriceChangePct   *float64 `json:"price_change_pct_24h"`
		Volume24h        *float64 `json:"volume_24h"`
		High24h          *float64 `json:"high_24h"`
		Low24h           *float64 `json:"low_24h"`
	}

	err := app.DB.QueryRow(query, symbol, interval).Scan(
		&stats.Symbol, &stats.IntervalType, &stats.LatestPrice,
		&stats.PriceChange24h, &stats.PriceChangePct, &stats.Volume24h,
		&stats.High24h, &stats.Low24h)

	if err != nil {
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
		app.Logger.Error("Erreur upgrade WebSocket", err)
		return
	}
	defer conn.Close()

	app.Logger.Info("Nouvelle connexion WebSocket")

	// Boucle de lecture des messages
	for {
		var msg map[string]interface{}
		err := conn.ReadJSON(&msg)
		if err != nil {
			app.Logger.Error("Erreur lecture WebSocket", err)
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
			app.Logger.Error("Erreur écriture WebSocket", err)
			break
		}
	}
}

// Démarrage du serveur
func (app *App) Start() error {
	// Connexions
	if err := app.ConnectDatabase(); err != nil {
		return err
	}

	if err := app.ConnectRedis(); err != nil {
		return err
	}

	// Configuration des routes
	app.SetupRoutes()

	// Démarrage du serveur
	app.Logger.Infof("Démarrage du serveur sur le port %s", app.Config.Port)

	server := &http.Server{
		Addr:    ":" + app.Config.Port,
		Handler: app.Router,
	}

	// Gestion gracieuse de l'arrêt
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)
		<-sigint

		app.Logger.Info("Arrêt du serveur...")

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			app.Logger.Error("Erreur arrêt serveur", err)
		}

		// Fermeture des connexions
		if app.DB != nil {
			app.DB.Close()
		}
		if app.Redis != nil {
			app.Redis.Close()
		}
	}()

	app.Logger.Info("Server started successfully")
	return server.ListenAndServe()
}

// Point d'entrée principal
func main() {
	app := NewApp()

	if err := app.Start(); err != nil && err != http.ErrServerClosed {
		log.Fatal("Erreur démarrage serveur:", err)
	}

	app.Logger.Info("Serveur arrêté")
}
