package job

import (
	"shortpress-server/internal/repository/db"
	"shortpress-server/pkg/log"
	"shortpress-server/pkg/sid"
)

type Job struct {
	logger *log.Logger
	sid    *sid.Sid
	tm     db.Transaction
}

func NewJob(
	tm db.Transaction,
	logger *log.Logger,
	sid *sid.Sid,
) *Job {
	return &Job{
		logger: logger,
		sid:    sid,
		tm:     tm,
	}
}
