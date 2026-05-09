package model

import (
	"time"
)

// User represents a C-end user in the system
type User struct {
	ID               uint       `gorm:"column:id;primaryKey;autoIncrement" json:"-"`
	UserID           string     `gorm:"column:user_id;uniqueIndex:uk_user_id" json:"user_id"`
	Email            string     `gorm:"column:email" json:"email"`
	Identifier       string     `gorm:"column:identifier" json:"identifier"`
	SiteID           string     `gorm:"column:site_id;index:idx_site_id" json:"site_id"`
	Status           int8       `gorm:"column:status;default:1;index:idx_status" json:"status"`
	LastLoginAt      *time.Time `gorm:"column:last_login_at" json:"last_login_at"`
	CreatedAt        time.Time  `gorm:"column:created_at;index:idx_created_at" json:"created_at"`
	PremiumType      int8       `gorm:"column:premium_type;not null;default:0" json:"premium_type"`
	PremiumExpiresAt *time.Time `gorm:"column:premium_expires_at" json:"premium_expires_at,omitempty"`
	Referer          string     `gorm:"column:referer" json:"referer"`
	UpdatedAt        time.Time  `gorm:"column:updated_at" json:"updated_at"`
	Ver              string     `gorm:"column:ver" json:"ver"`
}

// TableName specifies the table name for User model
func (User) TableName() string {
	return "users"
}

// IsValidPremiumMember checks if the user is a valid premium member
func (u *User) IsValidPremiumMember() bool {
	if u.PremiumExpiresAt == nil {
		return false
	}
	return u.PremiumType >= 1 && u.PremiumExpiresAt.After(time.Now())
}

// User status constants
const (
	UserStatusInactive = int8(1)   // Not activated
	UserStatusActive   = int8(2)   // Activated
	UserStatusDisabled = int8(3)   // Disabled
	UserStatusDeleted  = int8(127) // Deleted
)

// UserQuery represents query parameters for user listing
type UserQuery struct {
	SiteID     string // Site ID to filter users by
	UserID     string // Optional user ID filter
	SearchTerm string // Search term for email or nickname
	Status     int8   // Status filter: -1 for all statuses
}

type UserInfoView struct {
	ID               uint       `gorm:"column:id;primaryKey;autoIncrement" json:"-"`
	UserID           string     `gorm:"column:user_id;uniqueIndex:uk_user_id" json:"user_id"`
	Email            string     `gorm:"column:email" json:"email"`
	Identifier       string     `gorm:"column:identifier" json:"identifier"`
	SiteID           string     `gorm:"column:site_id;index:idx_site_id" json:"site_id"`
	Status           int8       `gorm:"column:status;default:1;index:idx_status" json:"status"`
	LastLoginAt      *time.Time `gorm:"column:last_login_at" json:"last_login_at"`
	CreatedAt        time.Time  `gorm:"column:created_at;index:idx_created_at" json:"created_at"`
	PremiumType      int8       `gorm:"column:premium_type;not null;default:0" json:"premium_type"`
	PremiumExpiresAt *time.Time `gorm:"column:premium_expires_at" json:"premium_expires_at,omitempty"`
	UpdatedAt        time.Time  `gorm:"column:updated_at" json:"updated_at"`
	Nickname         string     `gorm:"column:nickname" json:"nickname"` // 从 user_profiles 联表查询获取，不映射到数据库列
	Ver              string     `gorm:"column:ver" json:"ver"`
}
