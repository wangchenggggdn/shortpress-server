package model

import (
	"time"
)

// UserAuth represents user authentication methods
type UserAuth struct {
	ID           uint      `gorm:"column:id;primaryKey;autoIncrement" json:"-"`
	UserID       string    `gorm:"column:user_id;index:idx_user_id" json:"user_id"`
	Type         int8      `gorm:"column:type" json:"type"`
	Identifier   string    `gorm:"column:identifier" json:"identifier"`
	PasswordHash string    `gorm:"column:password_hash" json:"password_hash"`
	CreatedAt    time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at" json:"updated_at"`
}

// TableName specifies the table name for UserAuth model
func (UserAuth) TableName() string {
	return "user_auth"
}

// Auth type constants
const (
	AuthTypeEmail    = int8(0)
	AuthTypeGoogle   = int8(1)
	AuthTypeFacebook = int8(2)
	AuthTypeTwitter  = int8(3)
	AuthTypeTikTok   = int8(4)
)
