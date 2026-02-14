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

## 🛠️ PART V: DEVELOPER EXPERIENCE
*From a single binary to a resilient, self-healing mesh.*

### ⏪ Phase 16: Time-Travel Debugging (DONE v0.16.0)
- [x] **Deterministic Replay:** Record inputs to replay crashes locally.
- [x] **Snapshotting:** Save/Restore full VM state.

### 🖥️ Phase 17: Gojinn Studio (DONE v0.17.0)
- [x] **Visual Control Plane:** Web GUI for topology, metrics, and management.
- [x] **Hot Patching:** Update variables via UI.
- [x] **Language-Agnostic CLI:** Unified `gojinn` command family.

## ⚡ PART VI: THE DISTRIBUTED NERVOUS SYSTEM (NATS Saga)

### 🟢 Phase 18: The Nervous System (Core NATS) (DONE v0.18.0)
*Replacing the internal communication engine for massive scalability.*
- [x] **Embedded NATS Server:** Replaced Go channels with a production-grade embedded NATS server.
- [x] **Worker Queue Groups:** Load balancing via NATS Queue Subscriptions (Round-Robin).
- [x] **Hot Reload Protocol:** Zero-downtime updates via `_sys` control topics.
- [x] **Topology:** Basic node discovery replacing Memberlist.

### 🟠 Phase 19: The Memory (Persistence & Reliability) (DONE v0.19.1)
*Solving "Amnesia" and "Zombie Workers" failures.*
- [x] **JetStream Activation:** Enable File Store in the embedded server.
- [x] **Durable Messaging:** Replace `nats.Request` with `js.Publish` for guaranteed delivery.
- [x] **Automatic Retries:** Implement redelivery policies for failed jobs.
- [x] **Dead Letter Queues (DLQ):** Automatic handling of poisoned messages.

### 🔵 Phase 20: The Hive (True Clustering) (DONE v0.20.1)
*Solving the "Single Point of Failure".*
- [x] **Cluster Config:** Configure Routes and Gossip in NATS Server options.
- [x] **Seed URLs:** Allow passing seed nodes via Caddyfile.
- [x] **Leaf Nodes:** Implement Leaf Node architecture for Edge-to-Cloud scenarios.
- [x] **Multi-Node Testing:** Verify mesh connectivity via Docker Compose.

### 🟣 Phase 21: The Synapse (Distributed State) (DONE v0.21.0)
*Solving the "Volatile State" problem.*
- [x] **NATS Key-Value (KV):** Replace local `sync.Map` with JetStream KV.
- [x] **Global State:** Implement `host_kv_set` / `host_kv_get` backed by distributed KV.
- [x] **Consistency:** Ensure keys written on Node A are instantly readable on Node B.

### 🔴 Phase 22: The Overwatch (Distributed Observability) (DONE v0.22.0)
*Solving the "Black Box" problem in a mesh.*
- [x] **OpenTelemetry + NATS:** Inject TraceIDs into NATS message headers.
- [x] **NATS Metrics:** Export queue lag, msg/sec, and consumer status to Prometheus.
- [x] **Distributed Tracing:** Correlate Caddy RequestIDs with Worker processing across nodes.

### 🟡 Phase 23: The Fortress (Security & Hardening) (DONE v0.23.0)
*Solving the "Open Door" problem.*
- [x] **NATS Auth (NKeys):** Secure the cluster using Ed25519 signatures (NKeys) instead of plain passwords. Only authorized nodes can join.
- [x] **WASM Permissions:** Implement a "Scopes/Capabilities" system (e.g., Module A can read KV users but cannot write).
- [x] **Host Function Guard:** Block access to sensitive host functions (e.g., host_s3_put, host_net_connect) based on module configuration.

---

## 🔮 PART VII: THE NEXT GENERATION

### 🤖 Phase 24: The Agentic Interface (MCP) (DONE v0.24.0)
- [x] **Auto-MCP Generation:** Expose WASM functions as tools for Claude/OpenAI agents.
- [x] **Semantic Router:** Natural language routing to functions.

### 💎 Phase 25: The Sync Engine (Via LibSQL) (DONE v0.25.0)
- [x] **LibSQL Integration:** Replace standard SQLite driver with LibSQL server mode.
- [x] **Replication Tunnel:** Expose replication protocol safely via Caddy/WebSockets.

---

## 🏛️ PART VIII: INSTITUTIONAL GRADE SOVEREIGNTY (Completed)

### 🟢 Phase 26: Deterministic Cluster & Consensus (DONE v0.26.0)
*From distributed system to formally modeled distributed system.*
- [x] **Formal Leader Election:** Deterministic election for critical actors.
- [x] **Consistency Policies:** Configurable AP vs CP modes per namespace.
- [x] **Deterministic Failover:** Predictable actor relocation on node failure.
- [x] **Partition Policies:** Configurable behavior during network splits.
- [x] **Cluster State Machine:** Explicit distributed state model documentation.

### 🟠 Phase 27: Disaster Recovery & Snapshots (DONE v0.27.0)
- [x] **Global Snapshot Engine:** KV + JetStream + SQLite unified snapshot.
- [x] **Incremental Backups:** WAL-based incremental persistence.
- [x] **Cluster-Level Restore:** Full restore into new topology.
- [x] **Versioned State Recovery:** Rollback to previous state points.
- [x] **Encryption At-Rest:** Official disk-level cryptographic storage.

### 🔵 Phase 28: Multi-Tenant Hard Isolation (DONE v0.28.0)
*From logical isolation to operational isolation.*
- [x] **Tenant Namespaces:** Dedicated NATS subjects & KV prefixes.
- [x] **Per-Tenant Streams:** Isolated JetStream persistence.
- [x] **Resource Quotas:** CPU/RAM/IO enforcement per tenant.
- [x] **Signed Audit Logs:** Tenant-scoped cryptographically verifiable logs.

### 🟣 Phase 29: Formal Security & Transparency (DONE v0.29.0)
- [x] **Independent Security Audit:** Simulated IA-driven architectural stress test.
- [x] **Threat Model Documentation:** STRIDE-based analysis (THREAT_MODEL.md).
- [x] **SBOM Publication:** Software Bill of Materials via Syft.
- [x] **Reproducible Builds:** Using -trimpath and -buildvcs=false.
- [x] **Binary Hash Transparency:** SHA-256 Checksums for every release.

### 🔴 Phase 30: Stability Freeze — v1.0 LTS (DONE v1.0.0)
- [x] **API Freeze:** No breaking changes in Caddyfile or SDK Host Functions.
- [x] **Backward Compatibility Policy:** Strict Semantic Versioning (SemVer).
- [x] **Long Term Support:** 18-month security support window.
- [x] **Production Hardening:** Pure Go (CGO-Free) architecture for absolute portability.
- [x] **Official Whitepaper:** Comprehensive Architecture + CAP Model (WHITEPAPER.md).
- [x] **Public Benchmarks:** Verified 9.7k RPS with 0.67ms min latency.

### ❌ Explicit Non-Goals
To keep the project focused, we will **NOT** build:
* **No "SaaS-in-a-Box":** We will not build billing engines, payment gateways, or multi-tier subscription logic into the core. Gojinn is an engine, not a storefront.
* Proprietary/Vendor-locked APIs.
* Replacement for heavy OS-level containers (Docker).
* "Magic" opaque networking layers.