# ⚛️ The Gojinn Manifesto

Gojinn is not a framework.  
It is not a platform.  
It is an **architectural stance**.

Modern infrastructure favors abstraction, indirection, and control planes.
Gojinn favors **determinism, locality, and sovereignty**.

---

## 🧠 First Principles

Gojinn is built on a single, non-negotiable idea:

> **Untrusted code must be isolated, deterministic, and explicitly authorized.**

Anything that weakens this principle does not belong in the runtime.

---

## 🛡️ Why Gojinn Exists

Platforms like AWS Lambda or Cloudflare Workers solve global scale —
at the cost of:
- vendor lock-in
- opaque execution models
- network-induced latency
- loss of runtime control

Gojinn exists for environments where:
- latency is measured in microseconds
- infrastructure must be auditable
- data cannot leave the machine
- operators must understand the full stack

---

## 🎯 Killer Use Cases

### 1. High-Density Multi-Tenant Compute
Run thousands of isolated user functions on a single machine.
- No containers
- No idle cost
- No scheduler wars

### 2. Regulated & Air-Gapped Environments
Banks, governments, critical infrastructure.
- Fully self-hosted
- Offline-capable
- Auditable binaries

### 3. In-Process Middleware
Execute logic *inside* the HTTP request path.
- Validation
- Transformation
- Feature flags

No hops. No proxies. No sidecars.

### 4. True WASI-First Polyglot
If it compiles to WASM, it runs.
- Go
- Rust
- Zig
- C / C++

Stdin/Stdout is the contract.

### 5. Deterministic Resource Control
Hard CPU and memory limits.
- No “best effort”
- No noisy neighbors
- No surprise bills

---

## ❌ Explicit Non-Goals

Gojinn will **not** become:
- a container orchestrator
- a Kubernetes abstraction
- a general-purpose PaaS
- a cloud control plane

Complexity is not hidden — it is rejected.

---

> *“The fastest function is the one that never leaves the process.”*
