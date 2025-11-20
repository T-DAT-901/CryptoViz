package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Currency représente une devise (crypto ou fiat)
// Table: currencies (regular PostgreSQL table)
type Currency struct {
	ID        uuid.UUID `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name      *string   `json:"name" gorm:"column:name;size:100"`
	Symbol    *string   `json:"symbol" gorm:"column:symbol;size:20"`
	Type      string    `json:"type" gorm:"column:type;size:10;not null"` // 'crypto' or 'fiat'
	CreatedAt time.Time `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`
}

// TableName spécifie le nom de la table pour GORM
func (Currency) TableName() string {
	return "currencies"
}

// BeforeCreate hook GORM appelé avant la création
func (c *Currency) BeforeCreate(tx *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	// Validate type
	if c.Type != "crypto" && c.Type != "fiat" {
		return gorm.ErrInvalidValue
	}
	return nil
}

// CurrencyRepository interface pour les opérations sur Currency
type CurrencyRepository interface {
	Create(currency *Currency) error
	GetByID(id uuid.UUID) (*Currency, error)
	GetBySymbol(symbol string) (*Currency, error)
	GetByType(currencyType string) ([]Currency, error)
	GetAll() ([]Currency, error)
	Update(currency *Currency) error
	Delete(id uuid.UUID) error
}

// currencyRepository implémentation concrète du repository
type currencyRepository struct {
	db *gorm.DB
}

// NewCurrencyRepository crée une nouvelle instance du repository
func NewCurrencyRepository(db *gorm.DB) CurrencyRepository {
	return &currencyRepository{db: db}
}

// Create insère une nouvelle devise
func (r *currencyRepository) Create(currency *Currency) error {
	return r.db.Create(currency).Error
}

// GetByID récupère une devise par son ID
func (r *currencyRepository) GetByID(id uuid.UUID) (*Currency, error) {
	var currency Currency
	err := r.db.Where("id = ?", id).First(&currency).Error
	if err != nil {
		return nil, err
	}
	return &currency, nil
}

// GetBySymbol récupère une devise par son symbole
func (r *currencyRepository) GetBySymbol(symbol string) (*Currency, error) {
	var currency Currency
	err := r.db.Where("symbol = ?", symbol).First(&currency).Error
	if err != nil {
		return nil, err
	}
	return &currency, nil
}

// GetByType récupère toutes les devises d'un type (crypto ou fiat)
func (r *currencyRepository) GetByType(currencyType string) ([]Currency, error) {
	var currencies []Currency
	err := r.db.Where("type = ?", currencyType).
		Order("name ASC").
		Find(&currencies).Error
	return currencies, err
}

// GetAll récupère toutes les devises
func (r *currencyRepository) GetAll() ([]Currency, error) {
	var currencies []Currency
	err := r.db.Order("type ASC, name ASC").Find(&currencies).Error
	return currencies, err
}

// Update met à jour une devise
func (r *currencyRepository) Update(currency *Currency) error {
	return r.db.Save(currency).Error
}

// Delete supprime une devise
func (r *currencyRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&Currency{}, id).Error
}
