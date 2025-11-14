package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// User représente un utilisateur de l'application
// Table: users (regular PostgreSQL table)
type User struct {
	ID                uuid.UUID `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	CurrencyID        uuid.UUID `json:"currency_id" gorm:"type:uuid;not null"` // Reference to currencies table
	Username          *string   `json:"username" gorm:"column:username;size:255"`
	PasswordEncrypted *string   `json:"-" gorm:"column:password_encrypted;size:255"` // Hidden in JSON
	FirstName         *string   `json:"first_name" gorm:"column:first_name;size:100"`
	LastName          *string   `json:"last_name" gorm:"column:last_name;size:100"`
	Email             *string   `json:"email" gorm:"column:email;size:255"`
	Phone             *string   `json:"phone" gorm:"column:phone;size:50"`
	Address           *string   `json:"address" gorm:"column:address;type:text"`
	Zipcode           *int      `json:"zipcode" gorm:"column:zipcode"`
	City              *string   `json:"city" gorm:"column:city;size:100"`
	Country           *string   `json:"country" gorm:"column:country;size:100"`
	CreatedAt         time.Time `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt         time.Time `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`

	// Relations
	Currency Currency `json:"currency,omitempty" gorm:"foreignKey:CurrencyID;references:ID"`
}

// TableName spécifie le nom de la table pour GORM
func (User) TableName() string {
	return "users"
}

// BeforeCreate hook GORM appelé avant la création
func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}

// UserRepository interface pour les opérations sur User
type UserRepository interface {
	Create(user *User) error
	GetByID(id uuid.UUID) (*User, error)
	GetByUsername(username string) (*User, error)
	GetByEmail(email string) (*User, error)
	GetAll(limit int, offset int) ([]User, error)
	Update(user *User) error
	Delete(id uuid.UUID) error
}

// userRepository implémentation concrète du repository
type userRepository struct {
	db *gorm.DB
}

// NewUserRepository crée une nouvelle instance du repository
func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

// Create insère un nouvel utilisateur
func (r *userRepository) Create(user *User) error {
	return r.db.Create(user).Error
}

// GetByID récupère un utilisateur par son ID
func (r *userRepository) GetByID(id uuid.UUID) (*User, error) {
	var user User
	err := r.db.Preload("Currency").Where("id = ?", id).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetByUsername récupère un utilisateur par son username
func (r *userRepository) GetByUsername(username string) (*User, error) {
	var user User
	err := r.db.Preload("Currency").Where("username = ?", username).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetByEmail récupère un utilisateur par son email
func (r *userRepository) GetByEmail(email string) (*User, error) {
	var user User
	err := r.db.Preload("Currency").Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetAll récupère tous les utilisateurs avec pagination
func (r *userRepository) GetAll(limit int, offset int) ([]User, error) {
	var users []User
	err := r.db.Preload("Currency").
		Limit(limit).
		Offset(offset).
		Order("created_at DESC").
		Find(&users).Error
	return users, err
}

// Update met à jour un utilisateur
func (r *userRepository) Update(user *User) error {
	return r.db.Save(user).Error
}

// Delete supprime un utilisateur
func (r *userRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&User{}, id).Error
}
