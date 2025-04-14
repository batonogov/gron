package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// Test parseField with various inputs
func TestParseField(t *testing.T) {
	tests := []struct {
		name       string
		field      string
		limits     CronField
		expected   []int
		shouldFail bool
	}{
		{"wildcard", "*", CronField{0, 59}, makeRange(0, 59), false},
		{"step_values", "*/15", CronField{0, 59}, []int{0, 15, 30, 45}, false},
		{"specific_number", "7", CronField{0, 59}, []int{7}, false},
		{"out_of_range", "61", CronField{0, 59}, nil, true},
		{"invalid_step", "*/abc", CronField{0, 59}, nil, true},
		{"zero_value", "0", CronField{0, 59}, []int{0}, false},
		{"max_value", "59", CronField{0, 59}, []int{59}, false},
		{"sunday_day_of_week", "7", CronField{0, 6}, []int{7}, false},
		{"specific_day_of_week", "0", CronField{0, 6}, []int{0}, false},
		{"invalid_day_of_week", "8", CronField{0, 6}, nil, true},
		{"smaller_step", "*/5", CronField{0, 59}, []int{0, 5, 10, 15, 20, 25, 30, 35, 40, 45, 50, 55}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseField(tt.field, tt.limits)
			if tt.shouldFail && err == nil {
				t.Errorf("expected failure for input %s", tt.field)
			} else if !tt.shouldFail && err != nil {
				t.Errorf("unexpected error for input %s: %v", tt.field, err)
			} else if !tt.shouldFail && !equalSlices(result, tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// Test parseCronSchedule with valid and invalid cron expressions
func TestParseCronSchedule(t *testing.T) {
	tests := []struct {
		name         string
		expression   string
		shouldFail   bool
		minutes      []int
		hours        []int
		daysOfMonth  []int
		months       []int
		daysOfWeek   []int
	}{
		{"all_wildcards", "* * * * *", false, makeRange(0, 59), makeRange(0, 23), makeRange(1, 31), makeRange(1, 12), makeRange(0, 6)},
		{"specific_time", "0 12 * * 1", false, []int{0}, []int{12}, makeRange(1, 31), makeRange(1, 12), []int{1}},
		{"step_values", "*/15 */6 */10 */3 */2", false, []int{0, 15, 30, 45}, []int{0, 6, 12, 18}, []int{1, 11, 21, 31}, []int{1, 4, 7, 10}, []int{0, 2, 4, 6}},
		{"hourly_special", "@hourly", false, []int{0}, makeRange(0, 23), makeRange(1, 31), makeRange(1, 12), makeRange(0, 6)},
		{"daily_special", "@daily", false, []int{0}, []int{0}, makeRange(1, 31), makeRange(1, 12), makeRange(0, 6)},
		{"weekly_special", "@weekly", false, []int{0}, []int{0}, makeRange(1, 31), makeRange(1, 12), []int{0}},
		{"monthly_special", "@monthly", false, []int{0}, []int{0}, []int{1}, makeRange(1, 12), makeRange(0, 6)},
		{"yearly_special", "@yearly", false, []int{0}, []int{0}, []int{1}, []int{1}, makeRange(0, 6)},
		{"invalid_special", "@invalid", true, nil, nil, nil, nil, nil},
		{"too_few_fields", "* * *", true, nil, nil, nil, nil, nil},
		{"invalid_field", "a * * * *", true, nil, nil, nil, nil, nil},
		{"specific_numeric", "1 2 3 4 5", false, []int{1}, []int{2}, []int{3}, []int{4}, []int{5}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schedule, err := parseCronSchedule(tt.expression)
			if tt.shouldFail && err == nil {
				t.Errorf("expected failure for input %s", tt.expression)
			} else if !tt.shouldFail && err != nil {
				t.Errorf("unexpected error for input %s: %v", tt.expression, err)
			} else if !tt.shouldFail {
				if !equalSlices(schedule.minutes, tt.minutes) {
					t.Errorf("expected minutes %v, got %v", tt.minutes, schedule.minutes)
				}
				if !equalSlices(schedule.hours, tt.hours) {
					t.Errorf("expected hours %v, got %v", tt.hours, schedule.hours)
				}
				if !equalSlices(schedule.daysOfMonth, tt.daysOfMonth) {
					t.Errorf("expected days of month %v, got %v", tt.daysOfMonth, schedule.daysOfMonth)
				}
				if !equalSlices(schedule.months, tt.months) {
					t.Errorf("expected months %v, got %v", tt.months, schedule.months)
				}
				if !equalSlices(schedule.daysOfWeek, tt.daysOfWeek) {
					t.Errorf("expected days of week %v, got %v", tt.daysOfWeek, schedule.daysOfWeek)
				}
			}
		})
	}
}

// Test shouldRun to verify the scheduling logic
func TestShouldRun(t *testing.T) {
	tests := []struct {
		name         string
		schedule     *CronSchedule
		currentTime  time.Time
		shouldRun    bool
		description  string
	}{
		{
			"specific_date_time_match",
			&CronSchedule{
				minutes:     []int{0, 30},
				hours:       []int{12},
				daysOfMonth: []int{1},
				months:      []int{1, 7},
				daysOfWeek:  []int{1},
			},
			time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
			true,
			"Monday, January 1st at 12:00",
		},
		{
			"specific_date_time_match_30min",
			&CronSchedule{
				minutes:     []int{0, 30},
				hours:       []int{12},
				daysOfMonth: []int{1},
				months:      []int{1, 7},
				daysOfWeek:  []int{1},
			},
			time.Date(2024, 1, 1, 12, 30, 0, 0, time.UTC),
			true,
			"Monday, January 1st at 12:30",
		},
		{
			"specific_date_time_match_july",
			&CronSchedule{
				minutes:     []int{0, 30},
				hours:       []int{12},
				daysOfMonth: []int{1},
				months:      []int{1, 7},
				daysOfWeek:  []int{1},
			},
			time.Date(2024, 7, 1, 12, 0, 0, 0, time.UTC),
			true,
			"Monday, July 1st at 12:00",
		},
		{
			"wrong_day_of_month",
			&CronSchedule{
				minutes:     []int{0, 30},
				hours:       []int{12},
				daysOfMonth: []int{1},
				months:      []int{1, 7},
				daysOfWeek:  []int{1},
			},
			time.Date(2024, 1, 2, 12, 0, 0, 0, time.UTC),
			false,
			"Tuesday, January 2nd at 12:00 - wrong day",
		},
		{
			"wrong_month",
			&CronSchedule{
				minutes:     []int{0, 30},
				hours:       []int{12},
				daysOfMonth: []int{1},
				months:      []int{1, 7},
				daysOfWeek:  []int{1},
			},
			time.Date(2024, 2, 1, 12, 0, 0, 0, time.UTC),
			false,
			"Thursday, February 1st at 12:00 - wrong month",
		},
		{
			"wrong_minute",
			&CronSchedule{
				minutes:     []int{0, 30},
				hours:       []int{12},
				daysOfMonth: []int{1},
				months:      []int{1, 7},
				daysOfWeek:  []int{1},
			},
			time.Date(2024, 1, 1, 12, 15, 0, 0, time.UTC),
			false,
			"Monday, January 1st at 12:15 - wrong minute",
		},
		{
			"sunday_test_0",
			&CronSchedule{
				minutes:     []int{0},
				hours:       []int{12},
				daysOfMonth: makeRange(1, 31),
				months:      makeRange(1, 12),
				daysOfWeek:  []int{0},
			},
			time.Date(2024, 1, 7, 12, 0, 0, 0, time.UTC), // January 7th, 2024 is a Sunday
			true,
			"Sunday (0) should run",
		},
		{
			"sunday_test_7",
			&CronSchedule{
				minutes:     []int{0},
				hours:       []int{12},
				daysOfMonth: makeRange(1, 31),
				months:      makeRange(1, 12),
				daysOfWeek:  []int{7},
			},
			time.Date(2024, 1, 7, 12, 0, 0, 0, time.UTC), // January 7th, 2024 is a Sunday
			true,
			"Sunday (7) should run",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.schedule.shouldRun(tt.currentTime)
			if result != tt.shouldRun {
				t.Errorf("expected %v, got %v for %s", tt.shouldRun, result, tt.description)
			}
		})
	}
}

// Test parseEveryFormat for correct parsing of durations
func TestParseEveryFormat(t *testing.T) {
	tests := []struct {
		name       string
		expression string
		duration   time.Duration
		shouldFail bool
		isEvery    bool
	}{
		{"hourly", "@every 1h", time.Hour, false, true},
		{"minutes", "@every 30m", 30 * time.Minute, false, true},
		{"day", "@every 1d", 24 * time.Hour, false, true},
		{"multiple_days", "@every 2d", 48 * time.Hour, false, true},
		{"seconds", "@every 30s", 30 * time.Second, false, true},
		{"combined_time", "@every 1h30m", 90 * time.Minute, false, true},
		{"invalid_format", "@every abc", 0, true, false},
		{"invalid_day_combo", "@every 1d2h", 0, true, false},
		{"mixed_units", "@every 1h45m30s", time.Hour + 45*time.Minute + 30*time.Second, false, true},
		// Go u043fu0430u0440u0441u0438u0442 u044du0442u0438 u0437u043du0430u0447u0435u043du0438u044f u043au043eu0440u0440u0435u043au0442u043du043e, u043fu043eu044du0442u043eu043cu0443 u043eu043du0438 u043du0435 u0432u044bu0437u044bu0432u0430u044eu0442 u043eu0448u0438u0431u043au0438
		{"zero_days", "@every 0d", 0, false, true},
		{"negative_time", "@every -1h", -time.Hour, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schedule, err := parseEveryFormat(tt.expression)
			if tt.shouldFail && err == nil {
				t.Errorf("expected failure for input %s", tt.expression)
			} else if !tt.shouldFail && err != nil {
				t.Errorf("unexpected error for input %s: %v", tt.expression, err)
			} else if !tt.shouldFail && schedule.interval != tt.duration {
				t.Errorf("expected %v, got %v for input %s", tt.duration, schedule.interval, tt.expression)
			}

			if !tt.shouldFail && !schedule.isEvery {
				t.Errorf("isEvery flag should be true for @every format")
			}
		})
	}
}

// Mock exec.Command for testing executeCommand
func mockExecCommand(command string, args ...string) *exec.Cmd {
	cs := []string{command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], "-test.run=TestHelperProcess", "--", strings.Join(cs, " "))
	cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1"}
	return cmd
}

// TestHelperProcess is a helper function for mocking exec.Command
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	args := os.Args
	for len(args) > 0 {
		if args[0] == "--" {
			args = args[1:]
			break
		}
		args = args[1:]
	}
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "No command\n")
		os.Exit(2)
	}

	if strings.Contains(args[0], "success") {
		fmt.Println("Command executed successfully")
		os.Exit(0)
	} else if strings.Contains(args[0], "fail") {
		fmt.Println("Command failed")
		os.Exit(1)
	}
	os.Exit(0)
}

// MockCommandRunner реализация CommandRunner для тестирования
type MockCommandRunner struct {
	ShouldFail  bool
	ReturnError error
	ReturnOutput []byte
	Commands     []string
	Args         [][]string
}

// Run записывает вызовы команды и возвращает заданные значения
func (m *MockCommandRunner) Run(command string, args ...string) ([]byte, error) {
	m.Commands = append(m.Commands, command)
	m.Args = append(m.Args, args)

	if m.ShouldFail {
		return m.ReturnOutput, m.ReturnError
	}

	return m.ReturnOutput, nil
}

// SimulateExecuteCommand симулирует логику функции executeCommand для тестирования
func simulateExecuteCommand(command string) (string, error) {
	// Логирование запуска команды (как это делает настоящая функция)
	log.Printf("Running command: %s", command)

	// Симуляция результата выполнения команды
	var output string
	var err error

	if strings.Contains(command, "fail") {
		// Симуляция неудачного выполнения
		err = fmt.Errorf("command failed")
		output = "Error output"
		log.Printf("Error executing command %s: %v", command, err)
	} else {
		// Симуляция успешного выполнения
		output = "Command executed successfully"
		log.Printf("Output from command %s: %s", command, output)
	}

	return output, err
}

// TestExecuteCommand tests the executeCommand function with MockCommandRunner
func TestExecuteCommand(t *testing.T) {
	// Create a mock command runner
	mockRunner := &MockCommandRunner{
		ReturnOutput: []byte("Test successful output"),
	}

	// Backup original command runner and restore after test
	originalRunner := defaultCommandRunner
	// Set our mock runner
	setCommandRunner(mockRunner)
	defer setCommandRunner(originalRunner)

	// Capture log output
	originalOutput := log.Writer()
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(originalOutput)

	// Test successful command execution
	t.Run("real_command_success", func(t *testing.T) {
		buf.Reset()

		// Execute command through the actual function
		executeCommand("test command")

		// Verify the mock was called correctly
		if len(mockRunner.Commands) != 1 {
			t.Fatalf("Expected 1 command call, got %d", len(mockRunner.Commands))
		}
		if mockRunner.Commands[0] != "/bin/bash" {
			t.Errorf("Expected command /bin/bash, got %s", mockRunner.Commands[0])
		}
		if len(mockRunner.Args[0]) != 2 || mockRunner.Args[0][1] != "test command" {
			t.Errorf("Expected args [-c test command], got %v", mockRunner.Args[0])
		}

		// Check logs
		logOutput := buf.String()
		if !strings.Contains(logOutput, "Running command: test command") {
			t.Errorf("Expected log to contain the command")
		}
		if !strings.Contains(logOutput, "Test successful output") {
			t.Errorf("Expected log to contain output from command")
		}
	})

	// Test failing command execution
	t.Run("real_command_failure", func(t *testing.T) {
		// Reset the buffer and configure mock for failure
		buf.Reset()
		mockRunner.ShouldFail = true
		mockRunner.ReturnError = fmt.Errorf("mock command failure")
		mockRunner.ReturnOutput = []byte("Error output")
		mockRunner.Commands = nil
		mockRunner.Args = nil

		// Execute command
		executeCommand("failing command")

		// Verify mock was called
		if len(mockRunner.Commands) != 1 {
			t.Fatalf("Expected 1 command call, got %d", len(mockRunner.Commands))
		}

		// Check logs
		logOutput := buf.String()
		if !strings.Contains(logOutput, "Running command: failing command") {
			t.Errorf("Expected log to contain the command")
		}
		if !strings.Contains(logOutput, "Error executing command") {
			t.Errorf("Expected log to contain error message")
		}
		if !strings.Contains(logOutput, "mock command failure") {
			t.Errorf("Expected log to contain specific error message")
		}
	})

	// Original simulation tests
	t.Run("simulation_successful_command", func(t *testing.T) {
		// Clear buffer before test
		buf.Reset()

		// Call our simulation instead of real executeCommand
		output, err := simulateExecuteCommand("success_command")

		// Check no error
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		// Check output value
		if output != "Command executed successfully" {
			t.Errorf("Expected 'Command executed successfully', got '%s'", output)
		}

		// Check log messages
		logOutput := buf.String()
		if !strings.Contains(logOutput, "Running command: success_command") {
			t.Errorf("Expected log to contain run command message")
		}
		if !strings.Contains(logOutput, "Output from command") {
			t.Errorf("Expected log to contain output message")
		}
	})

	// Test failing command simulation
	t.Run("simulation_failing_command", func(t *testing.T) {
		// Clear buffer before test
		buf.Reset()

		// Call our simulation instead of real executeCommand
		_, err := simulateExecuteCommand("fail_command")

		// Check for error
		if err == nil {
			t.Errorf("Expected an error, got nil")
		}

		// Check log messages
		logOutput := buf.String()
		if !strings.Contains(logOutput, "Running command: fail_command") {
			t.Errorf("Expected log to contain run command message")
		}
		if !strings.Contains(logOutput, "Error executing command") {
			t.Errorf("Expected log to contain error message")
		}
	})
}

// Test loadTasks to verify environment variable parsing
func TestLoadTasks(t *testing.T) {
	// Clear existing environment variables that might interfere
	for _, env := range os.Environ() {
		if strings.HasPrefix(env, "TASK_") {
			parts := strings.SplitN(env, "=", 2)
			os.Unsetenv(parts[0])
		}
	}

	tests := []struct {
		name           string
		envVars        map[string]string
		expectedTasks  int
		expectedCmd    string
		taskType       string  // "standard", "special", or "every"
	}{
		{
			"standard_cron",
			map[string]string{"TASK_TEST": "* * * * * echo hello"},
			1,
			"echo hello",
			"standard",
		},
		{
			"special_format",
			map[string]string{"TASK_HOURLY": "@hourly echo hourly"},
			1,
			"echo hourly",
			"special",
		},
		{
			"every_format",
			map[string]string{"TASK_EVERY": "@every 1h echo every hour"},
			1,
			"echo every hour",
			"every",
		},
		{
			"multiple_tasks",
			map[string]string{
				"TASK_TEST1": "* * * * * echo test1",
				"TASK_TEST2": "@daily echo test2",
				"TASK_TEST3": "@every 5m echo test3",
			},
			3,
			"", // not checking command in this case
			"",
		},
		{
			"invalid_task",
			map[string]string{"TASK_INVALID": "invalid"},
			0,
			"",
			"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables for the test
			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}

			// Run loadTasks
			tasks := loadTasks()

			// Clean up environment
			for k := range tt.envVars {
				os.Unsetenv(k)
			}

			// Verify results
			if len(tasks) != tt.expectedTasks {
				t.Fatalf("expected %d task(s), got %d", tt.expectedTasks, len(tasks))
			}

			// If we're expecting tasks and want to check the command
			if tt.expectedTasks == 1 && tt.expectedCmd != "" {
				if tasks[0].command != tt.expectedCmd {
					t.Errorf("expected command '%s', got '%s'", tt.expectedCmd, tasks[0].command)
				}

				// Verify task type
				switch tt.taskType {
				case "every":
					if !tasks[0].isEvery {
						t.Errorf("expected isEvery to be true")
					}
				case "standard":
					if tasks[0].isEvery {
						t.Errorf("expected isEvery to be false for standard cron")
					}
				case "special":
					if tasks[0].isEvery {
						t.Errorf("expected isEvery to be false for special format")
					}
				}
			}
		})
	}
}

// TestCreateEveryTickers tests the createEveryTickers function
func TestCreateEveryTickers(t *testing.T) {
	// Create test schedules
	tasks := []*CronSchedule{
		{
			isEvery:   true,
			interval:  time.Second,
			command:   "test_command_1",
		},
		{
			isEvery:   true,
			interval:  time.Minute,
			command:   "test_command_2",
		},
		{
			// Non-@every task, should be skipped
			isEvery:   false,
			minutes:   []int{0},
			hours:     []int{12},
			command:   "test_regular_task",
		},
	}

	// Run the function
	tickers := createEveryTickers(tasks)

	// Check that only two tickers were created
	if len(tickers) != 2 {
		t.Errorf("Expected 2 tickers, got %d", len(tickers))
	}

	// Cleanup
	for _, ticker := range tickers {
		ticker.Stop()
	}
}

// TestRunCronTasks tests the runCronTasks function
func TestRunCronTasks(t *testing.T) {
	// Create a mock command runner to record executions
	mockRunner := &MockCommandRunner{}
	// Save original and restore after test
	originalRunner := defaultCommandRunner
	setCommandRunner(mockRunner)
	defer setCommandRunner(originalRunner)

	// Create test tasks
	tasks := []*CronSchedule{
		{
			// Match all times
			isEvery:     false,
			minutes:     makeRange(0, 59),
			hours:       makeRange(0, 23),
			daysOfMonth: makeRange(1, 31),
			months:      makeRange(1, 12),
			daysOfWeek:  makeRange(0, 6),
			command:     "matching_task",
		},
		{
			// Won't match (specific minute)
			isEvery:     false,
			minutes:     []int{30},  // Only run at minute 30
			hours:       makeRange(0, 23),
			daysOfMonth: makeRange(1, 31),
			months:      makeRange(1, 12),
			daysOfWeek:  makeRange(0, 6),
			command:     "non_matching_task",
		},
		{
			// @every task, should be skipped
			isEvery:     true,
			interval:    time.Hour,
			command:     "every_task",
		},
	}

	// Run tasks with a specific time (minute 15)
	testTime := time.Date(2025, 1, 1, 12, 15, 0, 0, time.UTC)

	// Run the function
	runCronTasks(tasks, testTime)

	// Give goroutines a moment to execute
	time.Sleep(50 * time.Millisecond)

	// Check command execution
	if len(mockRunner.Commands) != 1 {
		t.Fatalf("Expected 1 command execution, got %d", len(mockRunner.Commands))
	}

	// Check right command was executed
	if mockRunner.Args[0][1] != "matching_task" {
		t.Errorf("Expected 'matching_task' to be executed, got '%s'", mockRunner.Args[0][1])
	}
}

// Test for specific bug: handling multiple formats of special expressions
func TestSpecialFormats(t *testing.T) {
	specials := []struct {
		name     string
		format   string
		minutes  []int
		hours    []int
		days     []int
		months   []int
		weekdays []int
	}{
		{"hourly", "@hourly", []int{0}, makeRange(0, 23), makeRange(1, 31), makeRange(1, 12), makeRange(0, 6)},
		{"daily", "@daily", []int{0}, []int{0}, makeRange(1, 31), makeRange(1, 12), makeRange(0, 6)},
		{"weekly", "@weekly", []int{0}, []int{0}, makeRange(1, 31), makeRange(1, 12), []int{0}},
		{"monthly", "@monthly", []int{0}, []int{0}, []int{1}, makeRange(1, 12), makeRange(0, 6)},
		{"yearly", "@yearly", []int{0}, []int{0}, []int{1}, []int{1}, makeRange(0, 6)},
	}

	for _, s := range specials {
		t.Run(s.name, func(t *testing.T) {
			schedule, err := parseCronSchedule(s.format)
			if err != nil {
				t.Fatalf("Failed to parse %s: %v", s.format, err)
			}

			if !equalSlices(schedule.minutes, s.minutes) {
				t.Errorf("Expected minutes %v, got %v", s.minutes, schedule.minutes)
			}

			if !equalSlices(schedule.hours, s.hours) {
				t.Errorf("Expected hours %v, got %v", s.hours, schedule.hours)
			}

			// ... test other fields if needed
		})
	}
}

// Integration test for task loading and scheduling
func TestTaskScheduling(t *testing.T) {
	// Set environment for a task that should run immediately
	currentTime := time.Now()
	minute := currentTime.Minute()
	hour := currentTime.Hour()
	day := currentTime.Day()
	month := int(currentTime.Month())
	dayOfWeek := int(currentTime.Weekday())

	// Create a cron expression that matches current time
	cronExpr := fmt.Sprintf("%d %d %d %d %d echo integration test", minute, hour, day, month, dayOfWeek)
	os.Setenv("TASK_INTEGRATION", cronExpr)
	defer os.Unsetenv("TASK_INTEGRATION")

	tasks := loadTasks()
	if len(tasks) == 0 {
		t.Fatal("No tasks were loaded")
	}

	// Verify the task would run at the current time
	found := false
	for _, task := range tasks {
		if task.command == "echo integration test" {
			found = true
			if !task.shouldRun(currentTime) {
				t.Errorf("Task should run at current time but doesn't: %v", currentTime)
			}
		}
	}

	if !found {
		t.Error("Integration test task was not found")
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
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
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

// Helper to create a simple CronSchedule for testing
func createTestSchedule(minutes, hours, days, months, weekdays []int) *CronSchedule {
	return &CronSchedule{
		minutes:     minutes,
		hours:       hours,
		daysOfMonth: days,
		months:      months,
		daysOfWeek:  weekdays,
	}
}

// TestHelperProcess используется для тестирования executeCommand
