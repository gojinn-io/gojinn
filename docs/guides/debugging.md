# 🔍 Observability & Debugging

Developing for WebAssembly *in-process* can be challenging because you don't have a traditional debugger attached to the server. Gojinn Phase 2 introduces "Enterprise Observability" features to help you see inside the black box.

---

## 📊 Metrics (Prometheus)

Gojinn automatically exposes native Prometheus metrics via Caddy's admin endpoint. This is the first place you should look to understand performance and health.

**Endpoint:** `http://localhost:2019/metrics` (Default)

### Key Metrics

| Metric Name | Type | Description |
| :--- | :--- | :--- |
| `gojinn_function_duration_seconds` | Histogram | Tracks how long your WASM function takes to run. Useful for spotting **Cold Starts** or performance regressions. Labeled by `path` and `status`. |
| `gojinn_active_sandboxes` | Gauge | Shows how many WASM VMs are currently running. If this number keeps growing but never drops, you might have a **Concurrency Leak** (requests getting stuck). |

**How to check via CLI:**

```bash
curl -s http://localhost:2019/metrics | grep gojinn
```

---

## 🆔 Distributed Tracing

Every request sent to your WASM function includes a `trace_id` field in the JSON payload.

- **Caddy (Host)**: Generates or propagates the Traceparent header.
- **Gojinn (Plugin)**: Injects this ID into the Input JSON.
- **WASM (Guest)**: Should use this ID in its logs.

**Debugging Tip:** Always print the `trace_id` in your error logs. This allows you to correlate a specific user error in the Frontend directly to the Caddy log entry and the WASM execution failure.

---

## ❌ Common Errors

### Error: 504 Gateway Timeout

**Cause:** Your function took longer to execute than the configured timeout.

**Reason A:** Infinite loop in your code (`for {}`).

**Reason B:** Heavy computation (e.g., recursive Fibonacci) on slow hardware.

**Solution:**

- Optimize your algorithm.
- Increase the `timeout` directive in your Caddyfile (e.g., `timeout 5s`).

### Error: 502 Bad Gateway

This means Gojinn couldn't get a valid JSON response from your WASM code.

**Common Causes:**

#### Dirty Stdout (The #1 Mistake)

- **Cause:** You used `fmt.Println("debug info")` to debug.
- **Why it breaks:** Gojinn expects pure JSON from Stdout. Any text before or after the JSON makes it invalid.
- **Fix:** Move all logs to Stderr.

#### Panic / Crash

- **Cause:** Your Go/Rust code exited with a non-zero code or panicked.
- **Fix:** Check Caddy logs. Gojinn captures the panic output and prints it there.

### Error: OOM (Out of Memory)

- **Symptom:** Logs showing `sys_mmap failed` or `failed to instantiate module`.
- **Cause:** The runtime needed more memory than the `memory_limit` allowed.
- **Fix:** Increase `memory_limit` in the Caddyfile. Standard Go binaries often need at least 64MB to start due to the Garbage Collector overhead.

---

## 🛠️ Recommended Debugging Flow

When something goes wrong, follow this ritual:

### 1. The "Two Terminals" Setup

- **Terminal 1:** Run `caddy run` (to see structured logs and WASM Stderr output).
- **Terminal 2:** Run your curl commands and build scripts.

### 2. Check the Metrics

Before diving into code, check if the server is healthy.

```bash
curl localhost:2019/metrics | grep gojinn_active_sandboxes
```

If it's > 0 when idle, your functions are hanging!

### 3. Test in Isolation

Don't test via browser immediately. Use curl to see the raw HTTP headers and status codes.

```bash
curl -v -X POST http://localhost:8080/api/function \
  -H "Content-Type: application/json" \
  -d '{"test": true}'
```