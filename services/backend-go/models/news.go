package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"gorm.io/gorm"
)

// StringArray type personnalisé pour gérer les arrays de strings en PostgreSQL (JSONB)
type StringArray []string

// Value implémente l'interface driver.Valuer pour la sérialisation
func (sa StringArray) Value() (driver.Value, error) {
	if len(sa) == 0 {
		return nil, nil
	}
	return json.Marshal(sa)
}

// Scan implémente l'interface sql.Scanner pour la désérialisation
func (sa *StringArray) Scan(value interface{}) error {
	if value == nil {
		*sa = nil
		return nil
	}

	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, sa)
	case string:
		return json.Unmarshal([]byte(v), sa)
	default:
		return errors.New("cannot scan into StringArray")
	}
}

// News représente les actualités crypto
// Table: news (hypertable TimescaleDB)
type News struct {
	Time           time.Time   `json:"time" gorm:"primaryKey;column:time;not null"`
	Source         string      `json:"source" gorm:"primaryKey;column:source;size:100;not null"`
	URL            string      `json:"url" gorm:"primaryKey;column:url;not null"`
	Title          string      `json:"title" gorm:"column:title;not null"`
	Content        string      `json:"content" gorm:"column:content;type:text"`
	SentimentScore *float64    `json:"sentiment_score" gorm:"column:sentiment_score;type:decimal(5,2)"`
	Symbols        StringArray `json:"symbols" gorm:"column:symbols;type:jsonb"` // Array of symbols: ["BTC/USDT", "ETH/USDT"]
	CreatedAt      time.Time   `json:"created_at" gorm:"column:created_at;autoCreateTime"`
}

// TableName spécifie le nom de la table pour GORM
func (News) TableName() string {
	return "news"
}

// AllNews représente une requête sur la vue unifiée (hot + cold storage)
// View: all_news (transparent tiering)
type AllNews struct {
	News
}

// TableName spécifie le nom de la vue pour GORM
func (AllNews) TableName() string {
	return "all_news"
}

// NewsRepository interface pour les opérations sur News
type NewsRepository interface {
	Create(news *News) error
	CreateBatch(news []*News) error
	GetAll(limit int) ([]News, error)
	GetBySymbol(symbol string, limit int) ([]News, error)
	GetByTimeRange(start, end time.Time, limit int) ([]News, error)
	GetBySource(source string, limit int) ([]News, error)
	GetBySentiment(minScore, maxScore float64, limit int) ([]News, error)
	GetByCompositeKey(time time.Time, source, url string) (*News, error)
	Update(news *News) error
	DeleteByCompositeKey(time time.Time, source, url string) error

	// Queries on unified view (hot + cold storage)
	GetAllUnified(limit int) ([]AllNews, error)
	GetBySymbolUnified(symbol string, limit int) ([]AllNews, error)
	GetByTimeRangeUnified(start, end time.Time, limit int) ([]AllNews, error)
}

// newsRepository implémentation concrète du repository
type newsRepository struct {
	db *gorm.DB
}

// NewNewsRepository crée une nouvelle instance du repository
func NewNewsRepository(db *gorm.DB) NewsRepository {
	return &newsRepository{db: db}
}

// Create insère une nouvelle actualité
func (r *newsRepository) Create(news *News) error {
	return r.db.Create(news).Error
}

// CreateBatch insère plusieurs actualités en une seule transaction
func (r *newsRepository) CreateBatch(news []*News) error {
	if len(news) == 0 {
		return nil
	}

	return r.db.Transaction(func(tx *gorm.DB) error {
		for _, n := range news {
			if err := tx.Create(n).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// GetAll récupère toutes les actualités (hot storage only)
func (r *newsRepository) GetAll(limit int) ([]News, error) {
	var news []News
	err := r.db.Order("time DESC").
		Limit(limit).
		Find(&news).Error
	return news, err
}

// GetBySymbol récupère les actualités pour un symbole spécifique
func (r *newsRepository) GetBySymbol(symbol string, limit int) ([]News, error) {
	var news []News
	// Utiliser l'opérateur @> de PostgreSQL pour JSONB contains
	err := r.db.Where("symbols @> ?", `["`+symbol+`"]`).
		Order("time DESC").
		Limit(limit).
		Find(&news).Error
	return news, err
}

// GetByTimeRange récupère les actualités dans une plage de temps
func (r *newsRepository) GetByTimeRange(start, end time.Time, limit int) ([]News, error) {
	var news []News
	err := r.db.Where("time BETWEEN ? AND ?", start, end).
		Order("time DESC").
		Limit(limit).
		Find(&news).Error
	return news, err
}

// GetBySource récupère les actualités par source
func (r *newsRepository) GetBySource(source string, limit int) ([]News, error) {
	var news []News
	err := r.db.Where("source = ?", source).
		Order("time DESC").
		Limit(limit).
		Find(&news).Error
	return news, err
}

// GetBySentiment récupère les actualités par score de sentiment
func (r *newsRepository) GetBySentiment(minScore, maxScore float64, limit int) ([]News, error) {
	var news []News
	err := r.db.Where("sentiment_score BETWEEN ? AND ?", minScore, maxScore).
		Order("time DESC").
		Limit(limit).
		Find(&news).Error
	return news, err
}

// GetByCompositeKey récupère une actualité par sa clé composite
func (r *newsRepository) GetByCompositeKey(time time.Time, source, url string) (*News, error) {
	var news News
	err := r.db.Where("time = ? AND source = ? AND url = ?", time, source, url).First(&news).Error
	if err != nil {
		return nil, err
	}
	return &news, nil
}

// Update met à jour une actualité
func (r *newsRepository) Update(news *News) error {
	return r.db.Save(news).Error
}

// DeleteByCompositeKey supprime une actualité par sa clé composite
func (r *newsRepository) DeleteByCompositeKey(time time.Time, source, url string) error {
	return r.db.Where("time = ? AND source = ? AND url = ?", time, source, url).Delete(&News{}).Error
}

// GetAllUnified récupère toutes les actualités (hot + cold storage via vue unifiée)
func (r *newsRepository) GetAllUnified(limit int) ([]AllNews, error) {
	var news []AllNews
	err := r.db.Order("time DESC").
		Limit(limit).
		Find(&news).Error
	return news, err
}

// GetBySymbolUnified récupère les actualités pour un symbole (hot + cold storage)
func (r *newsRepository) GetBySymbolUnified(symbol string, limit int) ([]AllNews, error) {
	var news []AllNews
	err := r.db.Where("symbols @> ?", `["`+symbol+`"]`).
		Order("time DESC").
		Limit(limit).
		Find(&news).Error
	return news, err
}

// GetByTimeRangeUnified récupère les actualités dans une plage de temps (hot + cold storage)
func (r *newsRepository) GetByTimeRangeUnified(start, end time.Time, limit int) ([]AllNews, error) {
	var news []AllNews
	err := r.db.Where("time BETWEEN ? AND ?", start, end).
		Order("time DESC").
		Limit(limit).
		Find(&news).Error
	return news, err
}
