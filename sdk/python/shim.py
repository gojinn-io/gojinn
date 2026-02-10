import sys
import json
import io

sys.stdout = io.TextIOWrapper(sys.stdout.buffer, encoding='utf-8')
sys.stdin = io.TextIOWrapper(sys.stdin.buffer, encoding='utf-8')

def send_response(status, body, headers=None):
    if headers is None:
        headers = {}
    
    headers["Content-Type"] = headers.get("Content-Type", "application/json")

    response = {
        "status": status,
        "headers": headers,
        "body": json.dumps(body) if not isinstance(body, str) else body
    }
    
    sys.stdout.write(json.dumps(response))
    sys.stdout.flush()

def run(handler):
    try:
        input_str = sys.stdin.read()
        
        if not input_str:
            req = {"body": "", "headers": {}, "method": "GET"}
        else:
            try:
                req = json.loads(input_str)
            except json.JSONDecodeError:
                req = {"body": input_str, "headers": {}, "method": "UNKNOWN"}

        result = handler(req)
        
        send_response(
            result.get("status", 200),
            result.get("body", ""),
            result.get("headers", {})
        )

    except Exception as e:
        sys.stderr.write(f"Python Runtime Error: {str(e)}\n")
        
        send_response(500, {
            "error": "Python Runtime Error",
            "message": str(e)
        })
