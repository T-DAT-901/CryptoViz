package models

import (
	"time"

	"gorm.io/gorm"
)

// Trade représente un trade individuel (haute fréquence)
// Table: trades (hypertable TimescaleDB)
type Trade struct {
	EventTs   time.Time `json:"event_ts" gorm:"column:event_ts;not null"`
	Exchange  string    `json:"exchange" gorm:"column:exchange;size:20;not null"`
	Symbol    string    `json:"symbol" gorm:"column:symbol;size:20;not null"`
	TradeID   string    `json:"trade_id" gorm:"column:trade_id;size:50;not null"`
	Price     float64   `json:"price" gorm:"column:price;type:decimal(20,8);not null"`
	Amount    float64   `json:"amount" gorm:"column:amount;type:decimal(20,8);not null"`
	Side      string    `json:"side" gorm:"column:side;size:4;not null"` // 'buy' or 'sell'
	CreatedAt time.Time `json:"created_at" gorm:"column:created_at;autoCreateTime"`
}

// TableName spécifie le nom de la table pour GORM
func (Trade) TableName() string {
	return "trades"
}

// TradeRepository interface pour les opérations sur Trade
type TradeRepository interface {
	Create(trade *Trade) error
	CreateBatch(trades []*Trade) error
	GetBySymbol(symbol string, limit int) ([]Trade, error)
	GetByExchangeAndSymbol(exchange string, symbol string, limit int) ([]Trade, error)
	GetByTimeRange(symbol string, start, end time.Time) ([]Trade, error)
	GetLatestTrades(limit int) ([]Trade, error)
}

// tradeRepository implémentation concrète du repository
type tradeRepository struct {
	db *gorm.DB
}

// NewTradeRepository crée une nouvelle instance du repository
func NewTradeRepository(db *gorm.DB) TradeRepository {
	return &tradeRepository{db: db}
}

// Create insère un nouveau trade avec UPSERT (idempotent)
// Utilise ON CONFLICT DO NOTHING pour ignorer les duplications
func (r *tradeRepository) Create(trade *Trade) error {
	// UPSERT avec ON CONFLICT DO NOTHING
	// Unique constraint: (trade_id, exchange, symbol, event_ts)
	// Si le trade existe déjà (même trade_id), on l'ignore
	query := `
		INSERT INTO trades (
			event_ts, exchange, symbol, trade_id,
			price, amount, side, created_at
		) VALUES (
			?, ?, ?, ?,
			?, ?, ?, ?
		)
		ON CONFLICT (trade_id, exchange, symbol, event_ts) DO NOTHING
	`

	return r.db.Exec(query,
		trade.EventTs, trade.Exchange, trade.Symbol, trade.TradeID,
		trade.Price, trade.Amount, trade.Side, trade.CreatedAt,
	).Error
}

// CreateBatch insère plusieurs trades en une seule transaction
func (r *tradeRepository) CreateBatch(trades []*Trade) error {
	if len(trades) == 0 {
		return nil
	}

	// Use transaction for batch insert
	return r.db.Transaction(func(tx *gorm.DB) error {
		for _, trade := range trades {
			query := `
				INSERT INTO trades (
					event_ts, exchange, symbol, trade_id,
					price, amount, side, created_at
				) VALUES (
					?, ?, ?, ?,
					?, ?, ?, ?
				)
				ON CONFLICT (trade_id, exchange, symbol, event_ts) DO NOTHING
			`
			if err := tx.Exec(query,
				trade.EventTs, trade.Exchange, trade.Symbol, trade.TradeID,
				trade.Price, trade.Amount, trade.Side, trade.CreatedAt,
			).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// GetBySymbol récupère les trades par symbole
func (r *tradeRepository) GetBySymbol(symbol string, limit int) ([]Trade, error) {
	var trades []Trade
	err := r.db.Where("symbol = ?", symbol).
		Order("event_ts DESC").
		Limit(limit).
		Find(&trades).Error
	return trades, err
}

// GetByExchangeAndSymbol récupère les trades par exchange et symbole
func (r *tradeRepository) GetByExchangeAndSymbol(exchange string, symbol string, limit int) ([]Trade, error) {
	var trades []Trade
	err := r.db.Where("exchange = ? AND symbol = ?", exchange, symbol).
		Order("event_ts DESC").
		Limit(limit).
		Find(&trades).Error
	return trades, err
}

// GetByTimeRange récupère les trades dans une plage de temps
func (r *tradeRepository) GetByTimeRange(symbol string, start, end time.Time) ([]Trade, error) {
	var trades []Trade
	err := r.db.Where("symbol = ? AND event_ts BETWEEN ? AND ?", symbol, start, end).
		Order("event_ts ASC").
		Find(&trades).Error
	return trades, err
}

// GetLatestTrades récupère les derniers trades toutes paires confondues
func (r *tradeRepository) GetLatestTrades(limit int) ([]Trade, error) {
	var trades []Trade
	err := r.db.Order("event_ts DESC").
		Limit(limit).
		Find(&trades).Error
	return trades, err
}
