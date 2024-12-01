---
name: Bug report
about: Create a report to help us improve Gron - Docker Cron Scheduler
title: '[BUG] '
labels: bug
assignees: ''
---

**Describe the bug**
A clear and concise description of what the bug is.

**To Reproduce**
Docker run command used:

```bash
docker run --rm \
-v /path/to/scripts:/scripts \
-e TASK_1=... \
...
ghcr.io/batonogov/gron:v0.1.0
```

Steps to reproduce the behavior:

1. Created script '...'
2. Set up task with schedule '...'
3. See error

**Expected behavior**
A clear and concise description of what you expected to happen.

**System Information:**

- Host OS: [e.g. Ubuntu 24.04, macOS 13.0]
- Docker version: [e.g. 27.3.1]
- Gron version: [e.g. 0.1.0]
- Schedule format used: [e.g. standard cron / @every syntax]

**Logs**
Please provide relevant Docker container logs:
paste logs here

## Content of the script that failed to execute (if relevant)

**Additional context**
Add any other context about the problem here:

- Volume mount details
- Task configuration specifics
- Any modifications to the default setup

**Screenshots**
If applicable, add screenshots to help explain your problem.
