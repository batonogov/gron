package main

import (
	"testing"
	"time"
)

func TestParseField(t *testing.T) {
	tests := []struct {
		field   string
		limits  CronField
		want    []int
		wantErr bool
	}{
		{"*", CronField{0, 5}, []int{0, 1, 2, 3, 4, 5}, false},
		{"*/2", CronField{0, 5}, []int{0, 2, 4}, false},
		{"3", CronField{0, 5}, []int{3}, false},
		{"7", CronField{0, 6}, []int{7}, false},
		{"8", CronField{0, 6}, nil, true},
	}

	for _, tt := range tests {
		got, err := parseField(tt.field, tt.limits)
		if (err != nil) != tt.wantErr {
			t.Errorf("parseField() error = %v, wantErr %v", err, tt.wantErr)
			return
		}
		if !compareSlices(got, tt.want) {
			t.Errorf("parseField() = %v, want %v", got, tt.want)
		}
	}
}

func TestParseCronSchedule(t *testing.T) {
	tests := []struct {
		cronExpr string
		wantErr  bool
	}{
		{"* * * * *", false},
		{"@hourly", false},
		{"@invalid", true},
	}

	for _, tt := range tests {
		_, err := parseCronSchedule(tt.cronExpr)
		if (err != nil) != tt.wantErr {
			t.Errorf("parseCronSchedule() error = %v, wantErr %v", err, tt.wantErr)
		}
	}
}

func TestShouldRun(t *testing.T) {
	schedule, err := parseCronSchedule("* * * * *")
	if err != nil {
		t.Fatalf("Failed to parse cron schedule: %v", err)
	}

	now := time.Now()
	if !schedule.shouldRun(now) {
		t.Errorf("shouldRun() = false, want true")
	}
}

func compareSlices(a, b []int) bool {
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
