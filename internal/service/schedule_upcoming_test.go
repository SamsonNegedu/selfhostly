package service

import (
	"testing"
	"time"

	"github.com/robfig/cron/v3"
)

func TestUpcomingRunsAreMergedInTimeOrder(t *testing.T) {
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	start, _ := parser.Parse("0 8 * * *")
	stop, _ := parser.Parse("0 22 * * *")
	from := time.Date(2026, 9, 20, 0, 14, 0, 0, time.UTC)

	runs := upcomingRuns(from, start, stop, 6)

	want := []struct {
		action string
		hour   int
		day    int
	}{{"start", 8, 20}, {"stop", 22, 20}, {"start", 8, 21}, {"stop", 22, 21}, {"start", 8, 22}, {"stop", 22, 22}}
	if len(runs) != len(want) {
		t.Fatalf("got %d runs, want %d", len(runs), len(want))
	}
	for i, w := range want {
		if runs[i].Action != w.action || runs[i].At.Hour() != w.hour || runs[i].At.Day() != w.day {
			t.Errorf("run %d = %s %s, want %s %02d:00 on the %dth", i, runs[i].Action, runs[i].At, w.action, w.hour, w.day)
		}
	}
}

func TestUpcomingRunsWithOnlyOneScheduleAndNone(t *testing.T) {
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	start, _ := parser.Parse("0 8 * * 1-5")
	from := time.Date(2026, 9, 18, 12, 0, 0, 0, time.UTC) // a Friday

	runs := upcomingRuns(from, start, nil, 3)
	if len(runs) != 3 || runs[0].At.Weekday() != time.Monday {
		t.Fatalf("weekday schedule from a Friday afternoon should next run on Monday, got %+v", runs)
	}
	if got := upcomingRuns(from, nil, nil, 6); got == nil || len(got) != 0 {
		t.Fatalf("no schedules must give an empty list, not nil: %#v", got)
	}
}
