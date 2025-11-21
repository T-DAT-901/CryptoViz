package models

import (
	"time"

	"gorm.io/gorm"
)

// Indicator représente les indicateurs techniques
// Table: indicators (hypertable TimescaleDB)
type Indicator struct {
	Time               time.Time `json:"time" gorm:"column:time;not null"`
	Symbol             string    `json:"symbol" gorm:"column:symbol;size:20;not null"`
	Timeframe          string    `json:"timeframe" gorm:"column:timeframe;size:5;not null"` // '1m', '15m', '1h', etc.
	IndicatorType      string    `json:"indicator_type" gorm:"column:indicator_type;size:20;not null"` // 'rsi', 'macd', 'bollinger', 'momentum', 'support_resistance'
	Value              *float64  `json:"value,omitempty" gorm:"column:value;type:decimal(20,8)"`
	ValueSignal        *float64  `json:"value_signal,omitempty" gorm:"column:value_signal;type:decimal(20,8)"` // For MACD signal
	ValueHistogram     *float64  `json:"value_histogram,omitempty" gorm:"column:value_histogram;type:decimal(20,8)"` // For MACD histogram
	UpperBand          *float64  `json:"upper_band,omitempty" gorm:"column:upper_band;type:decimal(20,8)"` // For Bollinger upper
	LowerBand          *float64  `json:"lower_band,omitempty" gorm:"column:lower_band;type:decimal(20,8)"` // For Bollinger lower
	MiddleBand         *float64  `json:"middle_band,omitempty" gorm:"column:middle_band;type:decimal(20,8)"` // For Bollinger middle
	SupportLevel       *float64  `json:"support_level,omitempty" gorm:"column:support_level;type:decimal(20,8)"` // Support price level
	ResistanceLevel    *float64  `json:"resistance_level,omitempty" gorm:"column:resistance_level;type:decimal(20,8)"` // Resistance price level
	SupportStrength    *float64  `json:"support_strength,omitempty" gorm:"column:support_strength;type:decimal(5,2)"` // Support strength score (0-100)
	ResistanceStrength *float64  `json:"resistance_strength,omitempty" gorm:"column:resistance_strength;type:decimal(5,2)"` // Resistance strength score (0-100)
	CreatedAt          time.Time `json:"created_at" gorm:"column:created_at;autoCreateTime"`
}

// TableName spécifie le nom de la table pour GORM
func (Indicator) TableName() string {
	return "indicators"
}

// AllIndicator représente une requête sur la vue unifiée (hot + cold storage)
// View: all_indicators (transparent tiering)
type AllIndicator struct {
	Indicator
}

// TableName spécifie le nom de la vue pour GORM
func (AllIndicator) TableName() string {
	return "all_indicators"
}

// IndicatorLatest représente les derniers indicateurs (continuous aggregate)
// Materialized View: indicators_latest
type IndicatorLatest struct {
	Bucket             time.Time `json:"bucket" gorm:"column:bucket;not null"`
	Symbol             string    `json:"symbol" gorm:"column:symbol;size:20;not null"`
	IndicatorType      string    `json:"indicator_type" gorm:"column:indicator_type;size:20;not null"`
	Timeframe          string    `json:"timeframe" gorm:"column:timeframe;size:5;not null"`
	Value              *float64  `json:"value,omitempty" gorm:"column:value;type:decimal(20,8)"`
	ValueSignal        *float64  `json:"value_signal,omitempty" gorm:"column:value_signal;type:decimal(20,8)"`
	ValueHistogram     *float64  `json:"value_histogram,omitempty" gorm:"column:value_histogram;type:decimal(20,8)"`
	UpperBand          *float64  `json:"upper_band,omitempty" gorm:"column:upper_band;type:decimal(20,8)"`
	LowerBand          *float64  `json:"lower_band,omitempty" gorm:"column:lower_band;type:decimal(20,8)"`
	MiddleBand         *float64  `json:"middle_band,omitempty" gorm:"column:middle_band;type:decimal(20,8)"`
	SupportLevel       *float64  `json:"support_level,omitempty" gorm:"column:support_level;type:decimal(20,8)"`
	ResistanceLevel    *float64  `json:"resistance_level,omitempty" gorm:"column:resistance_level;type:decimal(20,8)"`
	SupportStrength    *float64  `json:"support_strength,omitempty" gorm:"column:support_strength;type:decimal(5,2)"`
	ResistanceStrength *float64  `json:"resistance_strength,omitempty" gorm:"column:resistance_strength;type:decimal(5,2)"`
}

// TableName spécifie le nom de la vue matérialisée pour GORM
func (IndicatorLatest) TableName() string {
	return "indicators_latest"
}

// IndicatorRepository interface pour les opérations sur Indicator
type IndicatorRepository interface {
	Create(indicator *Indicator) error
	GetBySymbolAndType(symbol, indicatorType, timeframe string, limit int) ([]Indicator, error)
	GetAllBySymbol(symbol, timeframe string) ([]Indicator, error)
	GetLatestByType(symbol, indicatorType, timeframe string) (*Indicator, error)
	GetByTimeRange(symbol, indicatorType, timeframe string, start, end time.Time) ([]Indicator, error)

	// Queries on unified view (hot + cold storage)
	GetAllBySymbolUnified(symbol, timeframe string) ([]AllIndicator, error)
	GetAllByTimeRangeUnified(symbol, indicatorType, timeframe string, start, end time.Time) ([]AllIndicator, error)

	// Latest indicators queries (continuous aggregate)
	GetLatestIndicators(symbol, timeframe string) ([]IndicatorLatest, error)
}

// indicatorRepository implémentation concrète du repository
type indicatorRepository struct {
	db *gorm.DB
}

// NewIndicatorRepository crée une nouvelle instance du repository
func NewIndicatorRepository(db *gorm.DB) IndicatorRepository {
	return &indicatorRepository{db: db}
}

// Create insère un nouvel indicateur technique
func (r *indicatorRepository) Create(indicator *Indicator) error {
	return r.db.Create(indicator).Error
}

// GetBySymbolAndType récupère les indicateurs par symbole et type (hot storage only)
func (r *indicatorRepository) GetBySymbolAndType(symbol, indicatorType, timeframe string, limit int) ([]Indicator, error) {
	var indicators []Indicator
	err := r.db.Where("symbol = ? AND indicator_type = ? AND timeframe = ?", symbol, indicatorType, timeframe).
		Order("time DESC").
		Limit(limit).
		Find(&indicators).Error
	return indicators, err
}

// GetAllBySymbol récupère tous les indicateurs pour un symbole (dernière valeur de chaque type)
func (r *indicatorRepository) GetAllBySymbol(symbol, timeframe string) ([]Indicator, error) {
	var indicators []Indicator

	// Utiliser une sous-requête pour obtenir la dernière valeur de chaque type d'indicateur
	subQuery := r.db.Model(&Indicator{}).
		Select("symbol, indicator_type, timeframe, MAX(time) as max_time").
		Where("symbol = ? AND timeframe = ?", symbol, timeframe).
		Group("symbol, indicator_type, timeframe")

	err := r.db.Table("indicators").
		Joins("INNER JOIN (?) as latest ON indicators.symbol = latest.symbol AND indicators.indicator_type = latest.indicator_type AND indicators.timeframe = latest.timeframe AND indicators.time = latest.max_time", subQuery).
		Where("indicators.symbol = ? AND indicators.timeframe = ?", symbol, timeframe).
		Find(&indicators).Error

	return indicators, err
}

// GetLatestByType récupère le dernier indicateur d'un type spécifique
func (r *indicatorRepository) GetLatestByType(symbol, indicatorType, timeframe string) (*Indicator, error) {
	var indicator Indicator
	err := r.db.Where("symbol = ? AND indicator_type = ? AND timeframe = ?", symbol, indicatorType, timeframe).
		Order("time DESC").
		First(&indicator).Error
	if err != nil {
		return nil, err
	}
	return &indicator, nil
}

// GetByTimeRange récupère les indicateurs dans une plage de temps (hot storage only)
func (r *indicatorRepository) GetByTimeRange(symbol, indicatorType, timeframe string, start, end time.Time) ([]Indicator, error) {
	var indicators []Indicator
	err := r.db.Where("symbol = ? AND indicator_type = ? AND timeframe = ? AND time BETWEEN ? AND ?",
		symbol, indicatorType, timeframe, start, end).
		Order("time ASC").
		Find(&indicators).Error
	return indicators, err
}

// GetAllBySymbolUnified récupère tous les indicateurs (hot + cold storage via vue unifiée)
func (r *indicatorRepository) GetAllBySymbolUnified(symbol, timeframe string) ([]AllIndicator, error) {
	var indicators []AllIndicator

	// Query on unified view
	subQuery := r.db.Model(&AllIndicator{}).
		Select("symbol, indicator_type, timeframe, MAX(time) as max_time").
		Where("symbol = ? AND timeframe = ?", symbol, timeframe).
		Group("symbol, indicator_type, timeframe")

	err := r.db.Table("all_indicators").
		Joins("INNER JOIN (?) as latest ON all_indicators.symbol = latest.symbol AND all_indicators.indicator_type = latest.indicator_type AND all_indicators.timeframe = latest.timeframe AND all_indicators.time = latest.max_time", subQuery).
		Where("all_indicators.symbol = ? AND all_indicators.timeframe = ?", symbol, timeframe).
		Find(&indicators).Error

	return indicators, err
}

// GetAllByTimeRangeUnified récupère les indicateurs dans une plage de temps (hot + cold storage)
func (r *indicatorRepository) GetAllByTimeRangeUnified(symbol, indicatorType, timeframe string, start, end time.Time) ([]AllIndicator, error) {
	var indicators []AllIndicator
	err := r.db.Where("symbol = ? AND indicator_type = ? AND timeframe = ? AND time BETWEEN ? AND ?",
		symbol, indicatorType, timeframe, start, end).
		Order("time ASC").
		Find(&indicators).Error
	return indicators, err
}

// GetLatestIndicators récupère les derniers indicateurs via continuous aggregate
func (r *indicatorRepository) GetLatestIndicators(symbol, timeframe string) ([]IndicatorLatest, error) {
	var indicators []IndicatorLatest

	// Query the continuous aggregate for latest values
	// Get the most recent bucket for each indicator type
	subQuery := r.db.Model(&IndicatorLatest{}).
		Select("symbol, indicator_type, timeframe, MAX(bucket) as max_bucket").
		Where("symbol = ? AND timeframe = ?", symbol, timeframe).
		Group("symbol, indicator_type, timeframe")

	err := r.db.Table("indicators_latest").
		Joins("INNER JOIN (?) as latest ON indicators_latest.symbol = latest.symbol AND indicators_latest.indicator_type = latest.indicator_type AND indicators_latest.timeframe = latest.timeframe AND indicators_latest.bucket = latest.max_bucket", subQuery).
		Where("indicators_latest.symbol = ? AND indicators_latest.timeframe = ?", symbol, timeframe).
		Find(&indicators).Error

	return indicators, err
}
