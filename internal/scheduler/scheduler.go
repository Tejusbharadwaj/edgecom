//go:generate godoc -html . > ../../docs/internal/scheduler/index.html

package scheduler

import (
	"context"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"

	"github.com/tejusbharadwaj/edgecom/internal/api"
)

// Package scheduler implements background data fetching and processing.
// It manages periodic updates of time series data from external sources.
type Scheduler struct {
	ctx     context.Context
	fetcher *api.SeriesFetcher
	logger  *logrus.Logger
	cron    *cron.Cron
}

func NewScheduler(ctx context.Context, fetcher *api.SeriesFetcher, logger *logrus.Logger) *Scheduler {
	return &Scheduler{
		ctx:     ctx,
		fetcher: fetcher,
		logger:  logger,
		cron:    cron.New(),
	}
}

// Start begins the scheduling of periodic data fetches.
// It continues running until the context is canceled or an unrecoverable error occurs.
func (s *Scheduler) Start() error {
	// Run data fetch every 5 minutes
	_, err := s.cron.AddFunc("*/5 * * * *", s.collectData)
	if err != nil {
		return err
	}
	s.cron.Start()
	return nil
}

// collectData fetches data from the API and stores it in the database
func (s *Scheduler) collectData() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	endTime := time.Now()
	startTime := endTime.Add(-5 * time.Minute)

	if err := s.fetcher.FetchData(ctx, startTime, endTime); err != nil {
		s.logger.Error("Failed to fetch data", err)
	}
}

// Stop the scheduler
func (s *Scheduler) Stop() {
	s.cron.Stop()
}
