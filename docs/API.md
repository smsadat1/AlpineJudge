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
```

---

## POST /job

Creates a new judging job.

### Request

### Metadata

| Field             | Description                               |
|-------------------|-------------------------------------------|
| submission_id     | Client-generated unique job identifier    |
| language          | Programming language                      |
| source            | Submitted code in plain string format     |
| testset_id        | Testset identifier                        |

Example:

```json
{
    "submission_id": "j111",
    "language": "python",
    "filename": "print(\"Hello World!\")",
    "testset_id": "ts12",
}
```

---

## Response

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
---

---

## Verdict Status Codes

| Status    | Description           |
|-----------|-----------------------|
| AC        | Accepted              |
| WA        | Wrong Answer          |
| TLE       | Time Limit Exceeded   |
| MLE       | Memory Limit Exceeded |
| OLE       | Output Limit Exceeded |
| CE        | Compilation Error     |
| RE        | Runtime Error         |
| IE        | Internal Error        |


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
    └── GET /job/{job_id}/events
```