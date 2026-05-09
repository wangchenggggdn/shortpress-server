package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

// Ad represents an advertisement in the system
type Ad struct {
	ID        uint      `gorm:"column:id;primaryKey;autoIncrement" json:"-"`
	AdID      string    `gorm:"column:ad_id;uniqueIndex:uk_ad_id" json:"ad_id"`
	Name      string    `gorm:"column:name" json:"name"`
	Format    int8      `gorm:"column:format;default:1;index:idx_format" json:"format"`
	AdNetwork string    `gorm:"column:ad_network" json:"ad_network"`
	Conf      AdConfig  `gorm:"column:conf;type:json" json:"conf"`
	Status    int8      `gorm:"column:status;default:1;index:idx_status" json:"status"`
	CreatedAt time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at" json:"updated_at"`
}

// AdConfig represents the JSON configuration for an ad
type AdConfig map[string]interface{}

// Value implements the driver.Valuer interface for database serialization
func (c AdConfig) Value() (driver.Value, error) {
	if c == nil {
		return nil, nil
	}
	return json.Marshal(c)
}

// Scan implements the sql.Scanner interface for database deserialization
func (c *AdConfig) Scan(value interface{}) error {
	if value == nil {
		*c = make(AdConfig)
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}

	return json.Unmarshal(bytes, c)
}

// TableName specifies the table name for Ad model
func (Ad) TableName() string {
	return "ads"
}

// // AdShowStrategy represents the JSON strategy for showing an ad
// type AdShowStrategy map[string]interface{}

// // Value implements the driver.Valuer interface for database serialization
// func (s AdShowStrategy) Value() (driver.Value, error) {
// 	if s == nil {
// 		return nil, nil
// 	}
// 	return json.Marshal(s)
// }

// // Scan implements the sql.Scanner interface for database deserialization
// func (s *AdShowStrategy) Scan(value interface{}) error {
// 	if value == nil {
// 		*s = make(AdShowStrategy)
// 		return nil
// 	}

// 	bytes, ok := value.([]byte)
// 	if !ok {
// 		return errors.New("type assertion to []byte failed")
// 	}

// 	return json.Unmarshal(bytes, s)
// }

// Ad format constants
const (
	AdFormatDisplay = int8(1) // Display ads
	AdFormatUnlock  = int8(2) // Unlock ads
)

// Ad status constants
const (
	AdStatusEnabled  = int8(1)   // Enabled
	AdStatusDisabled = int8(2)   // Disabled
	AdStatusDeleted  = int8(127) // Deleted
)
