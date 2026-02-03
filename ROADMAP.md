# 🗺️ Project Roadmap: The Sovereign Cloud

> This roadmap describes optional and modular evolution paths.
> Not all phases are intended to live in the core runtime.

This roadmap transforms **Gojinn** from a high-performance runtime into a **Sovereign Cloud Platform**.
Our goal is to replace the fragmented stack of AWS Lambda + SQS + RDS + Kubernetes with a single, secure, and intelligent binary.

---

## 🏗️ PART I: THE FOUNDATION (Completed)
*The Core Engine is stable, fast, and production-ready.*

### 🟢 Phase 0: Foundation (DONE)
- [x] **In-process WASM execution** via `wazero`.
- [x] **Stdin / Stdout model** for request-response.
- [x] **Zero idle memory** footprint.
- [x] **Crash-safe execution** (panic isolation).

### 🟡 Phase 1: Workers Parity (DONE v0.1.0-alpha)
- [x] **Structured HTTP Context:** JSON/Proto Context mapping.
- [x] **Header Mutation & Status Codes.**
- [x] **Per-route Binding** via Caddyfile.

### 🔵 Phase 2: Enterprise Observability (DONE v0.2.0)
- [x] **Structured Logging Interface:** JSON logs for Datadog/Loki.
- [x] **OpenTelemetry Tracing:** Distributed tracing context.
- [x] **Prometheus Metrics:** Native duration and memory histograms.
- [x] **CPU Budgeting:** Basic timeouts.

### 🟣 Phase 3: High Performance (DONE v0.3.0)
- [x] **VM Pooling (Worker Pool):** Reusable warm sandboxes (<1ms latency).
- [x] **JIT Caching:** Global module compilation cache.
- [x] **Benchmarks:** Verified performance against Docker.

### 🟠 Phase 4: Host Capabilities (DONE v0.4.0)
- [x] **Host-Managed DB Pool:** Postgres/MySQL connection multiplexing.
- [x] **Gojinn KV:** In-memory key-value store.
- [x] **SQLite Sidecar:** Zero-latency local database support.
- [x] **Secure Debug Mode:** Browser-based debugging headers.

### 🔴 Phase 5: Production & Operations (DONE v0.5.0)
- [x] **Optimized Build Pipeline:** Makefile for lean binaries.
- [x] **Systemd Integration:** Native Linux service support.
- [x] **Log Rotation:** Native Caddy log management.
- [x] **Git-Push-to-Deploy:** Zero-downtime deployment scripts.

---

## 🚀 PART II: THE EXPANSION (Current Focus)
*Democratizing access, securing the perimeter, and enabling intelligence.*

### 🟤 Phase 6: Polyglot Support (DONE v0.6.0)
*Don't force users to learn Rust/Go. Support the ecosystem.*
- [x] **JavaScript/TypeScript Adapter:** Integrate Javy (QuickJS) for JS support.
- [x] **Python Adapter:** Support for Python 3.12 (VMware Build).
- [x] **PHP Adapter:** Support for PHP 8.2 (VMware Build) via CGI-style execution.
- [x] **Ruby Adapter:** Support for Ruby 3.2 (VMware Build) with library loading.
- [x] **.NET / C# Adapter:** Support for C# 8.0 via WASI SDK (Enterprise).
- [x] **Unified Build System:** Implemented via `Makefile`.

### 🛡️ Phase 7: The Fortress (Security Hardening) (DONE v0.7.0)
*Mathematical guarantees against bad code and attacks.*
- [x] **Fuel Metering:** Deterministic CPU limits (via Strict Timeouts).
- [x] **Memory Wall:** Strict per-sandbox RAM limits to prevent leaks.
- [x] **Capability-Based Security:** Explicit permissions (File System Mounts).
- [x] **Secrets Management:** ENV variables integration via Caddyfile.

### 💾 Phase 8: Data Persistence (DONE v0.8.0)
*Functions need to remember things.*
- [x] **WASI-Virt Integration:** Virtualize file systems for persistent storage.
- [x] **SQLite Mounts:** Allow functions to request a private SQLite file per tenant.
- [x] **Object Storage Bindings:** Native S3-compatible interface for WASM (MinIO/AWS).

### ⚡ Phase 9: Async & Event-Driven (DONE v0.9.0)
*Handling tasks beyond the HTTP request lifecycle.*
- [x] **Cron Triggers:** Native scheduler (`@every 5m`).
- [x] **Fire-and-Forget:** Async execution queue (Internal).
- [x] **Dead Letter Queues:** Automatic retries for failed background jobs.
- [x] **MQTT/Webhook Triggers:** Connect IoT and external events to functions.

### 🧠 Phase 10: Hybrid AI Engine (DONE v0.10.0)
*Democratizing AI access within the sandbox.*
- [x] **Host AI Bindings:** Unified interface for WASM to call LLMs.
- [x] **Provider Agnostic:** Support via Caddyfile for OpenAI, Anthropic, Gemini, and **Ollama (Local/Free)**.
- [x] **Smart Caching:** Semantic caching to reduce inference costs and latency.

---

## 🛡️ PART III: OPERATIONAL MATURITY
*Tools required to run Gojinn safely on the open internet.*

### 🚦 Phase 11: The Gatekeeper (Traffic Control) (DONE v0.11.0)
*Protecting the Sovereign Cloud from abuse.*
- [x] **Tenant Identity:** Simple Authentication (Basic Auth / Bearer Token) via Caddyfile.
- [x] **Rate Limiting:** Per-function or per-token request limits.
- [x] **Egress Filtering:** Control which external URLs functions can access.
- [x] **CORS Management:** Granular browser access control.

### 📊 Phase 12: The Accountant (Telemetry & Quotas) (DONE v0.12.0)
*Knowing where resources are going.*
- [x] **Usage Events:** Emit structured events (Duration, RAM, AI Tokens) for external analysis.
- [x] **Hard Quotas:** Automatically kill functions exceeding daily/monthly resource caps.
- [x] **Plugin System:** Allow WASM middlewares for custom logging/telemetry.

---

## 🌐 PART IV: THE SOVEREIGN CLOUD (Future)
*Distributed systems, Blockchain, and Code Sovereignty.*

### 🔗 Phase 13: Code Sovereignty (DONE v0.13.0)
*Trust, Verify, and Sign.*
- [x] **Cryptographic Signing:** Native Ed25519 signing embedded in WASM binary (replaced Blockchain for offline sovereignty).
- [x] **Supply Chain Security:** Runtime strictly enforces signature verification via Caddyfile policy.
- [x] **Integrity Guarantee:** Mathematical proof that code hasn't been tampered with (Signature validates Content Hash).

### 🕸️ Phase 14: The Mesh (P2P Federation) (DONE v0.14.0)
*Scale without a master node.*
- [x] **P2P Discovery:** Gossip protocol (Memberlist/WireGuard) for node discovery.
- [x] **Cluster Storage:** Sync Certificates and KV across nodes.
- [x] **Edge Routing:** Automatic request routing to the nearest available node.

### 🎭 Phase 15: Stateful Actors (DONE v0.15.0)
*Real-time applications without external DBs.*
- [x] **Actor Model:** Durable Objects implementation (State lives in RAM/Disk).
- [x] **Websockets Support:** massive concurrent connections handling.
- [x] **Global Locking:** Distributed consistency for actors.

---

## 🔮 PART V: THE NEXT GENERATION (Visionary)
*Redefining Developer Experience.*

### ⏪ Phase 16: Time-Travel Debugging (DONE v0.16.0)
- [x] **Deterministic Replay:** Record inputs to replay crashes locally.
- [x] **Snapshotting:** Save/Restore full VM state.

### 🖥️ Phase 17: Gojinn Studio
- [ ] **Visual Control Plane:** Web GUI for topology, metrics, and management.
- [ ] **Hot Patching:** Update variables via UI.
- [ ] **Language-Agnostic CLI:** Unified `gojinn` command family.

### 🤖 Phase 18: The Agentic Interface (MCP)
- [ ] **Auto-MCP Generation:** Expose WASM functions as tools for Claude/OpenAI agents.
- [ ] **Semantic Router:** Natural language routing to functions.
- [ ] **Self-Healing:** AI-driven log analysis and fix suggestions.

### 💎 Phase 19: The Sync Engine (Local-First)
- [ ] **SQLite Replication Protocol:** Sync browser-based SQLite with Server SQLite.
- [ ] **CRDT Integration:** Conflict-free data merging for offline-first apps.

---

### ❌ Explicit Non-Goals
To keep the project focused, we will **NOT** build:
* **No "SaaS-in-a-Box":** We will not build billing engines, payment gateways, or multi-tier subscription logic into the core. Gojinn is an engine, not a storefront.
* Proprietary/Vendor-locked APIs.
* Replacement for heavy OS-level containers (Docker).
* "Magic" opaque networking layers.