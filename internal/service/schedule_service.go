package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/selfhostly/internal/db"
	"github.com/selfhostly/internal/domain"
)

// ScheduleService implements business logic for managing app schedules
type scheduleService struct {
	database *db.DB
	logger   *slog.Logger
}

// NewScheduleService creates a new ScheduleService instance
func NewScheduleService(database *db.DB, logger *slog.Logger) domain.ScheduleService {
	return &scheduleService{
		database: database,
		logger:   logger,
	}
}

// ValidateCronExpression validates a cron expression
func (s *scheduleService) ValidateCronExpression(expression string) error {
	if expression == "" {
		return fmt.Errorf("cron expression cannot be empty")
	}

	// Try to parse the cron expression
	parser := cron.NewParser(cron.SecondOptional | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	_, err := parser.Parse(expression)
	if err != nil {
		return fmt.Errorf("invalid cron expression: %w", err)
	}

	return nil
}

// CreateSchedule creates a new schedule for an app
func (s *scheduleService) CreateSchedule(ctx context.Context, appID, startCron, stopCron, timezone string, enabled bool) (*db.AppSchedule, error) {
	s.logger.InfoContext(ctx, "creating schedule", "app_id", appID, "timezone", timezone)

	_, err := s.database.GetApp(appID)
	if err != nil {
		return nil, domain.WrapAppNotFound(appID, err)
	}

	if timezone == "" {
		timezone = "UTC"
	}

	if _, err := time.LoadLocation(timezone); err != nil {
		return nil, fmt.Errorf("invalid timezone: %w", err)
	}

	if startCron != "" {
		if err := s.ValidateCronExpression(startCron); err != nil {
			return nil, fmt.Errorf("invalid start cron expression: %w", err)
		}
	}

	if stopCron != "" {
		if err := s.ValidateCronExpression(stopCron); err != nil {
			return nil, fmt.Errorf("invalid stop cron expression: %w", err)
		}
	}

	if startCron == "" && stopCron == "" {
		return nil, fmt.Errorf("at least one of start_cron or stop_cron must be provided")
	}

	// Validate that start and stop are different if both are provided
	if startCron != "" && stopCron != "" && startCron == stopCron {
		return nil, fmt.Errorf("start and stop schedules cannot be the same")
	}

	// Check if schedule already exists
	existingSchedule, err := s.database.GetScheduleByAppID(appID)
	if err != nil {
		return nil, fmt.Errorf("failed to check for existing schedule: %w", err)
	}
	if existingSchedule != nil {
		return nil, fmt.Errorf("schedule already exists for app %s", appID)
	}

	// Create new schedule
	schedule := db.NewAppSchedule(appID, startCron, stopCron, timezone, enabled)
	if err := s.database.CreateSchedule(schedule); err != nil {
		return nil, fmt.Errorf("failed to create schedule: %w", err)
	}

	s.logger.InfoContext(ctx, "schedule created",
		"app_id", appID,
		"schedule_id", schedule.ID,
		"enabled", enabled)

	return schedule, nil
}

// UpdateSchedule updates an existing schedule
func (s *scheduleService) UpdateSchedule(ctx context.Context, appID, startCron, stopCron, timezone string, enabled bool) (*db.AppSchedule, error) {
	s.logger.InfoContext(ctx, "updating schedule", "app_id", appID)

	schedule, err := s.database.GetScheduleByAppID(appID)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing schedule: %w", err)
	}
	if schedule == nil {
		return nil, fmt.Errorf("no schedule found for app %s", appID)
	}

	if timezone == "" {
		timezone = "UTC"
	}

	if _, err := time.LoadLocation(timezone); err != nil {
		return nil, fmt.Errorf("invalid timezone: %w", err)
	}

	if startCron != "" {
		if err := s.ValidateCronExpression(startCron); err != nil {
			return nil, fmt.Errorf("invalid start cron expression: %w", err)
		}
	}

	if stopCron != "" {
		if err := s.ValidateCronExpression(stopCron); err != nil {
			return nil, fmt.Errorf("invalid stop cron expression: %w", err)
		}
	}

	if startCron == "" && stopCron == "" {
		return nil, fmt.Errorf("at least one of start_cron or stop_cron must be provided")
	}

	// Validate that start and stop are different if both are provided
	if startCron != "" && stopCron != "" && startCron == stopCron {
		return nil, fmt.Errorf("start and stop schedules cannot be the same")
	}

	// Update schedule fields
	schedule.StartCron = startCron
	schedule.StopCron = stopCron
	schedule.Timezone = timezone
	schedule.Enabled = enabled
	schedule.UpdatedAt = time.Now()

	// Save to database
	if err := s.database.UpdateSchedule(schedule); err != nil {
		return nil, fmt.Errorf("failed to update schedule: %w", err)
	}

	s.logger.InfoContext(ctx, "schedule updated",
		"app_id", appID,
		"schedule_id", schedule.ID,
		"enabled", enabled)

	return schedule, nil
}

// DeleteSchedule deletes a schedule for an app
func (s *scheduleService) DeleteSchedule(ctx context.Context, appID string) error {
	s.logger.InfoContext(ctx, "deleting schedule", "app_id", appID)

	// Get existing schedule to log it
	schedule, err := s.database.GetScheduleByAppID(appID)
	if err != nil {
		return fmt.Errorf("failed to get existing schedule: %w", err)
	}
	if schedule == nil {
		return fmt.Errorf("no schedule found for app %s", appID)
	}

	// Delete from database
	if err := s.database.DeleteSchedule(appID); err != nil {
		return fmt.Errorf("failed to delete schedule: %w", err)
	}

	s.logger.InfoContext(ctx, "schedule deleted",
		"app_id", appID,
		"schedule_id", schedule.ID)

	return nil
}

// GetSchedule gets a schedule for an app
func (s *scheduleService) GetSchedule(ctx context.Context, appID string) (*db.AppSchedule, error) {
	schedule, err := s.database.GetScheduleByAppID(appID)
	if err != nil {
		return nil, fmt.Errorf("failed to get schedule: %w", err)
	}

	// It's valid to not have a schedule for an app
	return schedule, nil
}

// CalculateNextRunTimes calculates next run times for given cron expressions (for testing/preview)
func (s *scheduleService) CalculateNextRunTimes(ctx context.Context, appID, startCron, stopCron, timezone string) (*domain.ScheduleNextRuns, error) {
	// Validate that start and stop are different if both are provided
	if startCron != "" && stopCron != "" && startCron == stopCron {
		return nil, fmt.Errorf("start and stop schedules cannot be the same")
	}

	parser := cron.NewParser(cron.SecondOptional | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)

	var nextStart, nextStop *time.Time

	location, err := time.LoadLocation(timezone)
	if err != nil {
		return nil, fmt.Errorf("failed to load timezone %s: %w", timezone, err)
	}

	var startSchedule, stopSchedule cron.Schedule

	if startCron != "" {
		var err error
		startSchedule, err = parser.Parse(startCron)
		if err != nil {
			return nil, fmt.Errorf("failed to parse start cron: %w", err)
		}

		nextStartTime := startSchedule.Next(time.Now().In(location))
		nextStart = &nextStartTime
	}

	if stopCron != "" {
		var err error
		stopSchedule, err = parser.Parse(stopCron)
		if err != nil {
			return nil, fmt.Errorf("failed to parse stop cron: %w", err)
		}

		// Calculate next stop time - if we have a start time and they're on the same pattern,
		// calculate stop AFTER start, not from now
		if nextStart != nil && startCron != "" {
			startParts := strings.Fields(startCron)
			stopParts := strings.Fields(stopCron)
			
			// If same day pattern, calculate stop after start
			if len(startParts) == 5 && len(stopParts) == 5 {
				sameDayPattern := startParts[2] == stopParts[2] && startParts[4] == stopParts[4]
				
				if sameDayPattern {
					// Calculate stop time after the start time
					nextStopTime := stopSchedule.Next(*nextStart)
					nextStop = &nextStopTime
				} else {
					// Different patterns, calculate from now
					nextStopTime := stopSchedule.Next(time.Now().In(location))
					nextStop = &nextStopTime
				}
			} else {
				nextStopTime := stopSchedule.Next(time.Now().In(location))
				nextStop = &nextStopTime
			}
		} else {
			// No start schedule, calculate from now
			nextStopTime := stopSchedule.Next(time.Now().In(location))
			nextStop = &nextStopTime
		}
	}

	return &domain.ScheduleNextRuns{
		AppID:     appID,
		NextStart: nextStart,
		NextStop:  nextStop,
	}, nil
}

// GetNextRunTimes calculates the next run times for a schedule
func (s *scheduleService) GetNextRunTimes(ctx context.Context, appID string) (*domain.ScheduleNextRuns, error) {
	schedule, err := s.database.GetScheduleByAppID(appID)
	if err != nil {
		return nil, fmt.Errorf("failed to get schedule: %w", err)
	}

	if schedule == nil || !schedule.Enabled {
		return &domain.ScheduleNextRuns{AppID: appID}, nil
	}

	return s.CalculateNextRunTimes(ctx, appID, schedule.StartCron, schedule.StopCron, schedule.Timezone)
}
