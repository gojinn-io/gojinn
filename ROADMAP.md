# 🗺️ Project Roadmap: The Sovereign Cloud

>This roadmap describes optional and modular evolution paths.
Not all phases are intended to live in the core runtime.


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
*Democratizing access and securing the perimeter.*

### 🟤 Phase 6: Polyglot Support (Active Priority)
*Don't force users to learn Rust/Go. Support the ecosystem.*
- [x] **JavaScript/TypeScript Adapter:** Integrate Javy (QuickJS) for JS support.
- [x] **Python Adapter:** Support for Python 3.12 (VMware Build).
- [x] **PHP Adapter:** Support for PHP 8.2 (VMware Build) via CGI-style execution.
- [x] **Ruby Adapter:** Support for Ruby 3.2 (VMware Build) with library loading.
- [x] **.NET / C# Adapter:** Support for C# 8.0 via WASI SDK (Enterprise).
- [x] **Unified Build System:** Implemented via `Makefile` (Replaces `gojinn build` CLI for flexibility).

### 🛡️ Phase 7: The Fortress (Security Hardening)
*Mathematical guarantees against bad code and attacks.*
- [ ] **Fuel Metering:** Deterministic CPU limits to prevent infinite loops.
- [ ] **Memory Wall:** Strict per-sandbox RAM limits to prevent leaks.
- [ ] **Capability-Based Security:** Explicit permissions (e.g., "Can this function access KV?").
- [ ] **Secrets Management:** Encrypted ENV variables integration.

### ⚡ Phase 8: Async & Event-Driven
*Handling tasks beyond the HTTP request lifecycle.*
- [ ] **Cron Triggers:** Native scheduler (`@every 5m`).
- [ ] **Fire-and-Forget:** Async execution queue (Internal).
- [ ] **Dead Letter Queues:** Automatic retries for failed background jobs.

### 🧠 Phase 9: Edge AI Inference
*Native Intelligence without external APIs.*
- [ ] **Host LLM Bindings:** Embed `llama.cpp` to allow WASM to call Local AI.
- [ ] **Zero-Copy Inference:** Shared memory between WASM and Model.

---

## 🌐 PART III: THE SOVEREIGN CLOUD (Future)
*Distributed systems, Blockchain, and Code Sovereignty.*

### 🔗 Phase 10: Code Sovereignty
*Trust, Verify, and Sign.*
- [ ] **Cryptographic Signing:** Blockchain/Ledger integration to verify WASM authorship.
- [ ] **Supply Chain Security:** Gojinn only runs modules signed by trusted keys.
- [ ] **Immutable Registry:** Hash-based addressing for functions.

### 🕸️ Phase 11: The Mesh (P2P Federation)
*Scale without a master node.*
- [ ] **P2P Discovery:** Gossip protocol (Memberlist/WireGuard) for node discovery.
- [ ] **Cluster Storage:** Sync Certificates and KV across nodes.
- [ ] **Edge Routing:** Automatic request routing to the nearest available node.

### 🎭 Phase 12: Stateful Actors
*Real-time applications without external DBs.*
- [ ] **Actor Model:** Durable Objects implementation (State lives in RAM/Disk).
- [ ] **Websockets Support:** massive concurrent connections handling.
- [ ] **Global Locking:** Distributed consistency for actors.

---

## 🔮 PART IV: THE NEXT GENERATION (Visionary)
*Redefining Developer Experience.*

### ⏪ Phase 13: Time-Travel Debugging
- [ ] **Deterministic Replay:** Record inputs to replay crashes locally.
- [ ] **Snapshotting:** Save/Restore full VM state.

### 🖥️ Phase 14: Gojinn Studio
- [ ] **Visual Control Plane:** Web GUI for topology, metrics, and management.
- [ ] **Hot Patching:** Update variables via UI.
- [ ] **Language-Agnostic CLI:** Unified gojinn build command (replacing Makefile).

### 🤖 Phase 15: The Agentic Interface (MCP)
- [ ] **Auto-MCP Generation:** Expose WASM functions as tools for Claude/OpenAI agents.
- [ ] **Semantic Router:** Natural language routing to functions.

### 💎 Phase 16: The Sync Engine (Local-First)
- [ ] **SQLite Replication Protocol:** Sync browser-based SQLite with Server SQLite.
- [ ] **CRDT Integration:** Conflict-free data merging for offline-first apps.

---

### ❌ Explicit Non-Goals
To keep the project focused, we will **NOT** build:
* Proprietary/Vendor-locked APIs.
* Replacement for heavy OS-level containers (Docker).
* "Magic" opaque networking layers.