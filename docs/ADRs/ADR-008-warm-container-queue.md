# ADR-008: Warm Container Queue

## Status

```text
Accepted
```

---

## Context
> Each submission required acquiring or creating a container before esxecution. Although images were already cached, task/container lifecycle still contributed measurable latency. Submission execution became coupled with container lifecycle management. The Runner already knows system capacity (via the scheduler), making proactive container preparation possible.

## Decision

> Runner maintains a queue of idle, pre-warmed containers.
> Incoming submissions are assigned to an available warm container immediately.
> Container lifecycle management is delegated to a background container manager responsible for replenishing the queue according to scheduler decisions.

Important ideas:

* scheduler decides **queue size**, not individual spawns.
* submission pipeline **consumes** containers.
* container manager **produces** containers.

---

## Rationale

Why is this better?

Examples:

* Reduces execution latency by removing container startup from the critical path.
* Separates submission execution from container lifecycle.
* Makes scheduling deterministic.
* Simplifies execution flow.
* Better suited for burst traffic.

---

## Consequences

Positive:

* Lower user-visible latency.
* Predictable execution.
* Simpler execution pipeline.
* Scheduler only manages desired queue depth.

Negative:

* Idle containers consume memory.
* Requires background queue maintenance.
* Queue depth tuning becomes important.
* More complex startup phase.

---

## Alternatives Considered

### Spawn container per submission

Rejected because:

* startup latency becomes part of every submission.

---

### Pool only container images

Rejected because:

* image caching removes pull latency but not container creation latency.

---

### Unlimited permanent containers

Rejected because:

* wastes memory and scales poorly.

---


**The scheduler no longer decides when to spawn containers for individual submissions. Instead, it determines the desired warm-container queue depth, while a dedicated container manager maintains that target asynchronously.**

**Container lifecycle became an asynchronous background responsibility instead of a synchronous submission responsibility.**

