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
	"cryptoviz-backend/internal/routes"

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

	// Initialisation des dépendances
	db := database.GetDB()
	deps := controllers.NewDependencies(db, redisClient, logger)

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
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatal("❌ Erreur lors de l'arrêt du serveur:", err)
	}

	logger.Info("✅ Serveur arrêté proprement")
}
