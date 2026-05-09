package service

import (
	"shortpress-server/internal/repository/db"
	"shortpress-server/pkg/jwt"
	"shortpress-server/pkg/log"
	"shortpress-server/pkg/sid"
)

type Service struct {
	logger *log.Logger
	sid    *sid.Sid
	jwt    *jwt.JWT
	tx     db.Transaction
}

func NewService(
	tx db.Transaction,
	logger *log.Logger,
	sid *sid.Sid,
	jwt *jwt.JWT,
) *Service {
	return &Service{
		logger: logger,
		sid:    sid,
		jwt:    jwt,
		tx:     tx,
	}
}

// 添加到 service.go
func (s *Service) Logger() *log.Logger {
	return s.logger
}

// 添加到 service.go
func (s *Service) Tx() db.Transaction {
	return s.tx
}
