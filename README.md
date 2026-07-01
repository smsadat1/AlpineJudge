# AlpineJudge

![banner](docs/assets/banner.png)

A stateless, secure, multi-language code execution engine built for high performance and easy deployment.

![Status](https://img.shields.io/badge/status-active-brightgreen)
![Architecture](https://img.shields.io/badge/microservices-enabled-blue)
![Sandbox](https://img.shields.io/badge/isolation-gVisor-red)


## Table of Contents
* [About](#about)
* [Getting Started](#getting-started)
* [Example Usage](#example-usage)
* [Design Philosophy](#design-philosophy)
* [Key Capabilities](#key-capabilities)
* [Non Goals](#non-goals)
* [Documentation](#documentation)
* [Name](#name)


## About

Running untrusted code is not just execution — it is a security problem.

Most systems struggle with:
- unsafe container escape risks
- inconsistent runtime environments
- resource abuse (CPU/memory/time)
- lack of controlled and reliable execution orchestration

AlpineJudge solves this by treating code execution as a hardened infrastructure layer rather than a simple runtime task.


## Getting Started

  - Setup CLI
  ```
  coming soon ...
  ```

  - Setup server (self-host)
  ```
  coming soon ...
  ```


## Example Usage
 ```
  coming soon ...
  ```


## Design Philosophy

AlpineJudge is designed around statelessness, isolation, predictability and reproducibility when executing untrusted code.

The system prioritizes:
  - strong runtime isolation
  - deterministic execution environments
  - clear separation of concerns across services


## System overview

AlpineJudge is composed of two subsystems:

 - Dispatcher -> request handling and orchestration 
 - Runner -> isolated code execution engine

Execution flow: 

  ` Client -> Dispatcher -> RunnerService -> Sandbox (gVisor) `

## Key Capabilities

  - Multi-language execution (Python, C/C++, Go, Java)
  - Versioned runtime support (e.g. Python 3.10 C++17 Go 1.22 )
  - Secure sandboxed execution using gVisor
  - Websocket based execution status streaming
  - Historical records of executions with timestamps



## Non-Goal 
AlpineJudge is not a contest management platform. 
It intentionally remains stateless and does not manage users, contests, submissions, or persistent application data. 
Those responsibilities belong to the integrating application. 
AlpineJudge focuses solely on validating, scheduling, executing, and evaluating code submissions.

## Documentation

Detailed technical documentation is available in `/docs`:

  - Architecture -> `docs/architecture.md`
  - Execution engine -> `docs/runner.md`
  - Security model -> `docs/security.md`
  - API references -> `docs/api.md`
  - Design decisions -> `docs/design-decisions.md`
