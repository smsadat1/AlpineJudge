# Executor

## Overview

Executor is the core service of the **Runner** subsystem.

Its responsibility is to receive validated execution rules from the Runner, create and manage the execution container, execute the submitted program against the provided testset, generate execution artifacts, and publish execution events.

Executor is the only service responsible for interacting directly with **containerd**.

---

## Responsibilities

Executor is responsible for:

- Pulling or retrieving cached container images.
- Building OCI runtime specifications.
- Creating and managing container lifecycle.
- Executing compilation (if required).
- Running all test cases.
- Monitoring execution timeout.
- Collecting stdout and stderr.
- Generating `result.json`.
- Uploading execution artifacts to S3.
- Publishing execution progress events.
- Cleaning up container resources.

Executor is **not** responsible for:

- Job validation.
- RabbitMQ queue consumption.
- Scheduling execution slots.
- HTTP communication.
- Result persistence outside S3.

---

## Pipeline

```text
Take ExecRules
        ↓
Ensure Container Image
        ↓
Build OCI Specification
        ↓
Create Container
        ↓
Compile (if required)
        ↓
Execute Test Cases
        ↓
Generate result.json
stdout.log
stderr.log
        ↓
Upload Artifacts to S3
        ↓
Destroy Container
```

---

## Inputs

Executor receives an `ExecRules` object from the Runner.

Typical fields include:

- Container image
- Language
- Language version
- Submission identifier
- Testset identifier
- Resource limits
- Timeout
- S3 object locations

---

## Outputs

After execution, Executor produces:

```text
result.json
stdout.log
stderr.log
```

These artifacts are uploaded to S3 under the submission directory.

Example:

```text
submits/
└── sub001/
    ├── main.py
    ├── result.json
    ├── stdout.log
    └── stderr.log
```

---

## Container Lifecycle

Executor owns the complete lifecycle of an execution container.

```text
Retrieve Image
        ↓
Create Snapshot
        ↓
Create Container
        ↓
Create Task
        ↓
Start Task
        ↓
Wait for Completion / Timeout
        ↓
Delete Task
        ↓
Delete Container
        ↓
Remove Snapshot
```

Every submission executes inside its own isolated container.

---

## Execution Model

Each submission is executed inside a single container.

Compiled languages are compiled once.

Each test case is executed as a fresh process to ensure that program state does not leak between tests.

After all test cases complete—or execution terminates due to an error or resource limit—the container is destroyed.

---

## Execution Events

Executor publishes execution events through RabbitMQ.

Typical events include:

- Container Created
- Compiling
- Running Test 1/N
- Running Test 2/N
- ...
- Finished

These events are consumed by Dispatcher to provide live progress updates through the SSE endpoint.

---

## Failure Handling

Executor detects and reports:

- Compilation Error (CE)
- Runtime Error (RE)
- Time Limit Exceeded (TLE)
- Memory Limit Exceeded (MLE)
- Output Limit Exceeded (OLE)
- Security Error (SE)
- Internal Error (IE)

Regardless of execution outcome, Executor attempts to clean up all container resources before returning.

---

## Design Notes

- Executor integrates directly with `containerd`; Docker is not used.
- gVisor (`runsc`) is used as the OCI runtime for sandbox isolation.
- Container images are cached locally by `containerd` and pulled only when absent.
- Execution artifacts are stored in S3, making the Runner stateless.
- Executor performs no scheduling decisions; scheduling is delegated to the RAD Scheduler.