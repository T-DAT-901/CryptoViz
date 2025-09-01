package models

import (
	"time"

	"gorm.io/gorm"
)

// TechnicalIndicator représente les indicateurs techniques
type TechnicalIndicator struct {
	Time          time.Time `json:"time" gorm:"primaryKey;column:time"`
	Symbol        string    `json:"symbol" gorm:"primaryKey;column:symbol;size:20"`
	IntervalType  string    `json:"interval_type" gorm:"primaryKey;column:interval_type;size:10"`
	IndicatorType string    `json:"indicator_type" gorm:"primaryKey;column:indicator_type;size:20"`
	Value         *float64  `json:"value" gorm:"column:value;type:decimal(20,8)"`
	ValueSignal   *float64  `json:"value_signal" gorm:"column:value_signal;type:decimal(20,8)"`
	ValueHisto    *float64  `json:"value_histogram" gorm:"column:value_histogram;type:decimal(20,8)"`
	UpperBand     *float64  `json:"upper_band" gorm:"column:upper_band;type:decimal(20,8)"`
	LowerBand     *float64  `json:"lower_band" gorm:"column:lower_band;type:decimal(20,8)"`
	MiddleBand    *float64  `json:"middle_band" gorm:"column:middle_band;type:decimal(20,8)"`
	CreatedAt     time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt     time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName spécifie le nom de la table pour GORM
func (TechnicalIndicator) TableName() string {
	return "crypto_indicators"
}

// BeforeCreate hook GORM appelé avant la création
func (ti *TechnicalIndicator) BeforeCreate(tx *gorm.DB) error {
	if ti.Time.IsZero() {
		ti.Time = time.Now()
	}
	return nil
}

// TechnicalIndicatorRepository interface pour les opérations sur TechnicalIndicator
type TechnicalIndicatorRepository interface {
	Create(indicator *TechnicalIndicator) error
	GetBySymbolAndType(symbol, indicatorType, interval string, limit int) ([]TechnicalIndicator, error)
	GetAllBySymbol(symbol, interval string) ([]TechnicalIndicator, error)
	GetLatestByType(symbol, indicatorType, interval string) (*TechnicalIndicator, error)
	GetByTimeRange(symbol, indicatorType, interval string, start, end time.Time) ([]TechnicalIndicator, error)
}

// technicalIndicatorRepository implémentation concrète du repository
type technicalIndicatorRepository struct {
	db *gorm.DB
}

// NewTechnicalIndicatorRepository crée une nouvelle instance du repository
func NewTechnicalIndicatorRepository(db *gorm.DB) TechnicalIndicatorRepository {
	return &technicalIndicatorRepository{db: db}
}

// Create insère un nouvel indicateur technique
func (r *technicalIndicatorRepository) Create(indicator *TechnicalIndicator) error {
	return r.db.Create(indicator).Error
}

// GetBySymbolAndType récupère les indicateurs par symbole et type
func (r *technicalIndicatorRepository) GetBySymbolAndType(symbol, indicatorType, interval string, limit int) ([]TechnicalIndicator, error) {
	var indicators []TechnicalIndicator
	err := r.db.Where("symbol = ? AND indicator_type = ? AND interval_type = ?", symbol, indicatorType, interval).
		Order("time DESC").
		Limit(limit).
		Find(&indicators).Error
	return indicators, err
}

// GetAllBySymbol récupère tous les indicateurs pour un symbole (dernière valeur de chaque type)
func (r *technicalIndicatorRepository) GetAllBySymbol(symbol, interval string) ([]TechnicalIndicator, error) {
	var indicators []TechnicalIndicator

	// Utiliser une sous-requête pour obtenir la dernière valeur de chaque type d'indicateur
	subQuery := r.db.Model(&TechnicalIndicator{}).
		Select("symbol, indicator_type, interval_type, MAX(time) as max_time").
		Where("symbol = ? AND interval_type = ?", symbol, interval).
		Group("symbol, indicator_type, interval_type")

	err := r.db.Table("crypto_indicators").
		Joins("INNER JOIN (?) as latest ON crypto_indicators.symbol = latest.symbol AND crypto_indicators.indicator_type = latest.indicator_type AND crypto_indicators.interval_type = latest.interval_type AND crypto_indicators.time = latest.max_time", subQuery).
		Where("crypto_indicators.symbol = ? AND crypto_indicators.interval_type = ?", symbol, interval).
		Find(&indicators).Error

	return indicators, err
}

// GetLatestByType récupère le dernier indicateur d'un type spécifique
func (r *technicalIndicatorRepository) GetLatestByType(symbol, indicatorType, interval string) (*TechnicalIndicator, error) {
	var indicator TechnicalIndicator
	err := r.db.Where("symbol = ? AND indicator_type = ? AND interval_type = ?", symbol, indicatorType, interval).
		Order("time DESC").
		First(&indicator).Error
	if err != nil {
		return nil, err
	}
	return &indicator, nil
}

// GetByTimeRange récupère les indicateurs dans une plage de temps
func (r *technicalIndicatorRepository) GetByTimeRange(symbol, indicatorType, interval string, start, end time.Time) ([]TechnicalIndicator, error) {
	var indicators []TechnicalIndicator
	err := r.db.Where("symbol = ? AND indicator_type = ? AND interval_type = ? AND time BETWEEN ? AND ?",
		symbol, indicatorType, interval, start, end).
		Order("time ASC").
		Find(&indicators).Error
	return indicators, err
}
