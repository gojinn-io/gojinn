from sdk.python.shim import run
import sys

def handler(req):
    # Logs go to Caddy (stderr)
    sys.stderr.write(f"Received a Python request: {req['method']}\n")
    
    user_agent = req['headers'].get('User-Agent', ['Unknown'])[0]
    
    return {
        "status": 200,
        "headers": {
            "X-Gojinn-Lang": f"Python {sys.version.split()[0]}",
            "X-Snake-Power": "True"
        },
        "body": {
            "message": "Hello from Python via WebAssembly!",
            "received_headers": req['headers'],
            "your_agent": user_agent
        }
    }

if __name__ == "__main__":
    run(handler)
