import sys
import json
import io

# Ensure STDOUT uses utf-8 and does not partially buffer lines
sys.stdout = io.TextIOWrapper(sys.stdout.buffer, encoding='utf-8')
sys.stdin = io.TextIOWrapper(sys.stdin.buffer, encoding='utf-8')

def send_response(status, body, headers=None):
    if headers is None:
        headers = {}
    
    # Add default header
    headers["Content-Type"] = headers.get("Content-Type", "application/json")

    response = {
        "status": status,
        "headers": headers,
        "body": json.dumps(body) if not isinstance(body, str) else body
    }
    
    # Write everything at once to avoid partial flush
    sys.stdout.write(json.dumps(response))
    sys.stdout.flush()

def run(handler):
    try:
        # 1. Read Input (STDIN)
        input_str = sys.stdin.read()
        
        if not input_str:
            req = {"body": "", "headers": {}, "method": "GET"}
        else:
            try:
                req = json.loads(input_str)
            except json.JSONDecodeError:
                req = {"body": input_str, "headers": {}, "method": "UNKNOWN"}

        # 2. Normalize Headers (to a simple dict if necessary)
        # Gojinn sends map[string][]string, but to simplify in Python we keep it raw
        # and let the user handle it, or simplify it here. We'll deliver it raw.
        
        # 3. Execute Handler
        result = handler(req)
        
        # 4. Send Response
        send_response(
            result.get("status", 200),
            result.get("body", ""),
            result.get("headers", {})
        )

    except Exception as e:
        # 5. Global Error Handling (stderr goes to Caddy logs)
        # Print to stderr for debugging
        sys.stderr.write(f"Python Runtime Error: {str(e)}\n")
        
        # Return JSON error to the client
        send_response(500, {
            "error": "Python Runtime Error",
            "message": str(e)
        })
