package task

import (
	"shortpress-server/internal/repository/db"
	"shortpress-server/pkg/log"
	"shortpress-server/pkg/sid"
)

type Task struct {
	logger *log.Logger
	sid    *sid.Sid
	tm     db.Transaction
}

func NewTask(
	tm db.Transaction,
	logger *log.Logger,
	sid *sid.Sid,
) *Task {
	return &Task{
		logger: logger,
		sid:    sid,
		tm:     tm,
	}
}
