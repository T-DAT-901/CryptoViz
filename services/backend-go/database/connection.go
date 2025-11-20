package database

import (
	"fmt"
	"log"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
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
		// Disable automatic table creation - we use init.sql
		DisableAutomaticPing:   false,
		SkipDefaultTransaction: true,
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

	// Verify TimescaleDB extension
	if err := verifyTimescaleDB(); err != nil {
		log.Printf("⚠️  Warning: %v", err)
	}

	return nil
}

// verifyTimescaleDB vérifie que l'extension TimescaleDB est installée
func verifyTimescaleDB() error {
	var version string
	err := DB.Raw("SELECT extversion FROM pg_extension WHERE extname = 'timescaledb'").Scan(&version).Error
	if err != nil {
		return fmt.Errorf("TimescaleDB extension not found: %w", err)
	}

	log.Printf("✅ TimescaleDB version %s detected", version)
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

// GetDatabaseInfo retourne les informations sur la base de données
func GetDatabaseInfo() (map[string]interface{}, error) {
	info := make(map[string]interface{})

	// Get TimescaleDB version
	var tsVersion string
	err := DB.Raw("SELECT extversion FROM pg_extension WHERE extname = 'timescaledb'").Scan(&tsVersion).Error
	if err == nil {
		info["timescaledb_version"] = tsVersion
	}

	// Get PostgreSQL version
	var pgVersion string
	err = DB.Raw("SHOW server_version").Scan(&pgVersion).Error
	if err == nil {
		info["postgresql_version"] = pgVersion
	}

	// Get hypertables count
	var hypertablesCount int64
	err = DB.Raw("SELECT COUNT(*) FROM timescaledb_information.hypertables").Scan(&hypertablesCount).Error
	if err == nil {
		info["hypertables_count"] = hypertablesCount
	}

	// Get continuous aggregates count
	var caggCount int64
	err = DB.Raw("SELECT COUNT(*) FROM timescaledb_information.continuous_aggregates").Scan(&caggCount).Error
	if err == nil {
		info["continuous_aggregates_count"] = caggCount
	}

	// Get database size
	var dbSize string
	err = DB.Raw("SELECT pg_size_pretty(pg_database_size(current_database()))").Scan(&dbSize).Error
	if err == nil {
		info["database_size"] = dbSize
	}

	return info, nil
}
