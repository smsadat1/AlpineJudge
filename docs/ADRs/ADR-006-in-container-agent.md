# ADR-006: Use In-Container Execution Agent

## Status
Accepted

## Context

AlpineJudge initially attempted to orchestrate compilation and
test execution directly from Runner using containerd tasks.

This introduced challenges:

- OCI Process.Args are static.
- Multiple sequential executions inside a container became difficult.
- Streaming execution status required additional coordination.
- Judge logic became tightly coupled with container lifecycle code.

## Decision

Introduce a lightweight static binary named AJRunnerAgent
inside every language image.

Runner generates an execspec.json and mounts it into the
container workspace.

AJRunnerAgent is responsible for:

- Reading execspec.json
- Executing compile phase if required
- Iterating testcases
- Comparing outputs
- Generating verdicts
- Streaming execution events via stdout
- Producing artifacts

Runner remains responsible for:

- Container lifecycle
- Scheduling
- S3 interactions
- RabbitMQ interactions
- Resource limits

## Consequences

### Positive

- Simplifies Runner implementation
- Keeps judge logic close to execution environment
- Makes SSE event streaming straightforward
- Language support remains extensible
- Avoids complex multi-process orchestration in containerd

### Negative

- Requires bundling agent into execution images
- Adds another binary to maintain
- Requires image rebuilds when agent changes