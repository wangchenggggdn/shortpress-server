package common

import (
	"shortpress-server/internal/repository/db"
	"shortpress-server/pkg/log"
)

// BaseService defines the common interface for all services
// This interface provides the core dependencies (logger and transaction)
// that all services need, without creating circular import dependencies.
type BaseService interface {
	Logger() *log.Logger
	Tx() db.Transaction
}
