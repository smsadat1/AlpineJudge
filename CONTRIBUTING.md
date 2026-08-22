# Contributing to AlpineJudge

Thank you for your interest in contributing to AlpineJudge.

AlpineJudge is a code-execution and judging platform with a Go-based execution engine and a Python SDK. Different parts of the project have very different levels of complexity, so contributions are classified according to the subsystem being modified.

The goal of this document is to make the expectations clear before you start working on a contribution.

---

## Contribution Areas

AlpineJudge has two major contribution surfaces:

### 1. SDK and Documentation

The Python SDK and documentation are the most accessible areas for contribution.

Examples include:

* Python SDK improvements
* API ergonomics
* SDK typing
* SDK examples
* SDK tests
* Documentation improvements
* Deployment documentation
* API documentation
* Tutorials and usage examples
* Improvements specific to college/platform integration

These contributions generally do not require detailed knowledge of the execution engine internals.

The college development team is particularly encouraged to contribute in this area and adapt the SDK to the needs of the contest platform.

---

### 2. Go Execution Engine

The execution engine is substantially more complex and operates close to the operating system and container runtime.

Contributors working on the engine should have a solid understanding of:

* Go
* Goroutines and channels
* `select`
* Contexts and cancellation
* Synchronization
* Race conditions
* Deadlocks
* Concurrent resource management
* Error propagation and failure handling
* Linux processes and signals
* Linux namespaces and cgroups
* OCI container concepts
* Container runtimes
* containerd
* Unix sockets and IPC
* Container lifecycle management
* Resource isolation
* Reliable messaging
* Testing concurrent systems
* CI and regression testing

The engine should not be treated as a conventional application where an apparently local change can be assumed to have only local consequences.

Changes may affect container lifecycle, IPC, resource management, concurrency, failure recovery, or security boundaries.

If you are unfamiliar with these concepts, start with the SDK, documentation, or test infrastructure before modifying the execution engine.

---

# Contribution Difficulty

## Easy — Documentation and SDK

Examples:

* Fix documentation errors
* Improve deployment instructions
* Add SDK examples
* Improve SDK documentation
* Improve API documentation
* Add or improve SDK tests
* Improve developer experience

Expected process:

1. Make the change.
2. Verify that documentation/examples remain correct.
3. Run relevant tests where applicable.
4. Submit a pull request.

---

## Medium — Bug Fixes and Test Improvements

Examples:

* Fix an existing bug
* Improve error handling
* Add regression tests
* Improve test infrastructure
* Improve CI
* Fix SDK or engine behavior without changing major architecture

Expected process:

1. Identify and reproduce the problem.
2. Understand the relevant subsystem before modifying it.
3. Make the smallest appropriate fix.
4. Run the tests related to the affected subsystem.
5. Add a new regression test when necessary to prevent the problem from returning.
6. Run broader tests when the change can affect other subsystems.
7. Update documentation when behavior or configuration changes.
8. Explain the root cause and verification performed in the pull request.

A bug fix should ideally leave behind a test that demonstrates the failure no longer occurs.

---

## Hard — New Engine Features and Architectural Changes

Examples:

* Container lifecycle changes
* Warm-container orchestration changes
* Resource enforcement
* IPC changes
* Execution pipeline changes
* Isolation/security changes
* Retry and failure-recovery mechanisms
* Changes to concurrency architecture
* Changes involving containerd or OCI behavior
* Major API or subsystem changes

Expected process:

1. Understand the existing architecture and affected subsystem.
2. Identify relevant invariants and failure modes.
3. Implement the feature with appropriate tests.
4. Run the complete test suite.
5. Add tests specifically covering the new behavior.
6. Add regression coverage for newly discovered failure cases.
7. Update relevant documentation.
8. Add an ADR when the change introduces a significant architectural decision.
9. Ensure CI passes.
10. Clearly describe the design, trade-offs, failure handling, and testing performed in the pull request.

Large architectural changes should be discussed before implementation whenever practical.

---

# Testing Requirements

Tests are an important part of AlpineJudge development.

A contribution should not rely solely on manual verification when the behavior can be tested automatically.

Depending on the change, testing may include:

* Unit tests
* Integration tests
* End-to-end tests
* Regression tests
* Concurrent execution tests
* Container lifecycle tests
* Failure and retry tests
* SDK tests
* Full CI validation

When modifying behavior that previously failed, prefer adding a regression test that reproduces the original failure.

For changes affecting multiple subsystems, running only a narrow test may not be sufficient.

---

# Pull Requests

A pull request should explain:

* What changed
* Why it changed
* Which subsystem is affected
* How the change was tested
* Any new tests added
* Any relevant failure cases
* Documentation updated
* Architectural implications, if any

For non-trivial engine changes, include enough implementation context for reviewers to understand the reasoning behind the design.

Please avoid combining unrelated changes into a single pull request.

---

# Architectural Changes

AlpineJudge uses Architecture Decision Records (ADRs) to document significant architectural decisions.

An ADR may be required when a contribution:

* Introduces a new subsystem
* Changes a major execution flow
* Changes storage or messaging architecture
* Changes container orchestration behavior
* Changes isolation or security architecture
* Introduces a significant new dependency
* Reverses an existing architectural decision

When appropriate, discuss the proposed design before implementation.

---

# Security

Security-sensitive changes require additional care.

Do not publicly disclose an unpatched vulnerability through a pull request or issue.

If you discover a vulnerability involving:

* sandbox escape
* cross-submission data access
* verdict manipulation
* container isolation
* IPC
* resource isolation
* credential or artifact exposure

follow the project's security reporting process described in `SECURITY.md`.

Security fixes should include regression tests whenever practical.

---

# Code Quality

Contributions should follow the conventions already established in the affected subsystem.

Prefer:

* Small, focused changes
* Explicit error handling
* Clear ownership of resources
* Deterministic behavior
* Testable code
* Minimal unnecessary abstraction
* Existing project patterns over introducing new patterns without justification

Do not optimize for cleverness at the expense of correctness.

In the execution engine, correctness, isolation, failure handling, and observability take priority over superficial code brevity.

---

# Before Opening a Pull Request

Please verify:

* [ ] The change is focused and has a clear purpose.
* [ ] Relevant tests pass.
* [ ] Regression tests were added where necessary.
* [ ] New behavior has appropriate test coverage.
* [ ] Documentation was updated where necessary.
* [ ] ADRs were added where necessary.
* [ ] CI passes.
* [ ] No unrelated changes are included.
* [ ] Security implications have been considered.

For engine changes, additionally verify that concurrency, cancellation, resource ownership, container lifecycle, and failure behavior have been considered where applicable.

---

# Final Note

AlpineJudge is infrastructure intended to execute untrusted code.

A small change can therefore have consequences far beyond the function being modified.

When in doubt, investigate the behavior, reproduce it, write a test, and understand the underlying mechanism before changing it.

Thank you for contributing.
