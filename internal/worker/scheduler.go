package worker

import (
	"context"
	"log/slog"
	"time"
)

// Scheduler runs a job function on a fixed interval until the context
// is cancelled. It owns the loop; the job owns the work.
// Errors from the job are logged and the loop continues — one bad tick
// should never stop the worker.
type Scheduler struct {
	interval time.Duration
	job      func(ctx context.Context) error
	name     string
	logger   *slog.Logger
}

func NewScheduler(name string, interval time.Duration, job func(ctx context.Context) error, logger *slog.Logger) *Scheduler {
	return &Scheduler{
		name:     name,
		interval: interval,
		job:      job,
		logger:   logger,
	}
}

func (s *Scheduler) Start(ctx context.Context) error {
	s.logger.Info("worker started", "name", s.name, "interval", s.interval)

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := s.job(ctx); err != nil {
				s.logger.Error("worker tick failed", "name", s.name, "error", err)
			}
		case <-ctx.Done():
			s.logger.Info("worker stopped", "name", s.name)
			return ctx.Err()
		}
	}
}
