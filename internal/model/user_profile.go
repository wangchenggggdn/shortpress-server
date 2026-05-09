package model

import (
	"time"
)

// UserProfile represents a user's profile information
type UserProfile struct {
	ID         uint      `gorm:"column:id;primaryKey;autoIncrement" json:"-"`
	UserID     string    `gorm:"column:user_id;uniqueIndex:uk_user_id" json:"user_id"`
	Nickname   string    `gorm:"column:nickname" json:"nickname"`
	AvatarURL  string    `gorm:"column:avatar_url" json:"avatar_url"`
	Bio        string    `gorm:"column:bio;type:text" json:"bio"`
	AutoUnlock *bool     `gorm:"column:auto_unlock" json:"auto_unlock"` // Whether auto-unlock is enabled
	CreatedAt  time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt  time.Time `gorm:"column:updated_at" json:"updated_at"`
}

// TableName specifies the table name for UserProfile model
func (UserProfile) TableName() string {
	return "user_profiles"
}
