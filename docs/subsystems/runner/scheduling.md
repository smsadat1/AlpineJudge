# RADS - Resource-Aware Dynamic Scheduler

## Overview

RADS (Resource-Aware Dynamic Scheduler) is the scheduling subsystem of Alpine Judge.

Its purpose is to dynamically control execution concurrency based on host resource availability while preventing resource exhaustion.

RADS operates on a per-runner basis.

```text
1 Runner Daemon = 1 Host
```

Each runner independently calculates its execution capacity and admits jobs accordingly.

---

## Goals

* Prevent CPU overcommitment.
* Prevent memory exhaustion.
* Adapt concurrency to current system conditions.
* Remain operational during resource pressure.
* Recover automatically when resources become available.
* Avoid container admission beyond safe limits.

---

## Constants

```text
OVER_SUB_FACTOR = 2

SLOT_FLOOR = 0
```

---

## Derived Values

```text
SLOT_BASELINE = CpuCoreCount

SLOT_CEILING =
    CpuCoreCount * OVER_SUB_FACTOR
```

Example:

```text
Host:
    12 CPU Cores

SLOT_BASELINE = 12
SLOT_CEILING  = 24
```

---

## Runtime Values

### Memory Slots

Represents the maximum number of containers that can be safely admitted based on available memory.

```text
memorySlots = (0.80 * totalMemoryMB) / MemoryLimitPerContainerMB
```

20% of host memory is reserved for:

* Operating System
* RabbitMQ Client
* Runner Daemon
* Container Runtime
* Miscellaneous Services

### Available Slots

Final admission capacity.

```text
availableSlots =
    clamp(
        min(memorySlots, SLOT_CEILING),
        SLOT_FLOOR,
        SLOT_CEILING
    )
```

### Used Slots

```text
usedSlots = runningContainers
```

### Idle Slots

```text
idleSlots = availableSlots - usedSlots
```

---

## Scheduler States

### NORMAL

```text
availableSlots >= SLOT_BASELINE
```

Characteristics:

* Full operational capacity.
* New jobs admitted immediately.
* Container execution proceeds normally.

---

### DEGRADED

```text
0 < availableSlots < SLOT_BASELINE
```

Characteristics:

* Resource pressure detected.
* Concurrency reduced.
* New jobs continue to be processed within reduced limits.

---

### CRITICAL

```text
availableSlots == SLOT_FLOOR
```

Characteristics:

* No safe execution capacity available.
* New job admission suspended.
* Existing containers continue running.
* Scheduler continues monitoring for recovery.

---

## Scheduling Algorithm

### Capacity Recalculation

Monitor Service continuously reports:

```text
CPU Usage
Memory Usage
System Load
```

RADS periodically recalculates:

```text
memorySlots
availableSlots
idleSlots
```

---

### Job Admission

While:

```text
idleSlots > 0
```

RADS may:

```text
Pull Job
Start Container
```

until:

```text
idleSlots == 0
```

---

### Capacity Exhaustion

When:

```text
usedSlots >= availableSlots
```

RADS stops admitting new jobs.

Jobs remain buffered in RabbitMQ.

Running containers continue execution.

---

### Capacity Reduction

If:

```text
availableSlots < usedSlots
```

RADS:

```text
Does NOT terminate running containers.
```

Instead:

```text
Stop admitting new jobs.
Wait for running containers to finish.
Resume admissions when:

usedSlots <= availableSlots
```

---

## Recovery Model

RADS is designed to be self-recovering.

As long as:

* Runner Daemon remains operational.
* Host Operating System remains operational.
* RabbitMQ remains reachable.

resource pressure automatically transitions:

```text
NORMAL
    ↓
DEGRADED
    ↓
CRITICAL
    ↓
DEGRADED
    ↓
NORMAL
```

without manual intervention.

RADS prefers admission throttling over service failure.

---

## Design Principle

```text
Throttle.
Do not overcommit.
Do not crash.
Recover automatically.
```
