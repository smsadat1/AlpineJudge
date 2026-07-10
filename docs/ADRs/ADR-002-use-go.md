# ADR-002: Why Go?

## Status
Accepted

## Context
The Runner is responsible for:

* Parallel execution
* RabbitMQ consumption
* Resource-aware scheduling
* Container lifecycle management
* System monitoring

## Decision
AlpineJudge is implemented in Go.

## Rationale

Go provides:
* Lightweight goroutines for concurrent execution
* Native containerd client libraries
* Strong static typing
* Simple deployment as a single binary
* Excellent support for long-running infrastructure services

While Python can also implement these features, Go better aligns with the concurrency and systems programming requirements of AlpineJudge.

## Alternatives Considered

### Python

Pros:
* Faster development
* Rich ecosystem

Cons:
* Less mature containerd support
* Concurrency model is less natural for this workload
* Requires more runtime dependencies

