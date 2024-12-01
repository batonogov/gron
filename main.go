package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// CronField represents the allowed range for a cron expression field
type CronField struct {
	min, max int
}

var cronFields = []CronField{
	{0, 59}, // minutes
	{0, 23}, // hours
	{1, 31}, // days of month
	{1, 12}, // months
	{0, 6},  // days of week
}

// CronSchedule represents a parsed cron expression with its command
type CronSchedule struct {
	minutes     []int
	hours       []int
	daysOfMonth []int
	months      []int
	daysOfWeek  []int
	command     string
	interval    time.Duration // For @every format
	isEvery     bool          // Flag to identify @every format
	rawSchedule string        // Original schedule string for logging
}

var specialSchedules = map[string]string{
	"@hourly":  "0 * * * *",
	"@daily":   "0 0 * * *",
	"@weekly":  "0 0 * * 0",
	"@monthly": "0 0 1 * *",
	"@yearly":  "0 0 1 1 *",
}

func parseField(field string, limits CronField) ([]int, error) {
	var result []int
	if field == "*" {
		for i := limits.min; i <= limits.max; i++ {
			result = append(result, i)
		}
		return result, nil
	}

	if strings.Contains(field, "*/") {
		step, err := strconv.Atoi(strings.TrimPrefix(field, "*/"))
		if err != nil {
			return nil, err
		}
		for i := limits.min; i <= limits.max; i += step {
			result = append(result, i)
		}
		return result, nil
	}

	if strings.Contains(field, "-") {
		parts := strings.Split(field, "-")
		start, err := strconv.Atoi(parts[0])
		if err != nil {
			return nil, err
		}
		end, err := strconv.Atoi(parts[1])
		if err != nil {
			return nil, err
		}
		for i := start; i <= end; i++ {
			result = append(result, i)
		}
		return result, nil
	}

	if strings.Contains(field, ",") {
		for _, part := range strings.Split(field, ",") {
			val, err := strconv.Atoi(part)
			if err != nil {
				return nil, err
			}
			result = append(result, val)
		}
		return result, nil
	}

	val, err := strconv.Atoi(field)
	if err != nil {
		return nil, err
	}
	if limits == cronFields[4] && val == 7 {
		val = 0
	}
	if val < limits.min || val > limits.max {
		return nil, fmt.Errorf("value %d out of range", val)
	}
	return []int{val}, nil
}

func parseCronSchedule(cronExpr string) (*CronSchedule, error) {
	if strings.HasPrefix(cronExpr, "@") {
		if schedule, ok := specialSchedules[cronExpr]; ok {
			cronExpr = schedule
		} else {
			return nil, fmt.Errorf("unknown special schedule: %s", cronExpr)
		}
	}

	fields := strings.Fields(cronExpr)
	if len(fields) < 5 {
		return nil, fmt.Errorf("invalid cron expression")
	}

	schedule := &CronSchedule{}
	var err error
	schedule.minutes, err = parseField(fields[0], cronFields[0])
	if err != nil {
		return nil, err
	}
	schedule.hours, err = parseField(fields[1], cronFields[1])
	if err != nil {
		return nil, err
	}
	schedule.daysOfMonth, err = parseField(fields[2], cronFields[2])
	if err != nil {
		return nil, err
	}
	schedule.months, err = parseField(fields[3], cronFields[3])
	if err != nil {
		return nil, err
	}
	schedule.daysOfWeek, err = parseField(fields[4], cronFields[4])
	if err != nil {
		return nil, err
	}
	return schedule, nil
}

func parseEveryFormat(duration string) (*CronSchedule, error) {
	durationStr := strings.TrimPrefix(duration, "@every ")
	d, err := time.ParseDuration(durationStr)
	if err != nil {
		return nil, err
	}

	return &CronSchedule{
		interval:    d,
		isEvery:     true,
		rawSchedule: "@every " + durationStr,
	}, nil
}

func loadTasks() []*CronSchedule {
	var tasks []*CronSchedule

	for _, env := range os.Environ() {
		if strings.HasPrefix(env, "TASK_") {
			parts := strings.SplitN(env, "=", 2)
			if len(parts) != 2 {
				continue
			}

			taskDef := strings.TrimSpace(parts[1])
			fields := strings.Fields(taskDef)
			if len(fields) < 2 {
				log.Printf("Invalid task format: %s", taskDef)
				continue
			}

			var schedule *CronSchedule
			var err error
			var command string

			if strings.HasPrefix(fields[0], "@") {
				if strings.HasPrefix(fields[0], "@every") {
					everyExpr := fields[0] + " " + fields[1]
					schedule, err = parseEveryFormat(everyExpr)
					command = strings.Join(fields[2:], " ")
				} else {
					schedule, err = parseCronSchedule(fields[0])
					command = strings.Join(fields[1:], " ")
				}
			} else {
				cronExpr := strings.Join(fields[:5], " ")
				schedule, err = parseCronSchedule(cronExpr)
				command = strings.Join(fields[5:], " ")
				schedule.rawSchedule = cronExpr
			}

			if err != nil {
				log.Printf("Failed to parse task: %s, error: %v", taskDef, err)
				continue
			}

			schedule.command = command
			tasks = append(tasks, schedule)
		}
	}
	return tasks
}

func (s *CronSchedule) shouldRun(t time.Time) bool {
	contains := func(arr []int, val int) bool {
		for _, v := range arr {
			if v == val {
				return true
			}
		}
		return false
	}
	return contains(s.minutes, t.Minute()) &&
		contains(s.hours, t.Hour()) &&
		contains(s.daysOfMonth, t.Day()) &&
		contains(s.months, int(t.Month())) &&
		contains(s.daysOfWeek, int(t.Weekday()))
}

func executeCommand(command string) {
	log.Printf("Running command: %s", command)
	cmd := exec.Command("/bin/bash", "-c", command)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Error executing command: %s, error: %v", command, err)
	}
	log.Printf("Output: %s", string(output))
}

func main() {
	tasks := loadTasks()

	for _, task := range tasks {
		log.Printf("Scheduled task: '%s' with schedule '%s'", task.command, task.rawSchedule)
	}

	for _, task := range tasks {
		if task.isEvery {
			go func(t *CronSchedule) {
				ticker := time.NewTicker(t.interval)
				defer ticker.Stop()
				for range ticker.C {
					executeCommand(t.command)
				}
			}(task)
		}
	}

	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		now := <-ticker.C
		for _, task := range tasks {
			if !task.isEvery && task.shouldRun(now) {
				go executeCommand(task.command)
			}
		}
	}
}
