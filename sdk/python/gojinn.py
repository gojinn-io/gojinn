import sys
import json
import io

if sys.stdin.encoding != 'utf-8':
    sys.stdin = io.TextIOWrapper(sys.stdin.buffer, encoding='utf-8')
if sys.stdout.encoding != 'utf-8':
    sys.stdout = io.TextIOWrapper(sys.stdout.buffer, encoding='utf-8')

class Request:
    def __init__(self, raw_dict):
        self.body = raw_dict.get("body", "")
        self.headers = raw_dict.get("headers", {})
        self.method = raw_dict.get("method", "POST")
        self.uri = raw_dict.get("uri", "/")

    def json(self):
        """Helper para parsear o body como JSON automaticamente"""
        try:
            if isinstance(self.body, dict):
                return self.body
            return json.loads(self.body)
        except:
            return {}

class Response:
    def __init__(self, body, status=200, headers=None):
        self.body = body
        self.status = status
        self.headers = headers if headers else {}
        if "Content-Type" not in self.headers:
            self.headers["Content-Type"] = "application/json"
        self.headers["X-Runtime"] = "Gojinn-Python"

    def to_dict(self):
        final_body = self.body
        if not isinstance(self.body, str):
            final_body = json.dumps(self.body)

        return {
            "status": self.status,
            "headers": self.headers,
            "body": final_body
        }

class Logger:
    def info(self, msg):
        sys.stderr.write(f"[INFO] {msg}\n")
    
    def error(self, msg):
        sys.stderr.write(f"[ERROR] {msg}\n")

    def warn(self, msg):
        sys.stderr.write(f"[WARN] {msg}\n")

logger = Logger()

def handle(handler_func):
    try:
        input_data = sys.stdin.read()
        
        req_payload = {}
        if input_data:
            try:
                req_payload = json.loads(input_data)
            except json.JSONDecodeError:
                req_payload = {"body": input_data}
        
        req = Request(req_payload)
        
        result = handler_func(req)
        
        final_response = None
        if isinstance(result, Response):
            final_response = result
        elif isinstance(result, dict) or isinstance(result, list) or isinstance(result, str):
            final_response = Response(result)
        else:
            final_response = Response("")
            
        sys.stdout.write(json.dumps(final_response.to_dict()))
        
    except Exception as e:
        logger.error(f"Runtime Panic: {str(e)}")
        err_resp = {
            "status": 500,
            "headers": {"Content-Type": "application/json"},
            "body": json.dumps({"error": str(e), "type": "PythonRuntimeError"})
        }
        sys.stdout.write(json.dumps(err_resp))