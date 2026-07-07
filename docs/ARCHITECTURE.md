# Architecture

## Overview

AlpineJudge is a **stateless, distributed code execution engine** designed for secure, scalable and high-performance program execution.

The system is composed of two primary subsystems:

1. **Dispatcher**
2. **Runner**

Communication between the two subsystems is performed asynchronously through **RabbitMQ**, while execution artifacts are persisted in **S3-compatible object storage**.

---

## High-Level Architecture

```text
                Client
                   │
                   │ HTTP API
                   ▼
            +----------------+
            |   Dispatcher   |
            +----------------+
                   │
                   │ RabbitMQ
                   ▼
            +----------------+
            |     Runner     |
            +----------------+
                   │
                   │ containerd
                   ▼
             Execution Container
                   │
                   ▼
                  S3
```

---

# Dispatcher

Dispatcher is responsible for accepting client requests and preparing execution jobs.

It contains the following internal services:

- API Service
- Validator
- Submission Preparer
- Config Manager
- RabbitMQ Producer

## Responsibilities

Dispatcher is responsible for:

- Receiving job submissions.
- Validating request payloads.
- Validating language and language version.
- Validating uploaded source file.
- Verifying requested testset.
- Preparing submission payload.
- Uploading submission source to S3.
- Publishing execution jobs to RabbitMQ.
- Streaming execution events to clients through Server-Sent Events (SSE).
- Returning execution results through the Result API.

Dispatcher never executes user code.

---

# Runner

Runner is responsible for executing submitted programs.

Multiple Runner instances may exist simultaneously to provide horizontal scalability.

Each Runner contains the following internal services:

- RabbitMQ Consumer
- Local Queue
- RAD Scheduler
- Executor
- Config Manager

## Responsibilities

Runner is responsible for:

- Consuming execution jobs from RabbitMQ.
- Buffering jobs in a local queue.
- Dynamically scheduling execution capacity.
- Creating execution containers.
- Executing submitted programs.
- Publishing live execution events.
- Uploading execution artifacts to S3.
- Destroying execution containers after completion.

Runner contains no HTTP server and never communicates directly with clients.

---

# Execution Flow

```text
Client
    │
    ▼
POST /job
    │
    ▼
Dispatcher API
    │
    ▼
Validator
    │
    ▼
Submission Preparer
    │
    ▼
Upload source to S3
    │
    ▼
RabbitMQ
    │
    ▼
Runner Consumer
    │
    ▼
Local Queue
    │
    ▼
RADS Scheduler
    │
    ▼
Executor
    │
    ▼
containerd
    │
    ▼
Execution Container
    │
    ▼
Generate:

- result.json
- stdout.log
- stderr.log

    │
    ▼
Upload artifacts to S3
    │
    ▼
Execution Complete
```

---

# Live Event Pipeline

During execution, Runner continuously publishes execution events.

Typical events include:

- Queued
- Pulling Container Image
- Compiling
- Running Test 1/N
- Running Test 2/N
- Running Test N/N
- Finished

Dispatcher consumes these events and exposes them to clients through the SSE endpoint.

```text
Executor
      │
      ▼
RabbitMQ
      │
      ▼
Dispatcher
      │
      ▼
SSE
      │
      ▼
Client
```

---

# Artifact Storage

AlpineJudge stores all execution artifacts in an S3-compatible object store.

Example layout:

```text
submits/
└── sub001/
    ├── main.py
    ├── result.json
    ├── stdout.log
    └── stderr.log
```

S3 acts as the single source of truth for execution artifacts, allowing Runner instances to remain completely stateless.

---

# Scheduling

Runner uses the **Resource-Aware Dynamic Scheduler (RADS)** to determine how many execution containers may run concurrently.

RADS continuously monitors available system resources and dynamically adjusts execution capacity based on:

- Available memory
- CPU core count
- Configured oversubscription factor
- Configured memory reserve

This allows Runner to maximize hardware utilization while preventing resource exhaustion.

---

# Stateless Architecture

AlpineJudge intentionally maintains no persistent database.

Persistent data consists only of:

- Submitted source code
- Execution artifacts
- Result specifications

These are stored exclusively in S3.

Transient execution state is maintained in:

- RabbitMQ
- Runner local queue
- Running containers

This architecture allows Runner instances to be added or removed without requiring data synchronization.

---

# Horizontal Scaling

Dispatcher remains lightweight and publishes all execution jobs into a global RabbitMQ queue.

Multiple Runner instances consume from this queue independently.

```text
                Dispatcher
                     │
                     ▼
              RabbitMQ Queue
          ┌──────────┼──────────┐
          ▼          ▼          ▼
      Runner A   Runner B   Runner C
```

RabbitMQ automatically distributes jobs among available Runner instances.

---

# Design Principles

AlpineJudge is built around the following principles:

- Stateless execution
- Secure container isolation
- Native containerd integration
- Resource-aware scheduling
- Horizontal scalability
- S3-backed artifact storage
- RabbitMQ-based asynchronous execution
- Simple Linux deployment using systemd