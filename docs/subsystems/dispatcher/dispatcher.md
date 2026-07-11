# Dispatcher

## Overview

Dispatcher is one of the two major subsystems of **AlpineJudge**.

Its responsibility is to receive job requests from clients, validate them, prepare submissions for execution, and enqueue jobs for the Runner subsystem.

The Dispatcher does **not** execute user code.

---

## Architecture

Dispatcher consists of three services:

1. Validator
2. Config Manager
3. RabbitMQ Producer

```text
Client
    │
    ▼
+----------------------+
|      Dispatcher      |
|                      |
|  Validator           |
|  Config Manager      |
|  RabbitMQ Producer   |
+----------+-----------+
           │
           ▼
      RabbitMQ Queue
```

---

## Request Flow

```text
Client
    │
    ▼
Validator
    │
    ▼
Submission Preparation
    │
    ▼
RabbitMQ Producer
    │
    ▼
RabbitMQ
```

---

## Validator

The Validator is responsible for ensuring an incoming submission is valid before it is accepted for execution.

Validation includes:

- Job ID uniqueness
- Language availability
- Language version availability
- File extension validation
- Filename validation (`main.*`)
- Testset existence
- Testset version existence

If any validation fails, the request is rejected and no job is produced.

---

## Submission Preparation

Submission preparation is performed after successful validation.

Responsibilities:

1. Generate a unique `submission_id`.
2. Rename the submitted file:

```text
main.*
    ↓
{job_id}/{submission_id}.*
```

3. Upload the source file to S3.
4. Construct the Job Specification.

---

## Job Specification

After submission preparation, Dispatcher creates a `JobSpec` that will be sent to the Runner subsystem.

Example:

```json
{
    "job_id": "job_001",
    "submission_id": "sub_001",

    "language": "cpp",
    "version": "gcc13",

    "s3_key": "submissions/job_001/sub_001.cpp",

    "testset_id": "ts01",
    "testset_version": "v3"
}
```

The `JobSpec` is the communication contract between Dispatcher and Runner.

---

## RabbitMQ Producer

The RabbitMQ Producer publishes validated job specifications to the global RabbitMQ queue.

Responsibilities:

- Serialize `JobSpec`
- Publish message
- Report publishing failures

Dispatcher considers a submission accepted only after the job has been successfully published.

---

## Config Manager

The Config Manager is responsible for loading and providing runtime configuration.

It parses `config.yml` during startup and exposes configuration to other Dispatcher services.

Example configuration includes:

- RabbitMQ connection
- S3 configuration
- Supported languages
- Supported language versions
- Available testsets
- Runner configuration

Other Dispatcher services consume configuration through the Config Manager rather than reading configuration files directly.

---

## Responsibilities

Dispatcher is responsible for:

- Receiving job requests
- Validating submissions
- Preparing submissions
- Uploading source files
- Producing `JobSpec`
- Publishing jobs to RabbitMQ

Dispatcher is **not** responsible for:

- Executing code
- Compiling programs
- Running test cases
- Scheduling execution
- Producing verdicts

Those responsibilities belong to the **Runner** subsystem.

---

## Design Philosophy

Dispatcher acts as the admission layer of AlpineJudge.

Its primary goals are:

- Reject invalid submissions early.
- Keep Runner free from input validation.
- Produce a standardized `JobSpec`.
- Decouple client requests from execution using RabbitMQ.

The Dispatcher performs lightweight processing only; all code execution is delegated to the Runner subsystem.