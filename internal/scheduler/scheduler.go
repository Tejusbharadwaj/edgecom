// Package scheduler implements background data fetching and processing for time series data.
//
// The scheduler provides:
//   - Configurable periodic data fetching using cron expressions
//   - Context-aware execution with timeout handling
//   - Graceful shutdown support
//   - Structured logging of fetch operations
//   - Error handling and recovery
//
// Example Usage:
//
//	logger := logrus.New()
//	fetcher := api.NewSeriesFetcher(client, db, logger)
//
//	scheduler := scheduler.NewScheduler(ctx, fetcher, logger)
//	if err := scheduler.Start(); err != nil {
//	    log.Fatalf("Failed to start scheduler: %v", err)
//	}
//
//	defer scheduler.Stop()
package scheduler

import (
	"context"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"

	"github.com/tejusbharadwaj/edgecom/internal/api"
)

// Scheduler manages periodic data fetching operations.
// It uses cron scheduling to regularly update time series data
// from external sources and store it in the database.

type Scheduler struct {
	ctx     context.Context
	fetcher *api.SeriesFetcher
	logger  *logrus.Logger
	cron    *cron.Cron
}

// NewScheduler creates a new scheduler instance with the provided
// context, data fetcher, and logger. The context can be used to
// control the scheduler's lifecycle.
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
	s.logger.Info("Initializing scheduler with 5-minute intervals")

	_, err := s.cron.AddFunc("@every 5m", s.collectData)
	if err != nil {
		return err
	}

	s.cron.Start()
	s.logger.Info("Scheduler started successfully")
	return nil
}

// collectData fetches data from the API and stores it in the database
func (s *Scheduler) collectData() {
	s.logger.Info("Starting scheduled data collection")

	ctx, cancel := context.WithTimeout(s.ctx, 2*time.Minute)
	defer cancel()

	endTime := time.Now()
	startTime := endTime.Add(-5 * time.Minute)

	s.logger.WithFields(logrus.Fields{
		"startTime": startTime,
		"endTime":   endTime,
	}).Info("Fetching data")

	if err := s.fetcher.FetchData(ctx, startTime, endTime); err != nil {
		s.logger.WithError(err).Error("Failed to fetch data")
	} else {
		s.logger.Info("Successfully completed scheduled data collection")
	}
}

// Stop the scheduler
func (s *Scheduler) Stop() {
	s.cron.Stop()
}
