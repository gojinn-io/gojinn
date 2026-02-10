# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| v0.1.x  | :white_check_mark: |
| < 0.1.0 | :x:                |

## Isolation Model (The "Lamp")

Gojinn uses a strict **WebAssembly Sandbox** model based on [Wazero](https://wazero.io/).

1.  **Memory Isolation**: The guest code (WASM) cannot access the host (Caddy) memory. It operates within a linear memory buffer defined by the `memory_limit` directive.
2.  **No Filesystem Access**: By default, the guest module has **zero** access to the host filesystem. It can only read `stdin` (Request) and write to `stdout` (Response).
3.  **No Network Access**: The guest module cannot open sockets. All network IO must be proxied via the Host (Caddy).
4.  **CPU Budgeting**: Execution is strictly bounded by the `timeout` directive. If the clock runs out, the sandbox is instantly killed, preventing infinite loops (Halting Problem).

## Reporting a Vulnerability

If you discover a sandbox escape or a memory leak vulnerability, please do NOT open a public issue.

Email: contato@paulo.app.br
We will respond within 48 hours.