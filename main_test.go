package main

import (
	"testing"
)

// TestParseField checks the parseField function with valid and invalid inputs.
// It ensures that the function correctly handles both cases.
func TestParseField(t *testing.T) {
	// Test with a valid cron expression "*/15", expecting no error.
	_, err := parseField("*/15", CronField{0, 59})
	if err != nil {
		t.Errorf("parseField() error = %v, wantErr false", err)
	}

	// Test with an invalid cron expression "invalid", expecting an error.
	_, err = parseField("invalid", CronField{0, 59})
	if err == nil {
		t.Errorf("Expected error for invalid input, got nil")
	}
}

// TestParseCronSchedule verifies the parseCronSchedule function with a standard cron schedule.
// It ensures the function can handle valid cron schedule strings without errors.
func TestParseCronSchedule(t *testing.T) {
	// Test with a valid cron schedule "* * * * *", expecting no error.
	_, err := parseCronSchedule("* * * * *")
	if err != nil {
		t.Errorf("parseCronSchedule() error = %v, wantErr false", err)
	}
}

// TestParseEveryFormat checks the parseEveryFormat function with an interval format.
// It ensures the function can correctly parse and handle "@every" format schedules.
func TestParseEveryFormat(t *testing.T) {
	// Test with a valid "@every" format "1h", expecting no error.
	_, err := parseEveryFormat("@every 1h")
	if err != nil {
		t.Errorf("parseEveryFormat() error = %v, wantErr false", err)
	}
}
