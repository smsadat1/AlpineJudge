# ADR-003: Why S3?

## Status
Accepted

## Context

AlpineJudge stores:
* Source submissions
* Testsets

These artifacts must be accessible by all Runner instances.

## Decision
S3-compatible object storage is the system of record for execution artifacts.

## Rationale

Object storage provides:
* Shared storage across horizontally scaled runners
* High durability
* Efficient handling of large binary objects
* Separation of storage from compute

Keeping artifacts in object storage allows AlpineJudge itself to remain stateless.

## Alternatives Considered

### SQLite

Pros:
* Simple deployment

Cons:
* Not suitable for multiple Runner instances
* Local file synchronization becomes problematic

### PostgreSQL / MySQL

Pros:
* Centralized storage

Cons:
* Database is not optimized for storing source files and testsets
* Introduces persistent application state into AlpineJudge

