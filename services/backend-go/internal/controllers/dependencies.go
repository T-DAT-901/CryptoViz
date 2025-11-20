package controllers

import (
	"cryptoviz-backend/internal/kafka"
	"cryptoviz-backend/models"

	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// Dependencies contient toutes les dépendances nécessaires aux contrôleurs
type Dependencies struct {
	DB              *gorm.DB
	Redis           *redis.Client
	Logger          *logrus.Logger
	CandleRepo      models.CandleRepository
	IndicatorRepo   models.IndicatorRepository
	NewsRepo        models.NewsRepository
	TradeRepo       models.TradeRepository
	UserRepo        models.UserRepository
	CurrencyRepo    models.CurrencyRepository
	ConsumerManager *kafka.ConsumerManager
}

// NewDependencies crée une nouvelle instance de Dependencies
func NewDependencies(db *gorm.DB, redis *redis.Client, logger *logrus.Logger) *Dependencies {
	return &Dependencies{
		DB:            db,
		Redis:         redis,
		Logger:        logger,
		CandleRepo:    models.NewCandleRepository(db),
		IndicatorRepo: models.NewIndicatorRepository(db),
		NewsRepo:      models.NewNewsRepository(db),
		TradeRepo:     models.NewTradeRepository(db),
		UserRepo:      models.NewUserRepository(db),
		CurrencyRepo:  models.NewCurrencyRepository(db),
	}
}
