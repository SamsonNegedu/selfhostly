package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/selfhostly/internal/db"
	"github.com/selfhostly/internal/domain"
)

// Scheduler manages application schedules using cron expressions
type Scheduler struct {
	cron       *cron.Cron
	db         *db.DB
	logger     *slog.Logger
	mu         sync.RWMutex
	schedules  map[string]*db.AppSchedule
	entries    map[string]*scheduleEntry
	appService domain.AppService
}

// NewScheduler creates a new scheduler instance
func NewScheduler(database *db.DB, appService domain.AppService, logger *slog.Logger) *Scheduler {
	c := cron.New(cron.WithSeconds(), cron.WithParser(cron.NewParser(
		cron.SecondOptional | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor,
	)))

	return &Scheduler{
		cron:       c,
		db:         database,
		logger:     logger,
		schedules:  make(map[string]*db.AppSchedule),
		entries:    make(map[string]*scheduleEntry),
		appService: appService,
	}
}

// Start initializes and starts the scheduler
func (s *Scheduler) Start(ctx context.Context) error {
	s.logger.Info("Starting application scheduler")
	
	// Load all active schedules from database
	if err := s.loadSchedules(); err != nil {
		return err
	}
	
	// Start the cron scheduler
	s.cron.Start()
	
	// Watch for context cancellation
	go func() {
		<-ctx.Done()
		s.logger.Info("Shutting down scheduler")
		ctx := s.cron.Stop()
		select {
		case <-ctx.Done():
			s.logger.Info("Scheduler stopped gracefully")
		case <-time.After(10 * time.Second):
			s.logger.Warn("Scheduler shutdown timed out")
		}
	}()
	
	s.logger.Info("Scheduler started", "active_schedules", len(s.schedules))
	return nil
}

// loadSchedules loads all active schedules from the database
func (s *Scheduler) loadSchedules() error {
	schedules, err := s.db.GetAllSchedules()
	if err != nil {
		return err
	}
	
	for _, schedule := range schedules {
		if err := s.addSchedule(schedule); err != nil {
			s.logger.Error("Failed to load schedule", "app_id", schedule.AppID, "error", err)
		}
	}
	
	return nil
}

// scheduleEntry holds cron entry IDs for a schedule
type scheduleEntry struct {
	startEntryID cron.EntryID
	stopEntryID  cron.EntryID
}

// addSchedule adds a schedule to the cron scheduler
func (s *Scheduler) addSchedule(schedule *db.AppSchedule) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.schedules[schedule.AppID]; exists {
		s.removeScheduleUnsafe(schedule.AppID)
	}

	entry := &scheduleEntry{}

	if schedule.StartCron != "" {
		cronExpr := formatCronWithTimezone(schedule.StartCron, schedule.Timezone)
		entryID, err := s.cron.AddFunc(cronExpr, s.createStartHandler(schedule.AppID))
		if err != nil {
			return fmt.Errorf("failed to add start schedule: %w", err)
		}
		entry.startEntryID = entryID

		s.logger.Info("Added start schedule",
			"app_id", schedule.AppID,
			"cron", schedule.StartCron,
			"timezone", schedule.Timezone)
	}

	if schedule.StopCron != "" {
		cronExpr := formatCronWithTimezone(schedule.StopCron, schedule.Timezone)
		entryID, err := s.cron.AddFunc(cronExpr, s.createStopHandler(schedule.AppID))
		if err != nil {
			if entry.startEntryID != 0 {
				s.cron.Remove(entry.startEntryID)
			}
			return fmt.Errorf("failed to add stop schedule: %w", err)
		}
		entry.stopEntryID = entryID

		s.logger.Info("Added stop schedule",
			"app_id", schedule.AppID,
			"cron", schedule.StopCron,
			"timezone", schedule.Timezone)
	}

	s.schedules[schedule.AppID] = schedule
	s.entries[schedule.AppID] = entry
	return nil
}

// formatCronWithTimezone prepends CRON_TZ to the cron expression if timezone is specified
func formatCronWithTimezone(cronExpr, timezone string) string {
	if timezone != "" && timezone != "UTC" {
		return "CRON_TZ=" + timezone + " " + cronExpr
	}
	return cronExpr
}

// removeSchedule removes a schedule from the cron scheduler
func (s *Scheduler) removeSchedule(appID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.removeScheduleUnsafe(appID)
}

// removeScheduleUnsafe removes a schedule without locking (caller must hold lock)
func (s *Scheduler) removeScheduleUnsafe(appID string) {
	if entry, exists := s.entries[appID]; exists {
		if entry.startEntryID != 0 {
			s.cron.Remove(entry.startEntryID)
		}
		if entry.stopEntryID != 0 {
			s.cron.Remove(entry.stopEntryID)
		}
		delete(s.entries, appID)
	}

	delete(s.schedules, appID)
	s.logger.Info("Removed schedule", "app_id", appID)
}

// UpdateSchedule updates an existing schedule
func (s *Scheduler) UpdateSchedule(schedule *db.AppSchedule) error {
	s.removeSchedule(schedule.AppID)

	if schedule.Enabled {
		return s.addSchedule(schedule)
	}

	return nil
}

// RemoveSchedule removes a schedule (public method for external calls)
func (s *Scheduler) RemoveSchedule(appID string) {
	s.removeSchedule(appID)
}

// createStartHandler creates a handler function for starting an app
func (s *Scheduler) createStartHandler(appID string) func() {
	return func() {
		ctx := context.Background()
		s.logger.Info("Scheduled start triggered", "app_id", appID)

		if err := s.appService.CreateStartJob(ctx, appID); err != nil {
			s.logger.Error("Failed to create start job", "app_id", appID, "error", err)
		}
	}
}

// createStopHandler creates a handler function for stopping an app
func (s *Scheduler) createStopHandler(appID string) func() {
	return func() {
		ctx := context.Background()
		s.logger.Info("Scheduled stop triggered", "app_id", appID)

		if err := s.appService.CreateStopJob(ctx, appID); err != nil {
			s.logger.Error("Failed to create stop job", "app_id", appID, "error", err)
		}
	}
}
