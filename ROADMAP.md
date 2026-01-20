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

## 🔵 Phase 2: Enterprise Observability & Limits (Current Priority)
*Gain trust by proving reliability and giving developers "X-Ray vision".*

- [x] **Structured Logging Interface:** Implement host.Log(level, json) so WASM logs appear correctly structured in Datadog/Loki, not just as raw stdout text.
- [x] **OpenTelemetry Tracing:** Inject traceparent context into WASM so calls can be traced from Frontend -> Caddy -> WASM -> Database in a single waterfall view.
- [x] **Prometheus Metrics:** Expose native metrics: gojinn_function_duration_seconds and gojinn_active_sandboxes.
- [x] **CPU Budgeting:** Strict metering to prevent infinite loops.
- [x] **Security Policy (SECURITY.md):** Define security boundaries explicitly.

## 🟣 Phase 3: Trust & Adoption Strategy
*Remove barriers for adoption by legacy teams and CTOs.*

- [ ] **The "Strangler Fig" Examples:** Create a folder /examples/legacy-integration showing how to put Gojinn in front of Java/Spring and PHP/WordPress to replace specific slow endpoints.
- [ ] **Reproducible Benchmarks:** Public repo (gojinn-benchmarks) comparing Gojinn vs Docker/Lambda.
- [ ] **"Dogfooding" Case Study:** Blog post on migrating a production app.

## 🔴 Phase 4: Polyglot Support
*Expand the ecosystem. Don't force users to learn Rust.*

- [ ] **JavaScript/TypeScript Adapter:** Integrate Javy (QuickJS) to compile JS to WASM.
- [ ] **Python Adapter:** Support for RustPython packed as WASM.
- [ ] **Language-Agnostic CLI:** Gojinn build command to auto-detect language.

## 🟠 Phase 5: Stateful Serverless (Performance Critical)
*Solve the "Database Latency" problem.*

- [ ] **Smart Connection Pooling:** Host-managed database connections (Postgres/MySQL) shared with WASM to prevent "Too Many Connections" errors. (Crucial for scalability).
- [ ] **Gojinn KV:** In-memory key-value store exposed to WASM via Host Functions.
- [ ] **SQLite Sidecar:** Allow WASM to execute SQL queries on a local SQLite file (Zero-network DB).

## 🟤 Phase 6: Async & Event-Driven
*Compete with AWS Lambda's event ecosystem.*

- [ ] **Cron Triggers:** Native support for scheduling functions (@every 5m)
- [ ] **Worker Pools (Async Mode):** "Fire-and-forget" support using internal Go channels.
- [ ] **Queue Binding:** Internal buffer for async processing.

## 🔴 Phase 7: Distributed & Edge Scale
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