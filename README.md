# AlpineJudge

![banner](docs/assets/banner.png)

A stateless, secure, multi-language code execution engine built for high performance and easy deployment.

![Status](https://img.shields.io/badge/status-ongoing-brightgreen)
![Architecture](https://img.shields.io/badge/stateless-enabled-blue)
![Security](https://img.shields.io/badge/isolation-containerd-red)


## Table of Contents
* [About](#about)
* [Getting Started](#getting-started)
* [Example Usage](#example-usage)
* [Design Philosophy](#design-philosophy)
* [Key Capabilities](#key-capabilities)
* [Non Goals](#non-goal)
* [Documentation](#documentation)


## About

Running untrusted code is not just execution — it is a security problem.

Most systems struggle with:
- unsafe container escape risks
- inconsistent runtime environments
- resource abuse (CPU/memory/time)
- lack of controlled and reliable execution orchestration

AlpineJudge solves this by treating code execution as a hardened infrastructure layer rather than a simple runtime task.


## Getting Started

AlpineJudge can be used in two ways.

### Option 1 — Use the Python SDK (Recommended)

Install the SDK:

```bash
pip install alpinejudge-sdk
```

Submit a job:

```python
from alpinejudge import Client

client = Client("<ALPINEJUDGE_URL>")

result = client.submit(
    language="python",
    version="python3.12",
    file="main.py",
    testset="ts001",
    testset_version="v1",
)

print(result.status)
```

**Replace <ALPINEJUDGE_URL> with:**

- http://localhost:4004 if you're running a local self-hosted instance.
- The URL of your organization's AlpineJudge deployment.
- The URL of a hosted AlpineJudge service.

See the [Python SDK documentation](docs/sdk/pythonexamples.md) for more examples.

---

### Option 2 — Self Host AlpineJudge

To deploy your own Dispatcher and Runner instances, follow the installation guide.

**See [INSTALLATION.md](docs/INSTALLATIONS.md)**


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



## Non-Goal 
AlpineJudge is not a contest management platform. 
It intentionally remains stateless and does not manage users, contests, submissions, or persistent application data. 
Those responsibilities belong to the integrating application. 
AlpineJudge focuses solely on validating, scheduling, executing, and evaluating code submissions.

## Documentation

Detailed technical documentation is available in `/docs`:

  - Architecture -> `docs/ARCHITECTURE.md`
  - API references -> `docs/API.md`
  - Design decisions -> `docs/adrs`
  - Subsystem documentation (dispatcher) -> `docs/subsystems/dispatcher.md`
  - Subsystem documentation (runner) -> `docs/subsystems/runner`
