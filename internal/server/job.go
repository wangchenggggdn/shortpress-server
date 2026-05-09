package server

import (
	"context"
	"shortpress-server/pkg/log"
)

type JobServer struct {
	log *log.Logger
}

func NewJobServer(
	log *log.Logger,
) *JobServer {
	return &JobServer{
		log: log,
	}
}

func (j *JobServer) Start(ctx context.Context) error {
	// Tips: If you want job to start as a separate process, just refer to the task implementation and adjust the code accordingly.

	// eg: kafka consumer
	return nil
}
func (j *JobServer) Stop(ctx context.Context) error {
	return nil
}
