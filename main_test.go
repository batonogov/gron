package main

import (
	"os"
	"testing"
	"time"
)

// Test parseField with various inputs
func TestParseField(t *testing.T) {
	tests := []struct {
		field      string
		limits     CronField
		expected   []int
		shouldFail bool
	}{
		{"*", CronField{0, 59}, makeRange(0, 59), false},
		{"*/15", CronField{0, 59}, []int{0, 15, 30, 45}, false},
		{"7", CronField{0, 59}, []int{7}, false},
		{"61", CronField{0, 59}, nil, true},
		{"*/abc", CronField{0, 59}, nil, true},
	}

	for _, tt := range tests {
		result, err := parseField(tt.field, tt.limits)
		if tt.shouldFail && err == nil {
			t.Errorf("expected failure for input %s", tt.field)
		} else if !tt.shouldFail && err != nil {
			t.Errorf("unexpected error for input %s: %v", tt.field, err)
		} else if !equalSlices(result, tt.expected) {
			t.Errorf("expected %v, got %v", tt.expected, result)
		}
	}
}

// Test parseCronSchedule with valid and invalid cron expressions
func TestParseCronSchedule(t *testing.T) {
	tests := []struct {
		expression string
		shouldFail bool
	}{
		{"* * * * *", false},
		{"0 12 * * 1", false},
		{"@hourly", false},
		{"@invalid", true},
		{"* * *", true},
	}

	for _, tt := range tests {
		_, err := parseCronSchedule(tt.expression)
		if tt.shouldFail && err == nil {
			t.Errorf("expected failure for input %s", tt.expression)
		} else if !tt.shouldFail && err != nil {
			t.Errorf("unexpected error for input %s: %v", tt.expression, err)
		}
	}
}

// Test shouldRun to verify the scheduling logic
func TestShouldRun(t *testing.T) {
	schedule := &CronSchedule{
		minutes:     []int{0, 30},
		hours:       []int{12},
		daysOfMonth: []int{1},
		months:      []int{1, 7},
		daysOfWeek:  []int{1},
	}

	tests := []struct {
		currentTime time.Time
		shouldRun   bool
	}{
		{time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC), true},
		{time.Date(2024, 1, 1, 12, 30, 0, 0, time.UTC), true},
		{time.Date(2024, 7, 1, 12, 0, 0, 0, time.UTC), true},
		{time.Date(2024, 1, 2, 12, 0, 0, 0, time.UTC), false},
	}

	for _, tt := range tests {
		if result := schedule.shouldRun(tt.currentTime); result != tt.shouldRun {
			t.Errorf("expected %v, got %v for time %v", tt.shouldRun, result, tt.currentTime)
		}
	}
}

// Test parseEveryFormat for correct parsing of durations
func TestParseEveryFormat(t *testing.T) {
	tests := []struct {
		expression string
		duration   time.Duration
		shouldFail bool
	}{
		{"@every 1h", time.Hour, false},
		{"@every 30m", 30 * time.Minute, false},
		{"@every 1d", 24 * time.Hour, false},
		{"@every 2d", 48 * time.Hour, false},
		{"@every abc", 0, true},
		{"@every 1d2h", 0, true},
	}

	for _, tt := range tests {
		schedule, err := parseEveryFormat(tt.expression)
		if tt.shouldFail && err == nil {
			t.Errorf("expected failure for input %s", tt.expression)
		} else if !tt.shouldFail && err != nil {
			t.Errorf("unexpected error for input %s: %v", tt.expression, err)
		} else if !tt.shouldFail && schedule.interval != tt.duration {
			t.Errorf("expected %v, got %v for input %s", tt.duration, schedule.interval, tt.expression)
		}
	}
}

// Test loadTasks to verify environment variable parsing
func TestLoadTasks(t *testing.T) {
	os.Setenv("TASK_TEST", "* * * * * echo hello")
	defer os.Unsetenv("TASK_TEST")

	tasks := loadTasks()
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}

	if tasks[0].command != "echo hello" {
		t.Errorf("expected command 'echo hello', got '%s'", tasks[0].command)
	}
}

// Utility functions for tests
func makeRange(min, max int) []int {
	result := make([]int, max-min+1)
	for i := range result {
		result[i] = min + i
	}
	return result
}

func equalSlices(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}
