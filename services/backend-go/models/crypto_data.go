package models

import (
	"time"

	"gorm.io/gorm"
)

// CryptoData représente les données de prix crypto en temps réel
type CryptoData struct {
	Time         time.Time `json:"time" gorm:"primaryKey;column:time"`
	Symbol       string    `json:"symbol" gorm:"primaryKey;column:symbol;size:20"`
	IntervalType string    `json:"interval_type" gorm:"primaryKey;column:interval_type;size:10"`
	Open         float64   `json:"open" gorm:"column:open;type:decimal(20,8)"`
	High         float64   `json:"high" gorm:"column:high;type:decimal(20,8)"`
	Low          float64   `json:"low" gorm:"column:low;type:decimal(20,8)"`
	Close        float64   `json:"close" gorm:"column:close;type:decimal(20,8)"`
	Volume       float64   `json:"volume" gorm:"column:volume;type:decimal(20,8)"`
	CreatedAt    time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt    time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName spécifie le nom de la table pour GORM
func (CryptoData) TableName() string {
	return "crypto_data"
}

// BeforeCreate hook GORM appelé avant la création
func (cd *CryptoData) BeforeCreate(tx *gorm.DB) error {
	if cd.Time.IsZero() {
		cd.Time = time.Now()
	}
	return nil
}

// CryptoDataRepository interface pour les opérations sur CryptoData
type CryptoDataRepository interface {
	Create(data *CryptoData) error
	GetBySymbol(symbol string, interval string, limit int) ([]CryptoData, error)
	GetLatest(symbol string) (*CryptoData, error)
	GetByTimeRange(symbol string, interval string, start, end time.Time) ([]CryptoData, error)
	GetStats(symbol string, interval string) (*CryptoStats, error)
}

// CryptoStats représente les statistiques d'un symbole crypto
type CryptoStats struct {
	Symbol         string    `json:"symbol"`
	IntervalType   string    `json:"interval_type"`
	LatestPrice    *float64  `json:"latest_price"`
	PriceChange24h *float64  `json:"price_change_24h"`
	PriceChangePct *float64  `json:"price_change_pct_24h"`
	Volume24h      *float64  `json:"volume_24h"`
	High24h        *float64  `json:"high_24h"`
	Low24h         *float64  `json:"low_24h"`
	LastUpdate     time.Time `json:"last_update"`
}

// cryptoDataRepository implémentation concrète du repository
type cryptoDataRepository struct {
	db *gorm.DB
}

// NewCryptoDataRepository crée une nouvelle instance du repository
func NewCryptoDataRepository(db *gorm.DB) CryptoDataRepository {
	return &cryptoDataRepository{db: db}
}

// Create insère une nouvelle donnée crypto
func (r *cryptoDataRepository) Create(data *CryptoData) error {
	return r.db.Create(data).Error
}

// GetBySymbol récupère les données par symbole et intervalle
func (r *cryptoDataRepository) GetBySymbol(symbol string, interval string, limit int) ([]CryptoData, error) {
	var data []CryptoData
	err := r.db.Where("symbol = ? AND interval_type = ?", symbol, interval).
		Order("time DESC").
		Limit(limit).
		Find(&data).Error
	return data, err
}

// GetLatest récupère la dernière donnée pour un symbole
func (r *cryptoDataRepository) GetLatest(symbol string) (*CryptoData, error) {
	var data CryptoData
	err := r.db.Where("symbol = ?", symbol).
		Order("time DESC").
		First(&data).Error
	if err != nil {
		return nil, err
	}
	return &data, nil
}

// GetByTimeRange récupère les données dans une plage de temps
func (r *cryptoDataRepository) GetByTimeRange(symbol string, interval string, start, end time.Time) ([]CryptoData, error) {
	var data []CryptoData
	err := r.db.Where("symbol = ? AND interval_type = ? AND time BETWEEN ? AND ?",
		symbol, interval, start, end).
		Order("time ASC").
		Find(&data).Error
	return data, err
}

// GetStats calcule les statistiques pour un symbole
func (r *cryptoDataRepository) GetStats(symbol string, interval string) (*CryptoStats, error) {
	var stats CryptoStats

	// Utiliser une requête SQL brute pour les statistiques complexes
	// car GORM n'est pas optimal pour ce type de calculs
	query := `
		SELECT
			$1 as symbol,
			$2 as interval_type,
			(SELECT close FROM crypto_data WHERE symbol = $1 AND interval_type = $2 ORDER BY time DESC LIMIT 1) as latest_price,
			(SELECT close FROM crypto_data WHERE symbol = $1 AND interval_type = $2 ORDER BY time DESC LIMIT 1) -
			(SELECT close FROM crypto_data WHERE symbol = $1 AND interval_type = $2 AND time >= NOW() - INTERVAL '24 hours' ORDER BY time ASC LIMIT 1) as price_change_24h,
			((SELECT close FROM crypto_data WHERE symbol = $1 AND interval_type = $2 ORDER BY time DESC LIMIT 1) -
			(SELECT close FROM crypto_data WHERE symbol = $1 AND interval_type = $2 AND time >= NOW() - INTERVAL '24 hours' ORDER BY time ASC LIMIT 1)) /
			NULLIF((SELECT close FROM crypto_data WHERE symbol = $1 AND interval_type = $2 AND time >= NOW() - INTERVAL '24 hours' ORDER BY time ASC LIMIT 1), 0) * 100 as price_change_pct_24h,
			(SELECT SUM(volume) FROM crypto_data WHERE symbol = $1 AND interval_type = $2 AND time >= NOW() - INTERVAL '24 hours') as volume_24h,
			(SELECT MAX(high) FROM crypto_data WHERE symbol = $1 AND interval_type = $2 AND time >= NOW() - INTERVAL '24 hours') as high_24h,
			(SELECT MIN(low) FROM crypto_data WHERE symbol = $1 AND interval_type = $2 AND time >= NOW() - INTERVAL '24 hours') as low_24h,
			NOW() as last_update
	`

	err := r.db.Raw(query, symbol, interval).Scan(&stats).Error
	if err != nil {
		return nil, err
	}

	return &stats, nil
}
