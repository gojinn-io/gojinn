# 🔐 Gojinn Security Model

This document defines the **security model**, **trust boundaries**, and **explicit guarantees** of the Gojinn runtime.
It is intended for maintainers, contributors, auditors, and organizations running **untrusted WASM code** in production.

> **Scope:** This document describes *how Gojinn is designed to be secure*, not how to report vulnerabilities.
> For vulnerability disclosure, see `SECURITY.md`.

---

## 🎯 Security Philosophy

Gojinn is built on a strict principle:

> **Untrusted code must never be trusted — only constrained.**

The runtime assumes that **WASM modules may be buggy, malicious, or adversarial**.
Security is enforced through **sandboxing, capability-based access, and deterministic limits**, not developer intent.

Gojinn intentionally avoids relying on:
- Containers for isolation
- Kernel namespaces
- Implicit permissions

Instead, it enforces **explicit, in-process security boundaries**.

---

## 🧩 Assets to Protect

Gojinn is designed to protect the following assets:

- Host process integrity (the Go runtime)
- Other WASM modules running in the same process
- Host memory and CPU resources
- Database connections and pooled resources
- Secrets and environment variables
- Observability pipelines (logs, traces, metrics)

---

## 🧠 Threat Model

### Assumed Adversary Capabilities

The adversary may:
- Provide arbitrary WASM bytecode
- Trigger high request volume
- Attempt infinite loops or heavy CPU usage
- Attempt memory exhaustion
- Attempt unauthorized access to host capabilities
- Attempt to crash the runtime

### Out-of-Scope Threats

Gojinn does **not** attempt to protect against:
- Kernel-level exploits
- CPU speculative execution attacks (Spectre/Meltdown)
- Physical access to the machine
- Compromised host OS or hypervisor
- Malicious Go dependencies

---

## 🔒 Trust Boundaries

| Component | Trust Level |
|---------|------------|
| Gojinn Host (Go Runtime) | Trusted |
| wazero WASM Engine | Trusted |
| WASM Modules | **Untrusted** |
| User Configuration | Partially Trusted |
| External Databases | Untrusted |
| Observability Backends | Untrusted |

No WASM module is ever trusted by default.

---

## 🧱 Isolation Guarantees

Gojinn guarantees:

- **Memory Isolation:** WASM modules cannot access host memory outside their sandbox
- **Crash Isolation:** Panic or trap in a module does not crash the host
- **No Syscalls:** WASM code cannot perform OS syscalls
- **No Filesystem Access:** Disabled by default
- **No Network Access:** Disabled by default

Each execution runs inside a **sandboxed VM instance**.

---

## 🛂 Capability-Based Security

Access to host features is controlled via **explicit capabilities**.

### Capability Examples

- `kv.read`
- `kv.write`
- `db.query`
- `net.http`
- `secrets.read`
- `debug.enable`

Capabilities must be:
- Explicitly declared
- Explicitly granted
- Explicitly enforced

There are **no implicit permissions**.

---

## ⏱️ Resource Limiting

### CPU
- Execution timeouts enforced per request
- VM pooling prevents unbounded VM creation
- (Planned) Fuel-based deterministic execution

### Memory
- Per-module memory limits
- No shared heap between modules
- (Planned) Hard memory ceilings

### Concurrency
- Bounded worker pools
- Backpressure via host runtime

---

## 🔐 Secrets Handling

- Secrets are never embedded into WASM binaries
- Secrets are injected at runtime
- Secrets are scoped per module
- (Planned) Encrypted secret storage

Secrets are **read-only** by default.

---

## 🧪 Debug & Observability Safety

Debug capabilities:
- Are **opt-in**
- Are **disabled by default**
- Must never be enabled in untrusted environments

Logs and traces:
- Are structured
- Never expose secrets by default

---

## 🚨 Explicit Non-Guarantees

Gojinn does **not** guarantee:

- Protection against malicious business logic
- Safety if excessive capabilities are granted
- Defense against DDoS at the network layer
- Compliance with regulatory standards (PCI, HIPAA, etc.)

Security depends on **correct configuration**.

---

## 🔮 Roadmap Alignment

The following roadmap phases directly strengthen this model:

- **Phase 7 – The Fortress:** Fuel metering, memory walls, capability enforcement
- **Phase 10 – Code Sovereignty:** Cryptographic signing and supply chain security
- **Phase 11 – The Mesh:** Secure P2P federation

---

## 📌 Security Invariants

The following invariants must **never be broken**:

1. Untrusted code must not escape its sandbox
2. Capabilities must be explicit
3. Resource usage must be bounded
4. Module failure must not impact the host

Any change violating these invariants is a **security bug**.

---

## 🏁 Final Note

Security in Gojinn is **a first-class design constraint**, not an afterthought.

If you are extending the runtime, you are expected to:
- Understand this document
- Preserve its guarantees
- Explicitly document any new trust boundary

When in doubt: **deny by default**.