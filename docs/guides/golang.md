# 🐹 Creating Functions in Go

Go is the "native" language of the Cloud Native ecosystem and an excellent choice for Gojinn. Due to the nature of WebAssembly (WASI), writing functions for Gojinn is very similar to writing command-line tools (CLI).

---

## 📋 The Pattern (Boilerplate)

To avoid deserialization errors, we recommend copying and maintaining these base structures in your functions.

```go
package main

import (
    "encoding/json"
    "fmt"
    "io"
    "os"
)

// --- 1. Gojinn Structures (The Contract) ---

// Input Wrapper (Request)
type GojinnRequest struct {
    Method  string              `json:"method"`
    URI     string              `json:"uri"`
    Headers map[string][]string `json:"headers"`
    Body    string              `json:"body"`     // User payload comes here as a string
    TraceID string              `json:"trace_id"` // Distributed Tracing ID from Caddy
}

// Output Wrapper (Response)
type GojinnResponse struct {
    Status  int                 `json:"status"`
    Headers map[string][]string `json:"headers"`
    Body    string              `json:"body"`
}

// --- 2. Your Business Logic ---

type MyUserPayload struct {
    Name  string `json:"name"`
    Email string `json:"email"`
}

func main() {
    // We delegate logic to 'run' to make it testable
    if err := run(os.Stdin, os.Stdout); err != nil {
        fmt.Fprintf(os.Stderr, "Fatal error: %v\n", err)
        os.Exit(1)
    }
}

// run contains the core logic. It takes interfaces, so it can be tested easily.
func run(in io.Reader, out io.Writer) error {
    // A. Decode Input
    var req GojinnRequest
    if err := json.NewDecoder(in).Decode(&req); err != nil {
        return replyError(out, 400, "Invalid JSON input")
    }

    // B. Logs & Tracing (Use Stderr)
    // Always include TraceID in logs to correlate with Caddy logs
    fmt.Fprintf(os.Stderr, "[%s] Processing request for URI: %s\n", req.TraceID, req.URI)

    // C. Process your Payload
    var payload MyUserPayload
    if req.Body != "" {
        if err := json.Unmarshal([]byte(req.Body), &payload); err != nil {
             return replyError(out, 400, "Invalid user payload")
        }
    }

    // D. Respond
    responseData := map[string]string{
        "message": fmt.Sprintf("Hello, %s!", payload.Name),
        "trace":   req.TraceID,
    }
    responseJSON, _ := json.Marshal(responseData)

    return reply(out, 200, string(responseJSON))
}

// Helpers
func reply(out io.Writer, status int, body string) error {
    resp := GojinnResponse{
        Status: status,
        Headers: map[string][]string{
            "Content-Type": {"application/json"},
        },
        Body: body,
    }
    return json.NewEncoder(out).Encode(resp)
}

func replyError(out io.Writer, status int, msg string) error {
    errJSON := fmt.Sprintf(`{"error": "%s"}`, msg)
    return reply(out, status, errJSON)
}
```

## 🔧 Compilation

You have two main options for compiling your Go code to WASM.

### Option 1: Standard Compiler (Go Toolchain)

Full compatibility, fully supported by Gojinn. Recommended for most users.

```bash
# Since Go 1.21, use 'wasip1'
GOOS=wasip1 GOARCH=wasm go build -o function.wasm main.go
```
- **Pros:** 100% compatible with standard library.
- **Cons:** Binaries are larger (~2MB+).
- **Requirement:** Set `memory_limit` to at least 64MB in Caddyfile.

**Gojinn requirement**: Configure `memory_limit` of `64MB` or higher in the Caddyfile.

### Option 2: TinyGo (Performance)

Produces tiny binaries (~100KB - 500KB) and ultra-fast startup.

```bash
tinygo build -o function.wasm -target=wasi main.go
```

- **Pros:** Very low memory footprint (works with `memory_limit 10MB`).

- **Cons:** Does not support `encoding/json` reflection fully in some versions, and lags behind latest Go versions.

---

## 💡 Golden Tips

### Logging and Observability

- **Logs:** Always write logs to `os.Stderr`. Gojinn captures this and sends it to Caddy's structured logs.

- **Tracing:** Use the `trace_id` field from the request in your log messages. This allows you to trace a request from the Frontend -> Caddy -> WASM -> Database.

> ⚠️ **Critical:** Never use `fmt.Println` for logs. It writes to `Stdout`, which breaks the JSON response contract and causes **502 Bad Gateway** errors.

### Unit Testing

Because we use the `run(io.Reader, io.Writer)` pattern, you can easily test your function without running Caddy:

```bash
func TestRun(t *testing.T) {
    input := []byte(`{"method":"POST", "body": "..."}`)
    var output bytes.Buffer
    
    // Inject mock buffers
    err := run(bytes.NewReader(input), &output)
    
    // Assert output...
}
```