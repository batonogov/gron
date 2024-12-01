# Gron - Docker Cron Scheduler

A simple and flexible task scheduler in a Docker container that supports both standard cron syntax and simplified intervals.

## Features

- Standard cron syntax support
- Simplified interval syntax (`@every`)
- Script execution from mounted directory
- Easy configuration via environment variables

## Usage

### Basic Example

```bash
docker run --rm \
-v ./scripts/:/scripts/ \
-e TASK_1=*/1 * * * * /scripts/test_script1.sh \
-e TASK_2=@every 10s /scripts/test_script2.sh \
-e TASK_3=@hourly /scripts/test_script3.sh \
gron
```

### Command Structure

```bash
docker run [docker-options] \
-v /path/to/scripts:/scripts \
-e 'TASK_NAME=schedule command' \
gron
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
-e TASK_1=/5 /scripts/backup.sh \
-e TASK_2=@every 1h /scripts/health_check.sh \
-e TASK_3=0 0 /scripts/daily_report.sh \
gron
```

## License

MIT
