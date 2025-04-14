# Gron - Docker Cron Scheduler

[![Test Coverage: 80%](https://img.shields.io/badge/Test%20Coverage-80%25-success)](./)
[![Tests](https://github.com/batonogov/gron/actions/workflows/tests.yml/badge.svg)](https://github.com/batonogov/gron/actions/workflows/tests.yml)
[![Docker Image](https://github.com/batonogov/gron/actions/workflows/go.yml/badge.svg)](https://github.com/batonogov/gron/actions/workflows/go.yml)

A simple and flexible task scheduler in a Docker container that supports both standard cron syntax and simplified intervals.

## Features

- Standard cron syntax support
- Simplified interval syntax (`@every`)
- Script execution from mounted directory
- Easy configuration via environment variables
- Robust signal handling for graceful container shutdown
- Continuous task execution without premature exit

## Usage

### Using Docker Image

```bash
docker run --rm \
-v ./scripts/:/scripts/ \
-e 'TASK_1=*/1 * * * * /scripts/test_script1.sh' \
-e 'TASK_2=@every 10s /scripts/test_script2.sh' \
-e 'TASK_3=@hourly /scripts/test_script3.sh' \
-e 'TASK_4=@every 1d /scripts/daily_task.sh' \
ghcr.io/batonogov/gron:latest
```

### Using Binary in Your Dockerfile

You can download and use pre-built binaries in your own Docker images. Binaries are available for multiple platforms:

- Linux (amd64, arm64)
- macOS (Intel, Apple Silicon)

Example Dockerfile:

```dockerfile
FROM ubuntu:latest

# Download gron binary for linux/amd64
ADD https://github.com/batonogov/gron/releases/download/v0.2.2/gron-linux-amd64 /usr/local/bin/gron

# Make it executable
RUN chmod +x /usr/local/bin/gron

# Your configuration
COPY ./scripts /scripts
ENV TASK_1="*/5 * * * * /scripts/backup.sh"

# Run gron
CMD ["/usr/local/bin/gron"]
```

### Command Structure

```bash
docker run [docker-options] \
-v '/path/to/scripts:/scripts' \
-e 'TASK_NAME=schedule command' \
ghcr.io/batonogov/gron:latest
```

### Supported Schedule Formats

1. Standard cron syntax:
   - `*/1 * * * *` - every minute
   - `0 */2 * * *` - every 2 hours
   - `0 0 * * *` - daily at midnight

2. Simplified syntax with @every:
   - `@every 30s` - every 30 seconds
   - `@every 1h` - every hour
   - `@every 1d` - every day

## Mounting Scripts

Scripts must be mounted into the container using a volume. It's recommended to use the `/scripts/` directory:

```bash
-v ./local/scripts:/scripts
```

## Environment Variables

Tasks are configured through environment variables with the `TASK_` prefix:

- `TASK_1` - first task
- `TASK_2` - second task
- etc.

Format: `TASK_NAME=schedule command`

## Configuration Examples

### Multiple Tasks with Different Schedules

```bash
docker run --rm \
-v ./scripts/:/scripts/ \
-e 'TASK_1=/5 /scripts/backup.sh' \
-e 'TASK_2=@every 1h /scripts/health_check.sh' \
-e 'TASK_3=@daily /scripts/daily_report.sh' \
-e 'TASK_4=@every 1d /scripts/daily_task.sh' \
ghcr.io/batonogov/gron:latest
```

## Development

This project uses [Task](https://taskfile.dev) for build automation. After cloning the repository, you can use the following commands:

```bash
# Build the application
task build

# Run tests
task test

# Check test coverage
task test:coverage:detail

# Docker operations
task docker:build   # Build Docker image
task docker:run     # Run container
task docker:logs    # View logs
task docker:stop    # Stop container
```

### Docker Container Behavior

The application is designed to run continuously in a Docker container and will:

- Properly handle SIGTERM and SIGINT signals for graceful shutdown
- Continue running tasks until explicitly stopped
- Never exit prematurely, ensuring all scheduled tasks run as expected
- Automatically restart the scheduler if it unexpectedly exits

### Testing

The project has comprehensive test coverage (80%) including:

- Unit tests for parsing cron expressions
- Tests for command execution logic
- Tests for scheduler functionality
- Docker container lifecycle tests

To run tests with coverage report:

```bash
task test:coverage:detail
```

## License

MIT
