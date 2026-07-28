## ADR-007: Unified Event Streaming Protocol (UESP)

### Status
Accepted

### Context


* Execution Manager previously handled multiple communication paths.
* Final verdict, stdout, stderr, and execution events followed separate flows.
* Result generation waited for container teardown and artifact uploads, increasing latency.
* Multiple communication mechanisms complicated future protocol evolution.

---

### Decision


Introduce **UESP (Unified Event Streaming Protocol)** as the single communication protocol between **ajagent** and the **Execution Manager** over a Unix Domain Socket.

Every message exchanged is represented as a UESP Event.

The Execution Manager is responsible for routing each event:

* Result → RabbitMQ
* Stdout → S3
* Stderr → S3
* Progress events → RabbitMQ
* Future event types → appropriate backend

---

### Consequences


### Positive

* Single IPC protocol.
* Simplified agent implementation.
* Lower execution latency by removing post-execution processing from the critical path.
* Easily extensible with new event types.
* Cleaner separation between event production (Agent) and event routing (Execution Manager).

### Negative

* Event protocol becomes an internal compatibility boundary.
* Execution Manager now owns routing logic for every event type.
* Protocol versioning may become necessary in the future.

---

### Alternatives Considered


* Separate sockets for stdout/stderr/events.
* Stdout/stderr collected after container exit.
* Result generated after artifact upload.
* Independent streaming implementations.

Rejected due to increased complexity and higher latency.

---

### Future Work

* Support protocol versioning.
* Binary framing if JSON becomes a bottleneck.
* Compression for large stdout/stderr streams.
* Resource telemetry events.
* Compiler progress events.

