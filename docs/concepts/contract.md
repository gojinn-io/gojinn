# 📜 Function Contract

Gojinn is designed to be simple and language-agnostic. It communicates with your WebAssembly (WASM) binary exclusively through **Standard Input (Stdin)** and **Standard Output (Stdout)**.

There is no "magic" or mandatory proprietary libraries. Just serialized JSON.

---

## 📥 Input (Stdin)

When an HTTP request arrives at Caddy on a route configured for Gojinn, the plugin serializes the request data and writes it to the `Stdin` of your WASM process.

### Request Structure

```json
{
  "method": "POST",
  "uri": "/api/contact?id=123",
  "headers": {
    "Content-Type": ["application/json"],
    "User-Agent": ["curl/7.64.1"],
    "Traceparent": ["00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"]
  },
  "body": "{\"user\": \"john\", \"message\": \"hello\"}",
  "trace_id": "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"
}
```

#### Fields

- **method** (string): HTTP method (`GET`, `POST`, `PUT`, `DELETE`, etc.)
- **uri** (string): Request URI with query parameters
- **headers** (map): Map of HTTP headers, where each value is an array of strings
- **body** (string): Raw content of the request body
- **trace_id** (string): Distributed tracing identifier (W3C Trace Context or X-Request-ID). Use this to correlate logs.

> ⚠️ **Attention to Body**: The `body` field is always a string. If the client sent JSON, that JSON will be escaped (serialized) within the string. Your code must unmarshal this string internally to access the payload data.

#### Example in Go

```go
type GojinnRequest struct {
    Method  string              `json:"method"`
    URI     string              `json:"uri"`
    Headers map[string][]string `json:"headers"`
    Body    string              `json:"body"`
    TraceID string              `json:"trace_id"`
}
```

## 📤 Output (Stdout)

To respond to the request, your program must write a single JSON object to Stdout. Gojinn will read this output and transform it into an HTTP response for the client.

### Response Structure

```json
{
  "status": 200,
  "headers": {
    "Content-Type": ["application/json"],
    "X-Powered-By": ["Gojinn"]
  },
  "body": "{\"success\": true, \"id\": 99}"
}
```

#### Fields

- **status** (int): HTTP status code (e.g., 200, 404, 500)
- **headers** (map): Map of HTTP headers, where each value is an array of strings
- **body** (string): Raw content of the response body

#### Example in Go

```go
type GojinnResponse struct {
    Status  int                 `json:"status"`
    Headers map[string][]string `json:"headers"`
    Body    string              `json:"body"`
}
```

## ⚠️ Strict Rules

### Headers

Headers must be a map where the key is the header name and the value is an **array of strings**.

```text
✅ Correct:   "Content-Type": ["application/json"]
❌ Incorrect: "Content-Type": "application/json"
```

### Body

The response body must be a string. If you want to return JSON to the client, serialize your response object to a string before placing it here.

### Clean Stdout

**Do not print debug logs to Stdout.** Because Gojinn expects valid JSON from Stdout, printing plain text (like `fmt.Println("debug")`) will break the JSON parsing and cause a **502 Bad Gateway**.

- **Correct:** Write logs to `Stderr`.

- **Correct:** Use the provided Host Function `host_log` (see Logging documentation).

- **Incorrect:** `fmt.Println("I am here")` -> Breaks the protocol.

## 🐛 Debugging and Errors

If your program does not follow this strict contract, Caddy will return a **502 Bad Gateway** error. Examples of violations:

- Printing debug text to Stdout before the JSON.
- Returning a string instead of an integer for `status`
- Exiting the program with an error code
- Headers with values that are not arrays of strings

Check the Caddy logs (Stderr) to see the detailed deserialization error message.