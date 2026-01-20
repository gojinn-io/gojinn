# 🏗️ Strangler Fig Pattern Example

This example demonstrates how to use **Gojinn** to incrementally migrate a legacy monolith (PHP, Java, Python) without rewriting the entire system at once.

## 🚀 How to Run

You will need **3 terminal windows**.

### 1. Start the Legacy Server (Port 9090)

```bash
go run legacy-server/main.go
```

### 2. Start Caddy with Gojinn (Port 8080)

Run from the project root:

```bash
go run ../../cmd/caddy/main.go run --config Caddyfile
```

### 3. Test as a Client

#### Scenario A: Accessing the Legacy System

This request falls through to the old server (simulated 2s delay).

```bash
curl http://localhost:8080/home
```

#### Scenario B: Accessing the Migrated Endpoint

This request is intercepted by Gojinn (WASM) and returns instantly (<1ms).

```bash
curl -v http://localhost:8080/api/calc
```