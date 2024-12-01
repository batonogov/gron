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

// Define valid ranges for each cron field
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
	interval    time.Duration // Used for @every format
	isEvery     bool          // Flag to identify @every format
}

// Predefined special schedule formats
var specialSchedules = map[string]string{
	"@hourly":  "0 * * * *",
	"@daily":   "0 0 * * *",
	"@weekly":  "0 0 * * 0",
	"@monthly": "0 0 1 * *",
	"@yearly":  "0 0 1 1 *",
}

// parseField parses a single field of a cron expression
// Supports:
// - "*" for any value
// - "*/n" for step values
// - specific numbers
func parseField(field string, limits CronField) ([]int, error) {
	if field == "*" {
		result := make([]int, limits.max-limits.min+1)
		for i := range result {
			result[i] = i + limits.min
		}
		return result, nil
	}

	if strings.Contains(field, "*/") {
		step, err := strconv.Atoi(strings.TrimPrefix(field, "*/"))
		if err != nil {
			return nil, err
		}
		var result []int
		for i := limits.min; i <= limits.max; i += step {
			result = append(result, i)
		}
		return result, nil
	}

	// Обработка конкретного значения
	val, err := strconv.Atoi(field)
	if err != nil {
		return nil, err
	}
	if val < limits.min || val > limits.max {
		return nil, err
	}
	return []int{val}, nil
}

// parseCronSchedule parses a complete cron expression into a CronSchedule
// Supports both standard cron format and special formats (@hourly, @daily, etc.)
func parseCronSchedule(cronExpr string) (*CronSchedule, error) {
	// Проверяем специальные форматы
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

	// Парсим каждое поле
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

// shouldRun checks if the schedule should run at the given time
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

// parseEveryFormat parses the @every duration format
// Example: "@every 1h30m"
func parseEveryFormat(duration string) (*CronSchedule, error) {
	durationStr := strings.TrimPrefix(duration, "@every ")

	d, err := time.ParseDuration(durationStr)
	if err != nil {
		return nil, err
	}

	schedule := &CronSchedule{
		interval: d,
		isEvery:  true,
	}

	return schedule, nil
}

// loadTasks loads all tasks from environment variables
// Environment variables should be in format:
// TASK_* = "<cron expression> <command>"
// Supports three formats:
// 1. Standard cron: "* * * * * /path/to/command"
// 2. @every format: "@every 1h /path/to/command"
// 3. Special formats: "@hourly /path/to/command"
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

			// Проверяем специальные форматы (@hourly, @daily и т.д.)
			if strings.HasPrefix(fields[0], "@") && !strings.HasPrefix(fields[0], "@every") {
				schedule, err = parseCronSchedule(fields[0])
				if err != nil {
					log.Printf("Failed to parse special format '%s': %v", fields[0], err)
					continue
				}
				command = strings.Join(fields[1:], " ")
			} else if strings.HasPrefix(fields[0], "@every") {
				everyExpr := fields[0] + " " + fields[1]
				schedule, err = parseEveryFormat(everyExpr)
				if err != nil {
					log.Printf("Failed to parse @every format '%s': %v", everyExpr, err)
					continue
				}
				command = strings.Join(fields[2:], " ")
			} else {
				// Обрабатываем обычный cron формат
				cronExpr := strings.Join(fields[:5], " ")
				schedule, err = parseCronSchedule(cronExpr)
				if err != nil {
					log.Printf("Failed to parse cron expression '%s': %v", cronExpr, err)
					continue
				}
				command = strings.Join(fields[5:], " ")
			}

			schedule.command = command
			log.Printf("Scheduled task: '%s' with schedule '%s'", command, taskDef)
			tasks = append(tasks, schedule)
		}
	}
	return tasks
}

// executeCommand runs the specified command using bash
// Logs both the command execution and its output
func executeCommand(command string) {
	log.Printf("Running command: %s", command)
	cmd := exec.Command("/bin/bash", "-c", command)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Error executing command %s: %v", command, err)
	}
	log.Printf("Output from command %s: %s", command, string(output))
}

// main initializes and runs the cron scheduler
// Creates separate tickers for @every tasks
// and a main ticker for standard cron tasks
func main() {
	tasks := loadTasks()

	// Создаем отдельные тикеры для каждой @every задачи
	for _, task := range tasks {
		if task.isEvery {
			go func(t *CronSchedule) {
				ticker := time.NewTicker(t.interval)
				defer ticker.Stop()

				for range ticker.C {
					go executeCommand(t.command)
				}
			}(task)
		}
	}

	// Main ticker for regular cron jobs
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case t := <-ticker.C:
			for _, task := range tasks {
				if !task.isEvery && task.shouldRun(t) {
					go executeCommand(task.command)
				}
			}
		}
	}
}