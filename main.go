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

// CronField represents the allowed range for a cron expression field.
type CronField struct {
	min, max int
}

// Define valid ranges for each cron field: minutes, hours, day of the month, months, and days of the week.
var cronFields = []CronField{
	{0, 59}, // Minutes
	{0, 23}, // Hours
	{1, 31}, // Days of the month
	{1, 12}, // Months
	{0, 6},  // Days of the week
}

// CronSchedule represents a parsed cron expression and the associated command.
type CronSchedule struct {
	minutes     []int
	hours       []int
	daysOfMonth []int
	months      []int
	daysOfWeek  []int
	command     string
	interval    time.Duration // Duration for @every format.
	isEvery     bool          // Flag to indicate @every format.
}

// Predefined special schedule formats (e.g., @hourly, @daily).
var specialSchedules = map[string]string{
	"@hourly":  "0 * * * *",
	"@daily":   "0 0 * * *",
	"@weekly":  "0 0 * * 0",
	"@monthly": "0 0 1 * *",
	"@yearly":  "0 0 1 1 *",
}

// parseField parses a single field of a cron expression.
// Supports:
// - "*" for any value.
// - "*/n" for step values.
// - Specific numbers.
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

	val, err := strconv.Atoi(field)
	if err != nil {
		return nil, err
	}

	// Allow 7 to represent Sunday in the "day of the week" field.
	if (limits.min == 0 && limits.max == 6 && val == 7) || (val >= limits.min && val <= limits.max) {
		return []int{val}, nil
	}

	return nil, fmt.Errorf("value %d out of range for field", val)
}

// parseCronSchedule parses a complete cron expression into a CronSchedule.
// Supports standard cron format and special formats (e.g., @hourly).
func parseCronSchedule(cronExpr string) (*CronSchedule, error) {
	// Check for special formats.
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

	// Parse each field.
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

// shouldRun checks if the schedule should run at the given time.
func (s *CronSchedule) shouldRun(t time.Time) bool {
	contains := func(arr []int, val int) bool {
		for _, v := range arr {
			if v == val {
				return true
			}
		}
		return false
	}

	dayOfWeek := int(t.Weekday())

	return contains(s.minutes, t.Minute()) &&
		contains(s.hours, t.Hour()) &&
		contains(s.daysOfMonth, t.Day()) &&
		contains(s.months, int(t.Month())) &&
		// Проверяем оба возможных представления воскресенья (0 и 7)
		(contains(s.daysOfWeek, dayOfWeek) || (dayOfWeek == 0 && contains(s.daysOfWeek, 7)))
}

// parseEveryFormat parses the @every duration format.
// Example: "@every 1h30m".
func parseEveryFormat(duration string) (*CronSchedule, error) {
	durationStr := strings.TrimPrefix(duration, "@every ")

	// Convert days to hours
	if strings.HasSuffix(durationStr, "d") {
		daysStr := strings.TrimSuffix(durationStr, "d")
		days, err := strconv.Atoi(daysStr)
		if err != nil {
			return nil, fmt.Errorf("invalid days format: %v", err)
		}
		durationStr = fmt.Sprintf("%dh", days*24)
	}

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

// loadTasks loads all tasks from environment variables.
// Environment variables should be in the format:
// TASK_* = "<cron expression> <command>".
// Supports three formats:
// 1. Standard cron: "* * * * * /path/to/command".
// 2. @every format: "@every 1h /path/to/command".
// 3. Special formats: "@hourly /path/to/command".
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

			// Check for special formats (@hourly, @daily, etc.).
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
				// Handle standard cron format.
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

// CommandRunner u0438u043du0442u0435u0440u0444u0435u0439u0441 u0434u043bu044f u0437u0430u043fu0443u0441u043au0430 u043au043eu043cu0430u043du0434
type CommandRunner interface {
	Run(command string, args ...string) ([]byte, error)
}

// RealCommandRunner u0440u0435u0430u043bu044cu043du044bu0439 u0438u0441u043fu043eu043bu043du0438u0442u0435u043bu044c u043au043eu043cu0430u043du0434
type RealCommandRunner struct{}

// Run u0432u044bu043fu043eu043bu043du044fu0435u0442 u043au043eu043cu0430u043du0434u0443 u0438 u0432u043eu0437u0432u0440u0430u0449u0430u0435u0442 u0432u044bu0432u043eu0434
func (r *RealCommandRunner) Run(command string, args ...string) ([]byte, error) {
	cmd := exec.Command(command, args...)
	return cmd.CombinedOutput()
}

// u0433u043bu043eu0431u0430u043bu044cu043du044bu0439 u0438u0441u043fu043eu043bu043du0438u0442u0435u043bu044c u043au043eu043cu0430u043du0434
var defaultCommandRunner CommandRunner = &RealCommandRunner{}

// executeCommand runs the specified command using bash.
// Logs both the command execution and its output.
func executeCommand(command string) {
	log.Printf("Running command: %s", command)

	// u0438u0441u043fu043eu043bu044cu0437u0443u0435u043c u0438u043du0442u0435u0440u0444u0435u0439u0441 CommandRunner u0434u043bu044f u0432u043eu0437u043cu043eu0436u043du043eu0441u0442u0438 u043cu043eu043au0438u0440u043eu0432u0430u043du0438u044f
	output, err := defaultCommandRunner.Run("/bin/bash", "-c", command)

	if err != nil {
		log.Printf("Error executing command %s: %v", command, err)
	}
	log.Printf("Output from command %s: %s", command, string(output))
}

// setCommandRunner u0443u0441u0442u0430u043du0430u0432u043bu0438u0432u0430u0435u0442 u043au0430u0441u0442u043eu043cu043du044bu0439 u0438u0441u043fu043eu043bu043du0438u0442u0435u043bu044c u043au043eu043cu0430u043du0434 (u0434u043bu044f u0442u0435u0441u0442u043eu0432)
func setCommandRunner(runner CommandRunner) {
	defaultCommandRunner = runner
}

// createEveryTickers creates tickers for tasks with @every format.
// Returns a slice of created tickers for cleanup purpose.
func createEveryTickers(tasks []*CronSchedule) []*time.Ticker {
	var tickers []*time.Ticker

	for _, task := range tasks {
		if task.isEvery {
			ticker := time.NewTicker(task.interval)
			tickers = append(tickers, ticker)

			go func(t *CronSchedule, tkr *time.Ticker) {
				for range tkr.C {
					go executeCommand(t.command)
				}
			}(task, ticker)
		}
	}

	return tickers
}

// runCronTasks runs standard cron tasks that match the current time.
func runCronTasks(tasks []*CronSchedule, currentTime time.Time) {
	for _, task := range tasks {
		if !task.isEvery && task.shouldRun(currentTime) {
			go executeCommand(task.command)
		}
	}
}

// startCronScheduler starts the main cron scheduler loop.
// This is a blocking function that runs indefinitely.
func startCronScheduler(tasks []*CronSchedule) {
	// Setup and start the tickers for @every tasks
	everyTickers := createEveryTickers(tasks)

	// Make sure to clean up all tickers when done
	defer func() {
		for _, ticker := range everyTickers {
			ticker.Stop()
		}
	}()

	// Main ticker for regular cron jobs
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case t := <-ticker.C:
			runCronTasks(tasks, t)
		}
	}
}

// main initializes and runs the cron scheduler.
// Creates separate tickers for @every tasks and a main ticker for standard cron tasks.
func main() {
	tasks := loadTasks()
	startCronScheduler(tasks)
}
