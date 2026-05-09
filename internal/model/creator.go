package model

import (
	"time"
)

type Creator struct {
	CreatorID    string     `gorm:"column:creator_id"`
	Email        string     `gorm:"column:email"`
	PasswordHash string     `gorm:"column:password_hash"`
	Type         int        `gorm:"column:type"`
	Role         int        `gorm:"column:role"`
	Status       int        `gorm:"column:status"`
	LastLoginAt  *time.Time `gorm:"column:last_login_at"`
	CreatedAt    time.Time  `gorm:"column:created_at"`
	UpdatedAt    time.Time  `gorm:"column:updated_at"`
}

func (m *Creator) TableName() string {
	return "creators"
}
