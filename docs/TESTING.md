# AlpineJudge Testing Strategy

AlpineJudge follows a two-dimensional testing strategy.

Rather than treating testing as a single hierarchy, the project validates correctness from two independent perspectives:

1. **Functional Testing** — verifies correctness at increasing system scopes.
2. **Execution Testing** — verifies correctness across the Runner execution pipeline.

This approach allows failures to be isolated quickly while ensuring every architectural layer is independently verified.

---

# Functional Testing

Functional testing validates *what* is being tested.

The project follows a four-layer testing pyramid.

```
                Ultimate Test
                     ▲
                     │
              End-to-End Tests
                     ▲
                     │
           Integration Tests
                     ▲
                     │
               Unit Tests
```

## Layer 1 — Unit Tests

Tests an individual function in isolation.

Characteristics:

* No external dependencies whenever possible
* Fast execution
* High coverage
* Validate edge cases and failure paths

Examples:

* Validator functions
* Utility helpers
* Configuration parsers
* Scheduling algorithms

---

## Layer 2 — Integration Tests

Tests an entire service within a subsystem.

Characteristics:

* Multiple components working together
* May require temporary files or test harnesses
* Verifies service-level behavior

Examples:

* Container lifecycle service
* Task executor
* In-container agent
* RabbitMQ service
* MinIO service

---

## Layer 3 — End-to-End (E2E)

Tests an entire subsystem.

Characteristics:

* Starts the complete subsystem
* Exercises public interfaces
* Validates complete workflows

Examples:

* Dispatcher subsystem
* Runner subsystem

---

## Layer 4 — Ultimate Test

Tests AlpineJudge as a complete system.

This is the highest level of validation.

The Ultimate Test launches:

* Dispatcher
* RabbitMQ
* MinIO
* Runner

A real submission is sent through the Dispatcher until a final verdict is produced by the Runner.

This verifies the complete execution pipeline from client request to final result.

Because of its execution time, this test is intended for release validation rather than every CI push.

---

# Execution Testing

Execution testing validates *where* execution occurs inside the Runner.

Unlike the Dispatcher, the Runner consists of several execution layers.

Each layer is independently tested.

```
System Scheduler & Queue Consumer
                ▲
                │
Container Lifecycle Manager
                ▲
                │
Container Task Executor
                ▲
                │
      In-Container Agent
```

---

## Dispatcher

Dispatcher is comparatively simple.

Execution consists of a single logical layer.

Testing covers:

* HTTP server
* Request validation
* RabbitMQ connectivity
* Configuration loading

---

## Runner Layer 1 — In-Container Agent

The agent is responsible for execution inside the sandbox.

Dedicated test harness included.

Current coverage:

* 9 Unit Tests
* 3 Integration Tests

Integration tests validate both successful execution and expected failure scenarios.

---

## Runner Layer 2 — Container Task Executor

Responsible for:

* Creating containerd tasks
* Executing compile/run commands
* Collecting execution status

Validated using a dedicated container factory.

Current coverage:

* Integration Test

---

## Runner Layer 3 — Container Lifecycle Manager

Responsible for:

* Container creation
* Container cleanup
* Resource lifecycle

Validated through integration testing.

Current coverage:

* Integration Test

---

## Runner Layer 4 — System Scheduler & Queue Consumer

Responsible for:

* Queue consumption
* Resource scheduling
* Job orchestration
* Communication with MinIO and RabbitMQ

Validated through the Runner End-to-End test.

Dedicated repository and factory utilities provide isolated RabbitMQ and MinIO instances for testing.

---

# Continuous Verification

Every pull request and commit passes through automated verification.

Current verification pipeline includes:

* Code formatting
* Static analysis
* Linting
* Unit Tests
* Integration Tests
* Race Detection
* Code Coverage
* Benchmarks
* End-to-End Tests

Release workflows additionally execute the Ultimate Test before publishing.

---

# Design Philosophy

The testing strategy mirrors AlpineJudge's architecture.

Each execution layer is independently validated while functional tests progressively increase the scope of verification.

This separation provides several advantages:

* Fast identification of regressions
* Independent verification of architectural boundaries
* High confidence before releases
* Easier long-term maintenance

Testing is treated as a first-class engineering discipline rather than a post-development activity.
