# 🧞 Gojinn

> **High-Performance Serverless Runtime for Caddy**

[![Go Reference](https://pkg.go.dev/badge/github.com/caddyserver/caddy/v2.svg)](https://pkg.go.dev/github.com/pauloappbr/gojinn)
[![Build Status](https://img.shields.io/badge/build-passing-brightgreen)]()
[![Wasm Engine](https://img.shields.io/badge/engine-wazero-purple)](https://wazero.io)
[![Version](https://img.shields.io/badge/version-v0.4.1-blue)]()

**Gojinn** is an *in-process serverless* runtime for the [Caddy](https://caddyserver.com) web server.

It allows you to execute **Go**, **Rust**, **Zig**, and **C++** code (compiled to WebAssembly) directly in the HTTP request flow with **Host-Managed Capabilities** (Database, Key-Value Store, Logs).

With the release of **v0.4.0**, Gojinn introduces the **Official SDK** and **Sidecar Database Support** (Postgres/MySQL/SQLite), enabling true stateful serverless applications.

---

## 🚀 Why Gojinn?

Traditional serverless introduces network latency. Gojinn brings computation closer to the data.

| Feature | Description |
| :--- | :--- |
| **⚡ Microsecond Latency** | JIT Caching & Buffer Pooling ensure execution in **< 1ms**. |
| **🗄️ Zero-Latency DB** | **New:** Host-managed connection pools for Postgres, MySQL, and embedded SQLite. |
| **🧠 In-Memory KV** | **New:** Ultra-fast Key-Value store shared across requests (great for counters/cache). |
| **🏗️ Zero Infra** | No Docker daemon, no Kubernetes sidecars. It's just a Caddy plugin. |
| **👁️ Observable** | Native support for **Prometheus Metrics**, **Tracing**, and **Secure Remote Debugging**. |
| **🛡️ Secure** | Each request runs in a strict Sandbox via [Wazero](https://wazero.io). |

---

## Use Cases

* **Stateful Edge Logic:** Rate limiters, counters, and caching layers using Gojinn KV.
* **Database APIs:** Build REST/GraphQL endpoints querying SQL databases directly.
* **Hypermedia/HTMX:** Server-Side Rendering of HTML fragments with zero overhead.
* **Legacy Migration:** "Strangler Fig" pattern replacing monolith endpoints one by one.

---

## Documentation

### Getting Started
* [⚡ Quick Start Guide](./getting-started/quickstart.md)
* [📦 Installation](./getting-started/installation.md)

### Guides
* [🛠️ Golang SDK & Examples](./guides/golang.md)
* [🚢 Deployment & Operations](./guides/deployment.md) **(New)**
* [🐞 Debugging & Observability](./guides/debugging.md)

### Deep Dive
* [🏗 Architecture](./concepts/architecture.md)
* [🔌 JSON Contract](./concepts/contract.md)
* [📊 Benchmarks](./benchmark.md)