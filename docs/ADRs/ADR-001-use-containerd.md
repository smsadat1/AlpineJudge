# ADR-001: Why containerd?

## Status
Accepted

## Context
AlpineJudge executes untrusted user code inside isolated containers.

The execution engine requires fine-grained control over:

* Container lifecycle
* OCI runtime configuration
* Resource limits
* Filesystem mounts
* Namespace isolation

## Decision
AlpineJudge integrates directly with **containerd** instead of using Docker.

## Rationale

Docker is an excellent developer tool that provides a higher-level interface over containerd. However, AlpineJudge is not intended to provide a developer experience—it is an execution engine.

Using containerd directly provides:

* Lower-level control over container lifecycle
* Direct access to OCI runtime configuration
* Reduced abstraction layers
* Better integration with the Runner and Executor

## Alternatives Considered

### Docker Engine

Pros:
* Easier to use
* Mature ecosystem
* Excellent CLI

Cons:
* Additional abstraction layer
* Less direct control over runtime behavior
