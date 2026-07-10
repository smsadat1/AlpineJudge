# Installation Guide

## 1. Download AlpineJudge

```bash
wget https://github.com/smsadat/alpinejudge/releases/download/v0.1.0/alpinejudge-linux-amd64.tar.gz
tar -xzf alpinejudge-linux-amd64.tar.gz
cd alpinejudge-linux-amd64
chmod +x setup.sh
```

---

# 2. Prepare Configuration Files

Before running the setup script, create configuration files based on the provided examples.

Example files:

```text
dispatcher/config.example.yaml
dispatcher/dispatcher.example.env

runner/config.example.yaml
runner/runner.example.env
```

Environment variables contain deployment-specific secrets and service credentials such as:

* RabbitMQ connection information
* S3/MinIO credentials
* API secrets
* External service endpoints

---

# 3. Configure Dispatcher

Dispatcher configuration controls:

* Supported languages
* Allowed language versions
* Registered runners

Example:

```yaml
name: alpine-judge-dispatcher

languages:
  python:
    versions:
      - python3.10
      - python3.12

  cpp:
    versions:
      - c++17
      - c++20

runners:
  - runner-001
  - runner-002
```

### Language Configuration

Only languages listed in this file are accepted by Dispatcher.

If a submission contains:

```json
{
  "language": "rust",
  "version": "rust1.89"
}
```

but Rust is not configured, Dispatcher will reject the submission.

### Runner Registration

The `runners` section lists all valid runner identifiers that may connect to this Dispatcher.

Example:

```yaml
runners:
  - runner-001
  - runner-002
```

---

# 4. Configure Runner

Runner configuration controls:

* Available execution images
* Scheduling behavior
* Resource limits

Example:

```yaml
name: alpine-judge-runner

runner_id: runner-001
```

The `runner_id` must match one of the runner identifiers configured in Dispatcher.

---

## Execution Images

```yaml
images:
  - alpinejudge/gcc:v0.1.0
  - alpinejudge/python:v0.1.0
  - alpinejudge/go:v0.1.0
```

These images are automatically pulled and cached during Runner startup.

Add additional images here when enabling new languages.

---

## Scheduler Configuration

```yaml
scheduler:
  over_sub_factor: 2
  memory_reserve_percent: 20
```

### over_sub_factor

Controls how aggressively Runner oversubscribes execution slots.

Higher values increase throughput but may increase system pressure.

### memory_reserve_percent

Percentage of host memory reserved for the operating system and background services.

Runner will avoid scheduling new containers if remaining memory drops below this threshold.

---

## Resource Limits

```yaml
limits:
  memory_limit_mb: 1024
  pid_limit: 128
  cpu_quota: 2
  no_new_privileges: true
  readonly_rootfs: true
  timeout_sec: 300
```

### memory_limit_mb

Maximum memory available to a submission container.

### pid_limit

Maximum number of processes that may exist inside a container.

Provides protection against fork bombs.

### cpu_quota

Maximum CPU cores available to a container.

### no_new_privileges

Prevents privilege escalation inside containers.

Recommended to remain enabled.

### readonly_rootfs

Mounts container root filesystem as read-only.

Recommended to remain enabled.

### timeout_sec

Maximum execution duration before force termination.

---

# 5. Install Dispatcher

```bash
sudo ./setup.sh dispatcher \
    --env /path/to/.env \
    --config /path/to/config.yaml
```

---

# 6. Install Runner

```bash
sudo ./setup.sh runner \
    --id runner-001 \
    --env /path/to/.env
```

---

# 7. Configuration Locations

Dispatcher:

```text
/etc/alpinejudge/ajdispatcher/
```

Runner:

```text
/etc/alpinejudge/ajrunner/<runner_id>/
```

Systemd units:

```text
/etc/systemd/system/alpinjudge/ajdispatcher.service
/etc/systemd/system/alpinjudge/{runner_id}/ajrunner.service
```

---

# 8. Managing Services

Start services:

```bash
sudo systemctl start ajdispatcher
sudo systemctl start ajrunner
```

Enable on boot:

```bash
sudo systemctl enable ajdispatcher
sudo systemctl enable ajrunner
```

Restart services:

```bash
sudo systemctl restart ajdispatcher
sudo systemctl restart ajrunner
```

Reload configuration:

```bash
sudo systemctl reload ajdispatcher
sudo systemctl reload ajrunner
```

View logs:

```bash
journalctl -u ajdispatcher -f
journalctl -u ajrunner -f
```
