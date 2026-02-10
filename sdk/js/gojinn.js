const IO = {
  readAll: function () {
    const chunkSize = 64 * 1024;
    const chunks = [];
    let totalBytes = 0;

    while (true) {
      const buffer = new Uint8Array(chunkSize);
      let bytesRead = 0;
      try {
        bytesRead = Javy.IO.readSync(0, buffer);
      } catch (e) {
        break;
      }

      if (bytesRead === 0) break;

      chunks.push(buffer.subarray(0, bytesRead));
      totalBytes += bytesRead;
    }

    const finalBuffer = new Uint8Array(totalBytes);
    let offset = 0;
    for (const chunk of chunks) {
      finalBuffer.set(chunk, offset);
      offset += chunk.length;
    }

    const decoder = new TextDecoder();
    return decoder.decode(finalBuffer);
  },

  write: function (data) {
    const encoder = new TextEncoder();
    const buffer = encoder.encode(data);
    Javy.IO.writeSync(1, buffer);
  },

  log: function (message) {
    const encoder = new TextEncoder();
    const buffer = encoder.encode(message + "\n");
    Javy.IO.writeSync(2, buffer);
  }
};

class Request {
  constructor(raw) {
    this.body = raw.body || "";
    this.headers = raw.headers || {};
    this.method = raw.method || "POST";
  }

  json() {
    try {
      return typeof this.body === 'string' ? JSON.parse(this.body) : this.body;
    } catch (e) {
      return {};
    }
  }
}

class Response {
  constructor(body = "", status = 200, headers = {}) {
    this.body = body;
    this.status = status;
    this.headers = {
      "Content-Type": "application/json",
      "X-Runtime": "Gojinn-JS",
      ...headers
    };
  }

  toString() {
    return JSON.stringify({
      status: this.status,
      headers: this.headers,
      body: typeof this.body === 'object' ? JSON.stringify(this.body) : String(this.body)
    });
  }
}

export const logger = {
  info: (msg) => IO.log(`[INFO] ${msg}`),
  error: (msg) => IO.log(`[ERROR] ${msg}`),
  warn: (msg) => IO.log(`[WARN] ${msg}`)
};

export const kv = {
  set: (key, value) => logger.warn(`KV.set('${key}') not supported in JS adapter yet`),
  get: (key) => {
    logger.warn(`KV.get('${key}') not supported in JS adapter yet`);
    return null;
  }
};

export function handle(userHandler) {
  try {
    const rawInputStr = IO.readAll();
    let rawInput = {};
    
    if (rawInputStr) {
      try {
        rawInput = JSON.parse(rawInputStr);
      } catch (e) {
        rawInput = { body: rawInputStr };
      }
    }

    const req = new Request(rawInput);

    const result = userHandler(req);

    let finalResp;
    if (result instanceof Response) {
      finalResp = result;
    } else {
      finalResp = new Response(result);
    }

    IO.write(finalResp.toString());

  } catch (error) {
    const errResp = new Response(
      JSON.stringify({ error: error.message, stack: error.stack }), 
      500
    );
    IO.write(errResp.toString());
    logger.error(`Runtime Panic: ${error.message}`);
  }
}

export { Request, Response };