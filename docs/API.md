# API

## Overview

AlpineJudge exposes a minimal API through the Dispatcher subsystem.

Execution is asynchronous.

Typical flow:

```text
POST /job
      │
      ▼
Receive Job ID
      │
      ▼
GET /job/{job_id}/events
      │
      ▼
GET /job/{job_id}/result
```

---

## POST /job

Creates a new judging job.

### Request

### Metadata

| Field | Description |
|--------|-------------|
| job_id | Client-generated unique job identifier |
| language | Programming language |
| version | Language version |
| filename | Must follow `main.*` naming convention |
| testset_id | Testset identifier |
| testset_version | Testset version |

Example:

```json
{
    "job_id": "j111",
    "language": "python",
    "version": "python3.12",
    "filename": "main.py",
    "testset_id": "ts12",
    "testset_version": "v1"
}
```

### Source File

The source file is sent as multipart form data.

The filename **must** be:

```text
main.<extension>
```

Examples:

```text
main.cpp
main.c
main.py
main.java
main.go
```

Any other filename is rejected.

---

## Response

Returns immediately after the job has been validated and successfully queued.

```json
{
    "job_id": "j111",
    "status": "QUEUED"
}
```

---

## GET /job/{job_id}/events

Returns the current execution progress.

Example response:

```json
{
    "job_id": "j111",
    "status": "RUNNING",
    "event": "Running test case 3/20"
}
```

Possible events include:

- Queued
- Downloading source
- Preparing execution environment
- Compiling
- Running test case X/N
- Cleaning up
- Completed
- Failed

---

## GET /job/{job_id}/result

Returns the final judging result.

If execution has not completed, the endpoint should indicate that the result is not yet available (for example, by returning an appropriate status code such as `202 Accepted` or a response indicating the job is still in progress).

Example:

```json
{
    "job_id": "j111",
    "language": "python",
    "status": "AC",
    "elapsed_time_ms": 555,
    "memory_usage_mb": 24,
    "log_kb": 1202
}
```

---

## Verdict Status Codes

| Status | Description |
|---------|-------------|
| AC | Accepted |
| WA | Wrong Answer |
| TLE | Time Limit Exceeded |
| MLE | Memory Limit Exceeded |
| OLE | Output Limit Exceeded |
| CE | Compilation Error |
| RE | Runtime Error |
| IE | Internal Error |
| PE | Presentation Error |
| SE | Security Error (Sandbox Violation) |

---

## Execution Flow

```text
Client
    │
    │ POST /job
    ▼
Dispatcher
    │
    ▼
RabbitMQ
    │
    ▼
Runner
    │
    ▼
Execution
    │
    ├── GET /job/{job_id}/events
    │
    └── GET /job/{job_id}/result
```