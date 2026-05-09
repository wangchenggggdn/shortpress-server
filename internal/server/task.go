package server

import (
	"context"
	"shortpress-server/pkg/log"

	"github.com/go-co-op/gocron"
)

type TaskServer struct {
	log       *log.Logger
	scheduler *gocron.Scheduler
}

func NewTaskServer(
	log *log.Logger,
) *TaskServer {
	return &TaskServer{
		log: log,
	}
}
func (t *TaskServer) Start(ctx context.Context) error {
	return nil
}
func (t *TaskServer) Stop(ctx context.Context) error {
	t.scheduler.Stop()
	t.log.Info("TaskServer stop...")
	return nil
}
