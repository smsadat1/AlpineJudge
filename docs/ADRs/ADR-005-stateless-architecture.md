# ADR-005: Why Stateless?

**Decision**
> AlpineJudge does not own application state.

**Responsibilities intentionally excluded:**

* User management
* Contest management
* Authentication
* Submission history
* Scoreboards
* Persistent business data

**Rationale**
AlpineJudge is an execution engine, not a contest platform.
Business state is managed by the integrating application.

This keeps AlpineJudge:
* Easy to deploy
* Easy to scale horizontally
* Easier to test
* Easier to integrate into different systems

