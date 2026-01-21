# ⚙️ Caddyfile Directive: `gojinn`

The `gojinn` directive configures the WebAssembly runtime for a specific route. It executes the WASM binary, passing the request context via Stdin and returning the response via Stdout.

## Syntax

```caddy
gojinn <path_to_wasm_file> {
    timeout      <duration>
    memory_limit <size>
    pool_size    <int>
    env          <key> <value>
    args         <arg1> <arg2>...
}
```

## ⚠️ Important: Handler Ordering

Because Gojinn is a plugin, Caddy does not know its default execution order relative to standard directives (like `file_server` or `reverse_proxy`).

To avoid the error "directive 'gojinn' is not an ordered HTTP handler", you must either:

### Define the order globally (Recommended)

```caddy
{
    order gojinn last
}
```

### Or wrap it in a route block

```caddy
route {
    gojinn ./main.wasm
}
```

## ### `<path_to_wasm_file>`

**Type:** `string`  
**Required:** Yes

The path to the `.wasm` or `.wat` binary file. Can be a relative path (to the folder where Caddy was executed) or absolute.

## Sub-directives

### `timeout`

Sets the maximum execution time allowed for the function before the VM is forcibly terminated.

- **Default:** `60s` (1 minute)
- **Syntax:** `timeout <duration>`
- **Examples:** `100ms`, `2s`, `1m`

⚠️ **Important:** If the function exceeds this time, Gojinn will interrupt execution immediately and return a 504 Gateway Timeout error (or 500 depending on the stage). This protects your server against infinite loops (`while true`) and CPU exhaustion.

### `memory_limit`

Sets the hard limit on RAM memory that the Sandbox can allocate.

- **Default:** Unlimited (limited only by host RAM)
- **Syntax:** `memory_limit <size>`
- **Examples:** `128MB`, `512KB`, `1GB`

💡 **Tip for Go (Golang):** Binaries compiled with standard Go (not TinyGo) have a runtime overhead. We recommend setting at least 64MB or 128MB to avoid Out of Memory (OOM) errors during initialization.

### `pool_size`

Controls the number of pre-warmed WebAssembly workers (VMs) kept in memory for this specific route.

- **Default:** Auto-scaled (NumCPU × 4, minimum 50 workers)
- **Syntax:** `pool_size <int>`
- **Examples:** `100`, `1`

🚀 **Performance vs RAM:** Increasing this value improves concurrent throughput but consumes more RAM (~2-10MB per worker, depending on the guest language). Workers are provisioned in parallel during Caddy startup to ensure zero cold starts.

### `env`

Injects environment variables into the WASM process.

- **Syntax:** `env <KEY> <VALUE>`
- **Placeholder Support:** Yes. You can inject secrets from the host using `{env.VAR_NAME}`

### `args`

Passes command-line arguments to the WASM binary (accessible via `os.Args` in the guest).

- **Syntax:** `args <arg1> <arg2> ...`

## 📝 Configuration Examples

### Minimal Configuration

```caddy
{
    order gojinn last
}

:8080 {
    handle /api/simple {
        gojinn ./functions/simple.wasm
    }
}
```

### Production Configuration (Robust)

```caddy
{
    order gojinn last
    admin :2019 # Required for Prometheus Metrics
}

:8080 {
    handle /api/contact {
        # Set header before passing to Gojinn
        header Content-Type application/json

        gojinn ./functions/contact.wasm {
            # Kills slow processes (CPU Budgeting)
            timeout 2s 
            
            # Prevents memory leaks
            memory_limit 128MB 

            # Optimizes for high traffic (Trade RAM for CPU speed)
            pool_size 100
            
            # Safely injects host environment credentials
            env DB_HOST "10.0.0.5"
            env API_KEY {env.SECRET_API_KEY}
            
            # Enables verbose logging in function logs
            args --debug --json-logs
        }
    }
}
```