package models

import (
	"time"

	"gorm.io/gorm"
)

// Candle représente les données OHLCV (bougies) agrégées
// Table: candles (hypertable TimescaleDB)
type Candle struct {
	WindowStart   time.Time `json:"window_start" gorm:"column:window_start;not null"`
	WindowEnd     time.Time `json:"window_end" gorm:"column:window_end;not null"`
	Exchange      string    `json:"exchange" gorm:"column:exchange;size:20;not null"`
	Symbol        string    `json:"symbol" gorm:"column:symbol;size:20;not null"`
	Timeframe     string    `json:"timeframe" gorm:"column:timeframe;size:5;not null"` // '1m', '5m', '15m', '1h', '1d'
	Open          float64   `json:"open" gorm:"column:open;type:decimal(20,8);not null"`
	High          float64   `json:"high" gorm:"column:high;type:decimal(20,8);not null"`
	Low           float64   `json:"low" gorm:"column:low;type:decimal(20,8);not null"`
	Close         float64   `json:"close" gorm:"column:close;type:decimal(20,8);not null"`
	Volume        float64   `json:"volume" gorm:"column:volume;type:decimal(20,8);not null"`
	TradeCount    *int      `json:"trade_count" gorm:"column:trade_count"`
	Closed        bool      `json:"closed" gorm:"column:closed;not null"`
	FirstTradeTs  *time.Time `json:"first_trade_ts" gorm:"column:first_trade_ts"`
	LastTradeTs   *time.Time `json:"last_trade_ts" gorm:"column:last_trade_ts"`
	DurationMs    *int64    `json:"duration_ms" gorm:"column:duration_ms"`
	Source        *string   `json:"source" gorm:"column:source;size:20"` // 'collector-ws' or 'collector-historical'
	CreatedAt     time.Time `json:"created_at" gorm:"column:created_at;autoCreateTime"`
}

// TableName spécifie le nom de la table pour GORM
func (Candle) TableName() string {
	return "candles"
}

// AllCandle représente une requête sur la vue unifiée (hot + cold storage)
// View: all_candles (transparent tiering)
type AllCandle struct {
	Candle
}

// TableName spécifie le nom de la vue pour GORM
func (AllCandle) TableName() string {
	return "all_candles"
}

// CandleHourlySummary représente l'agrégation horaire des candles
// Materialized View: candles_hourly_summary (continuous aggregate)
type CandleHourlySummary struct {
	Hour        time.Time `json:"hour" gorm:"column:hour;not null"`
	Exchange    string    `json:"exchange" gorm:"column:exchange;size:20;not null"`
	Symbol      string    `json:"symbol" gorm:"column:symbol;size:20;not null"`
	Open        float64   `json:"open" gorm:"column:open;type:decimal(20,8)"`
	High        float64   `json:"high" gorm:"column:high;type:decimal(20,8)"`
	Low         float64   `json:"low" gorm:"column:low;type:decimal(20,8)"`
	Close       float64   `json:"close" gorm:"column:close;type:decimal(20,8)"`
	Volume      float64   `json:"volume" gorm:"column:volume;type:decimal(20,8)"`
	TradesCount *int64    `json:"trades_count" gorm:"column:trades_count"`
}

// TableName spécifie le nom de la vue matérialisée pour GORM
func (CandleHourlySummary) TableName() string {
	return "candles_hourly_summary"
}

// CandleStats représente les statistiques d'un symbole crypto
type CandleStats struct {
	Symbol         string    `json:"symbol"`
	Timeframe      string    `json:"timeframe"`
	LatestPrice    *float64  `json:"latest_price"`
	PriceChange24h *float64  `json:"price_change_24h"`
	PriceChangePct *float64  `json:"price_change_pct_24h"`
	Volume24h      *float64  `json:"volume_24h"`
	High24h        *float64  `json:"high_24h"`
	Low24h         *float64  `json:"low_24h"`
	LastUpdate     time.Time `json:"last_update"`
}

// CandleRepository interface pour les opérations sur Candle
type CandleRepository interface {
	Create(candle *Candle) error
	CreateBatch(candles []*Candle) error
	GetBySymbol(symbol string, timeframe string, limit int) ([]Candle, error)
	GetLatest(symbol string, exchange string) (*Candle, error)
	GetByTimeRange(symbol string, timeframe string, start, end time.Time) ([]Candle, error)
	GetStats(symbol string, timeframe string) (*CandleStats, error)
	GetDistinctSymbols() ([]string, error)

	// Queries on unified view (hot + cold storage)
	GetAllBySymbol(symbol string, timeframe string, limit int) ([]AllCandle, error)
	GetAllByTimeRange(symbol string, timeframe string, start, end time.Time) ([]AllCandle, error)

	// Hourly summary queries (continuous aggregate)
	GetHourlySummary(symbol string, exchange string, limit int) ([]CandleHourlySummary, error)
}

// candleRepository implémentation concrète du repository
type candleRepository struct {
	db *gorm.DB
}

// NewCandleRepository crée une nouvelle instance du repository
func NewCandleRepository(db *gorm.DB) CandleRepository {
	return &candleRepository{db: db}
}

// Create insère une nouvelle candle avec UPSERT (idempotent)
// Utilise ON CONFLICT pour prévenir les duplications
func (r *candleRepository) Create(candle *Candle) error {
	// UPSERT avec ON CONFLICT pour gérer les duplications
	// Unique constraint: (window_start, exchange, symbol, timeframe)
	query := `
		INSERT INTO candles (
			window_start, window_end, exchange, symbol, timeframe,
			open, high, low, close, volume,
			trade_count, closed, first_trade_ts, last_trade_ts, duration_ms, source, created_at
		) VALUES (
			?, ?, ?, ?, ?,
			?, ?, ?, ?, ?,
			?, ?, ?, ?, ?, ?, ?
		)
		ON CONFLICT (window_start, exchange, symbol, timeframe) DO UPDATE SET
			-- For open candles, merge data (take best values)
			high = CASE
				WHEN NOT candles.closed THEN GREATEST(candles.high, EXCLUDED.high)
				ELSE EXCLUDED.high
			END,
			low = CASE
				WHEN NOT candles.closed THEN LEAST(candles.low, EXCLUDED.low)
				ELSE EXCLUDED.low
			END,
			close = EXCLUDED.close,
			volume = CASE
				WHEN NOT candles.closed THEN candles.volume + EXCLUDED.volume
				ELSE EXCLUDED.volume
			END,
			trade_count = CASE
				WHEN NOT candles.closed THEN COALESCE(candles.trade_count, 0) + COALESCE(EXCLUDED.trade_count, 0)
				ELSE EXCLUDED.trade_count
			END,
			closed = EXCLUDED.closed,
			last_trade_ts = EXCLUDED.last_trade_ts,
			window_end = EXCLUDED.window_end,
			duration_ms = EXCLUDED.duration_ms
		WHERE NOT candles.closed OR EXCLUDED.closed
	`

	return r.db.Exec(query,
		candle.WindowStart, candle.WindowEnd, candle.Exchange, candle.Symbol, candle.Timeframe,
		candle.Open, candle.High, candle.Low, candle.Close, candle.Volume,
		candle.TradeCount, candle.Closed, candle.FirstTradeTs, candle.LastTradeTs,
		candle.DurationMs, candle.Source, candle.CreatedAt,
	).Error
}

// CreateBatch insère plusieurs candles en une seule transaction
func (r *candleRepository) CreateBatch(candles []*Candle) error {
	if len(candles) == 0 {
		return nil
	}

	// Use transaction for batch insert
	return r.db.Transaction(func(tx *gorm.DB) error {
		for _, candle := range candles {
			query := `
				INSERT INTO candles (
					window_start, window_end, exchange, symbol, timeframe,
					open, high, low, close, volume,
					trade_count, closed, first_trade_ts, last_trade_ts, duration_ms, source, created_at
				) VALUES (
					?, ?, ?, ?, ?,
					?, ?, ?, ?, ?,
					?, ?, ?, ?, ?, ?, ?
				)
				ON CONFLICT (window_start, exchange, symbol, timeframe) DO UPDATE SET
					high = CASE
						WHEN NOT candles.closed THEN GREATEST(candles.high, EXCLUDED.high)
						ELSE EXCLUDED.high
					END,
					low = CASE
						WHEN NOT candles.closed THEN LEAST(candles.low, EXCLUDED.low)
						ELSE EXCLUDED.low
					END,
					close = EXCLUDED.close,
					volume = CASE
						WHEN NOT candles.closed THEN candles.volume + EXCLUDED.volume
						ELSE EXCLUDED.volume
					END,
					trade_count = CASE
						WHEN NOT candles.closed THEN COALESCE(candles.trade_count, 0) + COALESCE(EXCLUDED.trade_count, 0)
						ELSE EXCLUDED.trade_count
					END,
					closed = EXCLUDED.closed,
					last_trade_ts = EXCLUDED.last_trade_ts,
					window_end = EXCLUDED.window_end,
					duration_ms = EXCLUDED.duration_ms
				WHERE NOT candles.closed OR EXCLUDED.closed
			`
			if err := tx.Exec(query,
				candle.WindowStart, candle.WindowEnd, candle.Exchange, candle.Symbol, candle.Timeframe,
				candle.Open, candle.High, candle.Low, candle.Close, candle.Volume,
				candle.TradeCount, candle.Closed, candle.FirstTradeTs, candle.LastTradeTs,
				candle.DurationMs, candle.Source, candle.CreatedAt,
			).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// GetBySymbol récupère les candles par symbole et timeframe (hot storage only)
func (r *candleRepository) GetBySymbol(symbol string, timeframe string, limit int) ([]Candle, error) {
	var candles []Candle
	err := r.db.Where("symbol = ? AND timeframe = ?", symbol, timeframe).
		Order("window_start DESC").
		Limit(limit).
		Find(&candles).Error
	return candles, err
}

// GetLatest récupère la dernière candle pour un symbole
func (r *candleRepository) GetLatest(symbol string, exchange string) (*Candle, error) {
	var candle Candle
	err := r.db.Where("symbol = ? AND exchange = ?", symbol, exchange).
		Order("window_start DESC").
		First(&candle).Error
	if err != nil {
		return nil, err
	}
	return &candle, nil
}

// GetByTimeRange récupère les candles dans une plage de temps (hot storage only)
func (r *candleRepository) GetByTimeRange(symbol string, timeframe string, start, end time.Time) ([]Candle, error) {
	var candles []Candle
	err := r.db.Where("symbol = ? AND timeframe = ? AND window_start BETWEEN ? AND ?",
		symbol, timeframe, start, end).
		Order("window_start ASC").
		Find(&candles).Error
	return candles, err
}

// GetAllBySymbol récupère les candles par symbole (hot + cold storage via vue unifiée)
func (r *candleRepository) GetAllBySymbol(symbol string, timeframe string, limit int) ([]AllCandle, error) {
	var candles []AllCandle
	err := r.db.Where("symbol = ? AND timeframe = ?", symbol, timeframe).
		Order("window_start DESC").
		Limit(limit).
		Find(&candles).Error
	return candles, err
}

// GetAllByTimeRange récupère les candles dans une plage de temps (hot + cold storage)
func (r *candleRepository) GetAllByTimeRange(symbol string, timeframe string, start, end time.Time) ([]AllCandle, error) {
	var candles []AllCandle
	err := r.db.Where("symbol = ? AND timeframe = ? AND window_start BETWEEN ? AND ?",
		symbol, timeframe, start, end).
		Order("window_start ASC").
		Find(&candles).Error
	return candles, err
}

// GetHourlySummary récupère l'agrégation horaire (continuous aggregate)
func (r *candleRepository) GetHourlySummary(symbol string, exchange string, limit int) ([]CandleHourlySummary, error) {
	var summaries []CandleHourlySummary
	err := r.db.Where("symbol = ? AND exchange = ?", symbol, exchange).
		Order("hour DESC").
		Limit(limit).
		Find(&summaries).Error
	return summaries, err
}

// GetStats calcule les statistiques pour un symbole using the database function
func (r *candleRepository) GetStats(symbol string, timeframe string) (*CandleStats, error) {
	var stats CandleStats

	// Use the database function get_crypto_stats()
	query := `SELECT * FROM get_crypto_stats($1, $2, 'BINANCE')`

	err := r.db.Raw(query, symbol, timeframe).Scan(&stats).Error
	if err != nil {
		return nil, err
	}

	return &stats, nil
}

// GetDistinctSymbols récupère la liste des symboles disponibles (actifs dans la dernière heure)
func (r *candleRepository) GetDistinctSymbols() ([]string, error) {
	var symbols []string

	// Get symbols that have data in the last hour (active symbols)
	query := `
		SELECT DISTINCT symbol
		FROM candles
		WHERE window_start > NOW() - INTERVAL '1 hour'
		ORDER BY symbol
	`

	err := r.db.Raw(query).Scan(&symbols).Error
	if err != nil {
		return nil, err
	}

	return symbols, nil
}
