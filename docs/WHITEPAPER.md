# Gojinn: The Sovereign Serverless Cloud
**Official Whitepaper & v1.0 LTS Stability Guarantee**

## 1. The v1.0 LTS Stability Guarantee
With the release of Gojinn v1.0, we enter a **Stability Freeze**. This ensures enterprise adopters can build upon the platform with absolute confidence.

* **API Freeze:** The Caddyfile directives (`gojinn { ... }`), the WASM SDK Host Functions (`host_kv_set`, `host_db_query`, etc.), and the REST/MCP endpoints are strictly frozen. No breaking changes will occur in the `1.x.x` lifecycle.
* **Backward Compatibility Policy:** We adhere to Semantic Versioning (SemVer). Minor updates (`1.x`) will only introduce non-breaking enhancements or new host functions. Patch updates (`1.x.y`) are strictly reserved for security fixes and performance hardening.
* **Long Term Support (LTS):** Gojinn v1.0 is an LTS release. It will receive critical security and bug-fix backports for **18 months** from the release date.
* **Production Hardening Only:** No feature creep. The focus of the `1.x` branch is exclusively on maximizing throughput, minimizing latency, and ensuring absolute stability.

## 2. Architecture & The CAP Theorem Model
Gojinn is a distributed, multi-tenant Serverless execution engine built over Caddy (Edge), Wazero (Compute), and NATS JetStream (State).

In the context of the **CAP Theorem** (Consistency, Availability, Partition Tolerance), Gojinn allows the operator to choose their stance via the `cluster_replicas` directive:

* **CP Mode (Consistency & Partition Tolerance) [Recommended]:** When `cluster_replicas` is set to 3 or 5, Gojinn's embedded JetStream enforces a Raft consensus algorithm. In the event of a network partition, the minority side will halt operations to prevent split-brain scenarios, prioritizing absolute data consistency (Audit Logs, KV State) over availability.
* **AP Mode (Availability & Partition Tolerance) [Edge Mode]:** When `cluster_replicas` is set to 1, nodes operate autonomously. If the cluster partitions, nodes continue to serve their local state and queue jobs. This prioritizes availability but sacrifices global strict consistency until the network heals.

## 3. Threat Model Summary
Gojinn employs a "Hard Isolation" architecture.
* **Execution Isolation:** WASM modules run in default-deny sandboxes with zero host OS access.
* **Resource Isolation:** A strict `cappedWriter` and CPU `context.WithTimeout` enforce I/O and memory quotas, mitigating "Silent Memory Exhaustion" and OOM crashes.
* **Tenant Isolation:** Dynamic NATS stream provisioning ensures Tenant A (`WORKER_A`) can never cross into Tenant B's message queue (`WORKER_B`).
* *(For full details, see `THREAT_MODEL.md`)*.

## 4. Failure Scenarios & Resilience
Gojinn is engineered to survive catastrophic failures.

| Scenario | System Response | Outcome |
| :--- | :--- | :--- |
| **Worker Infinite Loop** | CPU Context timeout kills the WASM execution. | Tenant job fails; Host server survives. |
| **Malicious Memory Leak** | Hard limit (e.g., 128MB) forces WASM engine to panic. | OOM bypass prevented; Tenant job crashes safely. |
| **Node Crash (Power Loss)** | Unacknowledged JetStream messages remain in the queue. | Upon reboot, messages are redelivered to available workers. Zero data loss. |
| **Host DB Disconnect** | LibSQL/SQLite embedded replica continues serving reads. | Reads succeed (stale); Writes queue or fail depending on sync policy. |
| **Audit Log Tampering** | Each job output is signed via HMAC-SHA256 (`StoreCipherKey`). | Cryptographic signature mismatch reveals tampering instantly. |

## 5. Public Benchmarks (Methodology)
*(Note: Live benchmark numbers depend on hardware. The following outlines the official testing methodology).*
* **Cold Start Latency:** Measured from HTTP request arrival to WASM `main()` execution. (Target: < 2ms).
* **Warm Execution Throughput:** Measured using `hey` or `wrk` against a cached memory instance.
* **Multi-Tenant Spin-up:** Measured by hitting the server with 100 unique IP addresses simultaneously to track dynamic NATS provisioning overhead.