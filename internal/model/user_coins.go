package model

import (
	"time"
)

// UserCoins represents a user's coin account
type UserCoins struct {
	ID                  uint      `gorm:"primaryKey;column:id"`
	UserID              string    `gorm:"column:user_id;index"`
	SiteID              string    `gorm:"column:site_id;index"`
	Balance             int       `gorm:"column:balance;default:0"`
	TotalEarned         int       `gorm:"column:total_earned;default:0"`
	TotalSpent          int       `gorm:"column:total_spent;default:0"`
	TotalRealMoneySpent int64     `gorm:"column:total_real_money_spent;default:0"`
	Present             int       `gorm:"column:present;default:0" json:"present"` // Coins from subscription rewards
	CompletedTasks      string    `gorm:"column:completed_tasks;type:text" json:"completed_tasks,omitempty"` // 已完成的任务列表，用逗号分隔
	CreatedAt           time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt           time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (UserCoins) TableName() string {
	return "user_coins"
}
