package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"gorm.io/gorm"
)

// StringArray type personnalisé pour gérer les arrays de strings en PostgreSQL
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

// CryptoNews représente les actualités crypto
type CryptoNews struct {
	Time           time.Time   `json:"time" gorm:"primaryKey;column:time;index"`
	Source         string      `json:"source" gorm:"primaryKey;column:source;size:100;not null"`
	URL            string      `json:"url" gorm:"primaryKey;column:url;size:1000;not null"`
	Title          string      `json:"title" gorm:"column:title;size:500;not null"`
	Content        string      `json:"content" gorm:"column:content;type:text"`
	SentimentScore *float64    `json:"sentiment_score" gorm:"column:sentiment_score;type:decimal(5,4)"`
	Symbols        StringArray `json:"symbols" gorm:"column:symbols;type:jsonb"`
	CreatedAt      time.Time   `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt      time.Time   `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName spécifie le nom de la table pour GORM
func (CryptoNews) TableName() string {
	return "crypto_news"
}

// BeforeCreate hook GORM appelé avant la création
func (cn *CryptoNews) BeforeCreate(tx *gorm.DB) error {
	if cn.Time.IsZero() {
		cn.Time = time.Now()
	}
	return nil
}

// CryptoNewsRepository interface pour les opérations sur CryptoNews
type CryptoNewsRepository interface {
	Create(news *CryptoNews) error
	GetAll(limit int) ([]CryptoNews, error)
	GetBySymbol(symbol string, limit int) ([]CryptoNews, error)
	GetByTimeRange(start, end time.Time, limit int) ([]CryptoNews, error)
	GetBySource(source string, limit int) ([]CryptoNews, error)
	GetBySentiment(minScore, maxScore float64, limit int) ([]CryptoNews, error)
	GetByCompositeKey(time time.Time, source, url string) (*CryptoNews, error)
	Update(news *CryptoNews) error
	DeleteByCompositeKey(time time.Time, source, url string) error
}

// cryptoNewsRepository implémentation concrète du repository
type cryptoNewsRepository struct {
	db *gorm.DB
}

// NewCryptoNewsRepository crée une nouvelle instance du repository
func NewCryptoNewsRepository(db *gorm.DB) CryptoNewsRepository {
	return &cryptoNewsRepository{db: db}
}

// Create insère une nouvelle actualité
func (r *cryptoNewsRepository) Create(news *CryptoNews) error {
	return r.db.Create(news).Error
}

// GetAll récupère toutes les actualités
func (r *cryptoNewsRepository) GetAll(limit int) ([]CryptoNews, error) {
	var news []CryptoNews
	err := r.db.Order("time DESC").
		Limit(limit).
		Find(&news).Error
	return news, err
}

// GetBySymbol récupère les actualités pour un symbole spécifique
func (r *cryptoNewsRepository) GetBySymbol(symbol string, limit int) ([]CryptoNews, error) {
	var news []CryptoNews
	// Utiliser une requête JSON pour PostgreSQL
	err := r.db.Where("symbols @> ?", `["`+symbol+`"]`).
		Order("time DESC").
		Limit(limit).
		Find(&news).Error
	return news, err
}

// GetByTimeRange récupère les actualités dans une plage de temps
func (r *cryptoNewsRepository) GetByTimeRange(start, end time.Time, limit int) ([]CryptoNews, error) {
	var news []CryptoNews
	err := r.db.Where("time BETWEEN ? AND ?", start, end).
		Order("time DESC").
		Limit(limit).
		Find(&news).Error
	return news, err
}

// GetBySource récupère les actualités par source
func (r *cryptoNewsRepository) GetBySource(source string, limit int) ([]CryptoNews, error) {
	var news []CryptoNews
	err := r.db.Where("source = ?", source).
		Order("time DESC").
		Limit(limit).
		Find(&news).Error
	return news, err
}

// GetBySentiment récupère les actualités par score de sentiment
func (r *cryptoNewsRepository) GetBySentiment(minScore, maxScore float64, limit int) ([]CryptoNews, error) {
	var news []CryptoNews
	err := r.db.Where("sentiment_score BETWEEN ? AND ?", minScore, maxScore).
		Order("time DESC").
		Limit(limit).
		Find(&news).Error
	return news, err
}

// GetByCompositeKey récupère une actualité par sa clé composite
func (r *cryptoNewsRepository) GetByCompositeKey(time time.Time, source, url string) (*CryptoNews, error) {
	var news CryptoNews
	err := r.db.Where("time = ? AND source = ? AND url = ?", time, source, url).First(&news).Error
	if err != nil {
		return nil, err
	}
	return &news, nil
}

// Update met à jour une actualité
func (r *cryptoNewsRepository) Update(news *CryptoNews) error {
	return r.db.Save(news).Error
}

// DeleteByCompositeKey supprime une actualité par sa clé composite
func (r *cryptoNewsRepository) DeleteByCompositeKey(time time.Time, source, url string) error {
	return r.db.Where("time = ? AND source = ? AND url = ?", time, source, url).Delete(&CryptoNews{}).Error
}
