package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

// AdLocation represents an ad placement location in the system
type AdLocation struct {
	ID        uint         `gorm:"column:id;primaryKey;autoIncrement" json:"-"`
	AdID      string       `gorm:"column:ad_id;index:idx_ad_id" json:"ad_id"`
	Location  string       `gorm:"column:location;index:idx_location" json:"location"`
	Name      string       `gorm:"column:name" json:"name"`
	SiteID    string       `gorm:"column:site_id;index:idx_site_id" json:"site_id"`
	ShowStg   ShowStrategy `gorm:"column:show_stg;type:json" json:"show_stg"`
	Status    int8         `gorm:"column:status;default:1;index:idx_status" json:"status"`
	CreatedAt time.Time    `gorm:"column:created_at" json:"created_at"`
	UpdatedAt time.Time    `gorm:"column:updated_at" json:"updated_at"`
}

// ad location join ad
type AdLocationView struct {
	Name     string       `gorm:"column:name" json:"name"`
	Location string       `gorm:"column:location"`
	SiteID   string       `gorm:"column:site_id"`
	ShowStg  ShowStrategy `gorm:"column:show_stg;type:json"`
	Status   int8         `gorm:"column:status"`
	NetWork  string       `gorm:"column:ad_network" `
	Format   int8         `gorm:"column:format"`
	AdID     string       `gorm:"column:ad_id"`
	Conf     AdConfig     `gorm:"column:conf;type:json" json:"conf"`
}

// ShowStrategy represents the JSON configuration for ad display strategy
type ShowStrategy map[string]interface{}

// Value implements the driver.Valuer interface for database serialization
func (s ShowStrategy) Value() (driver.Value, error) {
	if s == nil {
		return nil, nil
	}
	return json.Marshal(s)
}

// Scan implements the sql.Scanner interface for database deserialization
func (s *ShowStrategy) Scan(value interface{}) error {
	if value == nil {
		*s = make(ShowStrategy)
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}

	return json.Unmarshal(bytes, s)
}

// TableName specifies the table name for AdLocation model
func (AdLocation) TableName() string {
	return "ad_locations"
}

// AdLocation status constants
const (
	AdLocationStatusEnabled  = int8(1) // Enabled
	AdLocationStatusDisabled = int8(2) // Disabled
)
