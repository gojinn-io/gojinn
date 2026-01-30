# 🗺️ Project Roadmap

This roadmap transforms Gojinn from a runtime into a Global Serverless Platform capable of replacing legacy stacks and scaling to the edge.

## 🟢 Phase 0: Foundation
*Establish Gojinn as a stable, deterministic, production-grade runtime*

- [x] **In-process WASM execution** via `wazero`.
- [x] **Stdin / Stdout model** for request-response.
- [x] **Zero idle memory** footprint.
- [x] **Crash-safe execution** (panic isolation).
- [x] **Explicit execution timeouts** configuration.
- [x] **Memory limits** per execution.
- [x] **Documentation:** Architecture deep-dive & Transparent benchmarks.

## 🟡 Phase 1: Workers Parity
*Allow conceptual migration from Cloudflare Workers without copying edge abstractions.*

- [x] **Structured HTTP Context:** Pass Method, Headers, Path, and Query via structured JSON/Proto to Stdin.
- [x] **Header Mutation:** Allow WASM to modify response headers.
- [x] **Status Code Control:** Allow WASM to set HTTP 4xx/5xx codes.
- [x] **Environment Variables:** Configuration per function.
- [x] **Per-route Binding:** Map specific `.wasm` files to specific Caddy routes.

## 🔵 Phase 2: Enterprise Observability & Limits (Completed v0.2.0)
*Gain trust by proving reliability and giving developers "X-Ray vision".*

- [x] **Structured Logging Interface:** Implement host.Log(level, json) so WASM logs appear correctly structured in Datadog/Loki, not just as raw stdout text.
- [x] **OpenTelemetry Tracing:** Inject traceparent context into WASM so calls can be traced from Frontend -> Caddy -> WASM -> Database in a single waterfall view.
- [x] **Prometheus Metrics:** Expose native metrics: gojinn_function_duration_seconds and gojinn_active_sandboxes.
- [x] **CPU Budgeting:** Strict metering to prevent infinite loops.
- [x] **Security Policy (SECURITY.md):** Define security boundaries explicitly.

## 🟣 Phase 3: High Performance & Trust (Completed v0.3.0)
*Turn Gojinn into the fastest serverless runtime and prove it to legacy teams.*

### ⚙️ Core Engine (Performance)
- [x] **VM Pooling (Worker Pool):** Implement a pool of "warm" `wazero` instances. Instead of destroying the VM after use, reset its memory and put it back in the pool.
    - *Goal:* Drop latency from ~1ms to **micro-seconds**.
- [x] **JIT Caching:** Ensure compiled modules are cached in memory globally (Singleton pattern), ensuring zero-compilation on hot paths.

### 📢 Adoption Strategy
- [x] **The "Strangler Fig" Examples:** Create a folder /examples/legacy-integration showing how to put Gojinn in front of Java/Spring. *(Done in v0.2.1)*
- [x] **Reproducible Benchmarks:** Public repo (gojinn-benchmarks) comparing Gojinn (Pooled) vs Docker vs Native Go.
- [x] **"Dogfooding" Case Study:** Migrated `paulo.app.br` backend to Gojinn, reducing latency to <100ms and removing idle containers.

## 🟠 Phase 4: Host Capabilities & Developer Experience (Completed v0.4.0)
*Solve the "Database Latency" problem and embrace modern web patterns.*

- [x] **Secure Debug Mode:** Implement `X-Gojinn-Debug` header logic. If a secret matches, capture Stderr and inject it into Response Headers (`X-Logs`) for browser-based debugging.
- [x] **HTMX & SSR Patterns:** Create official examples and helpers for returning HTML fragments from WASM (Go Templates/Rust) instead of JSON. Validate the "Hypermedia-Driven" architecture.
- [x] **Host-Managed DB Pool:** Expose Host DB connections (Postgres/MySQL) to WASM via Host Functions to prevent "Too Many Connections" errors.
- [x] **Gojinn KV:** In-memory key-value store exposed to WASM via Host Functions.
- [x] **SQLite Sidecar:** Allow WASM to execute SQL queries on a local SQLite file (Zero-network DB).

## 🔴 Phase 5: Production & Operations (Completed v0.5.0)
*Professionalize the deployment. Treat Gojinn as critical infrastructure.*

- [x] **Optimized Build Pipeline:** Makefile/Script to generate lean binaries for Linux (AMD64/ARM64).
- [x] **Systemd Integration:** Create a `.service` file for auto-restart and log management.
- [x] **Log Rotation:** Ensure Caddy logs don't fill the disk.
- [x] **Git-Push-to-Deploy:** Simple hook or script to update the server based on Git tags.

## 🟤 Phase 6: Polyglot Support (Current Priority)
*Expand the ecosystem. Don't force users to learn Rust.*

- [ ] **JavaScript/TypeScript Adapter:** Integrate Javy (QuickJS).
- [ ] **Python Adapter:** Support for RustPython packed as WASM.
- [ ] **Language-Agnostic CLI:** Gojinn build command.

## 🟤 Phase 7: Async & Event-Driven
*Compete with AWS Lambda's event ecosystem.*

- [ ] **Cron Triggers:** Native support for scheduling functions (@every 5m)
- [ ] **Worker Pools (Async Mode):** "Fire-and-forget" support using internal Go channels.
- [ ] **Queue Binding:** Internal buffer for async processing.

## 🔴 Phase 8: Distributed & Edge Scale
*Scale horizontally across regions effortlessly.*

- [ ] **Cluster Storage Support:** Document and support using caddy-storage-redis or consul to synchronize TLS certificates across multiple Gojinn nodes (Edge capability).
- [ ] **OCI Registry Support:** Deploy functions by referencing container images (image: ghcr.io/user/func:v1) directly in Caddyfile (GitOps standard).
- [ ] **Universal Runtime:** Same binary runs on Bare Metal, VPS, and Edge.

## 🚀 Future Horizons (Version 2.0)
*Cutting-edge features for the next generation.*

- [ ] **Edge AI Inference:** Host-level bindings for LLM inference (llama.cpp), allowing WASM to call AI without overhead.
- [ ] **Time-Travel Debugging:** Save WASM execution inputs to replay failed requests locally

---

### ❌ Explicit Non-Goals
To keep the project focused, we will **NOT** build:
* Proprietary APIs or Vendor-specific abstractions.
* Forced control planes or billing layers.
* "Magic" networking or hidden overlays.