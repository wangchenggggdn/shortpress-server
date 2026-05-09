package model

import "gorm.io/gorm"

type ClientPlayer struct {
	gorm.Model
}

func (m *ClientPlayer) TableName() string {
	return "client_player"
}
