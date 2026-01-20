# 🦀 Creating Functions in Rust

Rust and WebAssembly are a perfect combination. Rust offers memory safety and native performance without the overhead of a Garbage Collector, making it ideal for high-density functions on Gojinn.

---

## ✅ Prerequisites

You will need the WASI target installed in your toolchain:

```bash
rustup target add wasm32-wasi
```

### Suggested Dependencies (Cargo.toml)

To handle the JSON contract, you need serde.

```toml
[package]
name = "gojinn-function"
version = "0.1.0"
edition = "2021"

[dependencies]
serde = { version = "1.0", features = ["derive"] }
serde_json = "1.0"
```

## 📋 The Pattern (Boilerplate)

Copy this structure to handle the Gojinn JSON contract correctly, including the new distributed tracing capabilities.

```rust
use std::collections::HashMap;
use std::io::{self, Read, Write};
use serde::{Deserialize, Serialize};

// --- 1. Gojinn Contract ---

#[derive(Deserialize)]
struct GojinnRequest {
    method: String,
    uri: String,
    headers: HashMap<String, Vec<String>>,
    body: String,     // User payload comes here as a string
    trace_id: String, // Distributed Tracing ID from Caddy
}

#[derive(Serialize)]
struct GojinnResponse {
    status: u16,
    headers: HashMap<String, Vec<String>>,
    body: String,
}

// --- 2. Your Business Logic Data ---

#[derive(Deserialize)]
struct MyPayload {
    name: String,
}

fn main() -> io::Result<()> {
    // A. Read Stdin (Input)
    let mut buffer = String::new();
    io::stdin().read_to_string(&mut buffer)?;

    // B. Unwrap Gojinn Request
    let req: GojinnRequest = match serde_json::from_str(&buffer) {
        Ok(r) => r,
        Err(e) => return reply_error(400, &format!("Invalid JSON Input: {}", e)),
    };

    // C. Logging (Use Stderr + TraceID)
    // We use eprintln! to send logs to Caddy without breaking the JSON response
    eprintln!("[{}] Processing request for URI: {}", req.trace_id, req.uri);

    // D. Process Internal Payload
    let name = if req.body.is_empty() {
        "Stranger".to_string()
    } else {
        match serde_json::from_str::<MyPayload>(&req.body) {
            Ok(p) => p.name,
            Err(_) => "Stranger".to_string(),
        }
    };

    // E. Respond
    let response_body = format!(r#"{{"message": "Hello from Rust, {}!", "trace": "{}"}}"#, name, req.trace_id);
    
    reply(200, response_body)
}

// Helper to write the JSON response to Stdout
fn reply(status: u16, body: String) -> io::Result<()> {
    let mut headers = HashMap::new();
    headers.insert("Content-Type".to_string(), vec!["application/json".to_string()]);

    let resp = GojinnResponse {
        status,
        headers,
        body,
    };

    let output = serde_json::to_string(&resp)?;
    io::stdout().write_all(output.as_bytes())?;
    Ok(())
}

fn reply_error(status: u16, msg: &str) -> io::Result<()> {
    let body = format!(r#"{{"error": "{}"}}"#, msg);
    reply(status, body)
}
```

## 🔧 Compilation

To compile your Rust function to WASI:

```bash
cargo build --target wasm32-wasi --release
```

The binary will be located at: `target/wasm32-wasi/release/your_function.wasm`

## 🚀 Why Rust on Gojinn?

- **Memory Stability**: Unlike Go, Rust rarely suffers from initial OOM (Out of Memory) since it doesn't verify a heavy runtime or Garbage Collector.

- **Predictable Performance**: No "Stop-the-world" pauses.

- **Binary Size**: After stripping, Rust binaries can be extremely small (< 2MB), which helps with cold start times.

### Pro-Tip: Reducing Binary Size

To make your function even faster, strip debug symbols in your Cargo.toml:

```toml
[profile.release]
lto = true
opt-level = 'z' # Optimize for size
strip = true
```