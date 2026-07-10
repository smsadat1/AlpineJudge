# ADR-004: Why RabbitMQ?

## Status
Accepted

## Context
Dispatcher and Runner communicate asynchronously through job specifications.
Reliable delivery is required.

## Decision
RabbitMQ is used as the messaging system.

## Rationale

RabbitMQ provides:
* Durable queues
* Reliable message delivery
* Acknowledgements
* Retry semantics
* Natural support for multiple Runner instances

These properties make it suitable for decoupling Dispatcher and Runner.

## Alternatives Considered

### Redis Pub/Sub

Pros:
* Lightweight
* Fast

Cons:
* Messages are transient
* No built-in delivery guarantees
* Not designed as a durable work queue

### gRPC

Pros:
* Strongly typed RPC
* Efficient communication

Cons:
* RPC is not a queue
* No durable message storage
* Additional coordination required for horizontal scaling
