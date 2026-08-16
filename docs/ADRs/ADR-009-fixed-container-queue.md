# ADR-009: Fixed Contianer Queue

## Statud
```text
Accepted
```

---

## Context 
> Earlier plan for container queue was dynamic queue sizing based on scheduler's decisions using system metrics. But dynamic queue isn't supported in Go's chan, so on one hand it needed custom workaround over Go chans and the other hand keeping sync between system metrics and scheduler's decisions in parallel. Which made determinism hard for warm container queue scaling.

## Decision
> RADs (Resource Aware Dynamic Scheduler) & system metrics logging is removed.
> Used CONTAINER_QUEUECAP from env vars to determine fixed warm container queue size. 

---

## Consequences

Postivies:
* Removed complexity of managing extra worker for RADS & system metrics
* Less surface for potential race conditions & concurrency bugs 
* Determinstic queueing 

Negatives:
* Likely to overwhelm system resource easily if CONTAINER_QUEUECAP is set too high.
* Can't autoscale container creation dynamically.


