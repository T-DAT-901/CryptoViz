package database

import (
	"fmt"
	"log"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"cryptoviz-backend/models"
)

// DB instance globale
var DB *gorm.DB

// Config structure de configuration de la base de données
type Config struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// NewConfig crée une nouvelle configuration depuis les variables d'environnement
func NewConfig() *Config {
	return &Config{
		Host:     getEnv("TIMESCALE_HOST", "timescaledb"),
		Port:     getEnv("TIMESCALE_PORT", "5432"),
		User:     getEnv("TIMESCALE_USER", "postgres"),
		Password: getEnv("TIMESCALE_PASSWORD", "password"),
		DBName:   getEnv("TIMESCALE_DB", "cryptoviz"),
		SSLMode:  getEnv("SSL_MODE", "disable"),
	}
}

// getEnv récupère une variable d'environnement avec une valeur par défaut
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// BuildDSN construit la chaîne de connexion PostgreSQL
func (c *Config) BuildDSN() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode)
}

// Connect établit la connexion à la base de données
func Connect() error {
	config := NewConfig()
	dsn := config.BuildDSN()

	// Configuration du logger GORM
	gormLogger := logger.Default
	if os.Getenv("GIN_MODE") == "release" {
		gormLogger = logger.Default.LogMode(logger.Silent)
	} else {
		gormLogger = logger.Default.LogMode(logger.Info)
	}

	// Configuration GORM
	gormConfig := &gorm.Config{
		Logger: gormLogger,
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	}

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), gormConfig)
	if err != nil {
		return fmt.Errorf("erreur de connexion à la base de données: %w", err)
	}

	// Configuration de la pool de connexions
	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("erreur d'accès à la base SQL: %w", err)
	}

	// Configuration des paramètres de connexion
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// Test de la connexion
	if err := sqlDB.Ping(); err != nil {
		return fmt.Errorf("impossible de ping la base de données: %w", err)
	}

	log.Println("✅ Connexion à TimescaleDB établie avec GORM")
	return nil
}

// AutoMigrate exécute les migrations automatiques
func AutoMigrate() error {
	if DB == nil {
		return fmt.Errorf("base de données non initialisée")
	}

	// Migration des modèles
	err := DB.AutoMigrate(
		&models.CryptoData{},
		&models.TechnicalIndicator{},
		&models.CryptoNews{},
	)
	if err != nil {
		return fmt.Errorf("erreur lors de la migration: %w", err)
	}

	// Configuration spécifique à TimescaleDB
	if err := setupTimescaleDB(); err != nil {
		log.Printf("⚠️  Avertissement TimescaleDB: %v", err)
		// Ne pas retourner d'erreur car TimescaleDB peut ne pas être disponible en dev
	}

	log.Println("✅ Migrations GORM terminées")
	return nil
}

// setupTimescaleDB configure les hypertables TimescaleDB
func setupTimescaleDB() error {
	// Créer les hypertables si elles n'existent pas
	hypertables := []struct {
		table      string
		timeColumn string
	}{
		{"crypto_data", "time"},
		{"crypto_indicators", "time"},
		{"crypto_news", "time"},
	}

	for _, ht := range hypertables {
		// Vérifier si la table est déjà une hypertable
		var exists bool
		query := `
			SELECT EXISTS (
				SELECT 1 FROM timescaledb_information.hypertables
				WHERE hypertable_name = ?
			)
		`
		if err := DB.Raw(query, ht.table).Scan(&exists).Error; err != nil {
			log.Printf("⚠️  Impossible de vérifier l'hypertable %s: %v", ht.table, err)
			continue
		}

		if !exists {
			// Créer l'hypertable
			createQuery := fmt.Sprintf(
				"SELECT create_hypertable('%s', '%s', if_not_exists => TRUE)",
				ht.table, ht.timeColumn,
			)
			if err := DB.Exec(createQuery).Error; err != nil {
				log.Printf("⚠️  Impossible de créer l'hypertable %s: %v", ht.table, err)
				continue
			}
			log.Printf("✅ Hypertable %s créée", ht.table)
		}
	}

	return nil
}

// Close ferme la connexion à la base de données
func Close() error {
	if DB == nil {
		return nil
	}

	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}

	return sqlDB.Close()
}

// GetDB retourne l'instance de la base de données
func GetDB() *gorm.DB {
	return DB
}

// Health vérifie la santé de la connexion
func Health() error {
	if DB == nil {
		return fmt.Errorf("base de données non initialisée")
	}

	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}

	return sqlDB.Ping()
}
