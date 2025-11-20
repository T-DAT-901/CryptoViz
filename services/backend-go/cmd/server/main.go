package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"cryptoviz-backend/database"
	"cryptoviz-backend/internal/config"
	"cryptoviz-backend/internal/controllers"
	"cryptoviz-backend/internal/kafka"
	"cryptoviz-backend/internal/kafka/consumers"
	"cryptoviz-backend/internal/routes"
	ws "cryptoviz-backend/internal/websocket"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"
)

func main() {
	// Configuration du logger
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetLevel(logrus.InfoLevel)

	// Chargement de la configuration
	cfg := config.Load()

	// Configuration du mode Gin
	gin.SetMode(cfg.GinMode)

	// Connexion à la base de données
	logger.Info("Connexion à la base de données...")
	if err := database.Connect(); err != nil {
		log.Fatal("❌ Erreur de connexion à la base de données:", err)
	}
	defer database.Close()
	logger.Info("✅ Connexion à la base de données établie")

	// Connexion à Redis
	logger.Info("Connexion à Redis...")
	redisClient := redis.NewClient(&redis.Options{
		Addr: cfg.RedisURL(),
	})

	// Test de la connexion Redis
	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatal("❌ Impossible de se connecter à Redis:", err)
	}
	defer redisClient.Close()
	logger.Info("✅ Connexion à Redis établie")

	// Initialisation du WebSocket Hub
	logger.Info("Initialisation du WebSocket Hub...")
	wsHub := ws.NewHub(logger)
	go wsHub.Run()
	logger.Info("✅ WebSocket Hub démarré")

	// Initialisation des dépendances
	db := database.GetDB()
	deps := controllers.NewDependencies(db, redisClient, logger, wsHub)

	// Configuration Kafka
	logger.Info("Configuration des consommateurs Kafka...")
	kafkaConfig := kafka.NewKafkaConfig(cfg)
	consumerManager := kafka.NewConsumerManager(logger)

	// Enregistrer les handlers Kafka
	candleHandler := consumers.NewCandleHandler(kafkaConfig, deps.CandleRepo, redisClient, logger)
	tradeHandler := consumers.NewTradeHandler(kafkaConfig, deps.TradeRepo, redisClient, logger)
	indicatorHandler := consumers.NewIndicatorHandler(kafkaConfig, deps.IndicatorRepo, redisClient, logger)
	newsHandler := consumers.NewNewsHandler(kafkaConfig, deps.NewsRepo, redisClient, logger)

	// Connect handlers to WebSocket Hub for real-time broadcasting
	tradeHandler.SetBroadcast(wsHub.Broadcast)
	candleHandler.SetBroadcast(wsHub.Broadcast)
	logger.Info("✅ WebSocket broadcast connected to Kafka handlers")

	if err := consumerManager.RegisterHandler(kafkaConfig, candleHandler); err != nil {
		log.Fatal("❌ Erreur lors de l'enregistrement du CandleHandler:", err)
	}
	if err := consumerManager.RegisterHandler(kafkaConfig, tradeHandler); err != nil {
		log.Fatal("❌ Erreur lors de l'enregistrement du TradeHandler:", err)
	}
	if err := consumerManager.RegisterHandler(kafkaConfig, indicatorHandler); err != nil {
		log.Fatal("❌ Erreur lors de l'enregistrement de l'IndicatorHandler:", err)
	}
	if err := consumerManager.RegisterHandler(kafkaConfig, newsHandler); err != nil {
		log.Fatal("❌ Erreur lors de l'enregistrement du NewsHandler:", err)
	}

	// Assigner le consumer manager aux dépendances
	deps.ConsumerManager = consumerManager

	// Démarrer les consommateurs Kafka
	go func() {
		if err := consumerManager.StartAll(ctx); err != nil {
			logger.Errorf("❌ Erreur lors du démarrage des consommateurs Kafka: %v", err)
		}
	}()
	logger.Info("✅ Consommateurs Kafka démarrés")

	// Configuration des routes
	router := routes.Setup(deps, logger)

	// Configuration du serveur HTTP
	server := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: router,
	}

	// Démarrage du serveur dans une goroutine
	go func() {
		logger.Infof("🚀 Serveur démarré sur le port %s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("❌ Erreur démarrage serveur:", err)
		}
	}()

	logger.Info("✅ Application démarrée avec succès")

	// Gestion gracieuse de l'arrêt
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	logger.Info("🛑 Arrêt du serveur...")

	// Timeout pour l'arrêt gracieux
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Arrêter les consommateurs Kafka
	logger.Info("Arrêt des consommateurs Kafka...")
	if err := consumerManager.StopAll(); err != nil {
		logger.Errorf("❌ Erreur lors de l'arrêt des consommateurs Kafka: %v", err)
	} else {
		logger.Info("✅ Consommateurs Kafka arrêtés proprement")
	}

	// Arrêter le serveur HTTP
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatal("❌ Erreur lors de l'arrêt du serveur:", err)
	}

	logger.Info("✅ Serveur arrêté proprement")
}
