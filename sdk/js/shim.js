// sdk/js/shim.js
// Polyglot Adapter: Transforms STDIN/STDOUT into safe Request/Response objects

function sendResponse(status, body, headers = {}) {
    try {
        const response = {
            status: status,
            headers: { 
                "Content-Type": "application/json",
                ...headers 
            },
            body: typeof body === 'string' ? body : JSON.stringify(body)
        };
        
        const jsonOutput = JSON.stringify(response);
        const encoder = new TextEncoder();
        Javy.IO.writeSync(1, encoder.encode(jsonOutput));
    } catch (e) {
        // Catastrophic IO failure (rare)
    }
}

globalThis.Gojinn = {
    run: function (userHandler) {
        try {
            // 1. Read Input
            const buffer = new Uint8Array(64 * 1024);
            let bytesRead = 0;
            try {
                bytesRead = Javy.IO.readSync(0, buffer);
            } catch (ioErr) {
                console.error("Read error:", ioErr);
            }

            // 2. Parse and Normalize Input
            let rawInput = {};
            if (bytesRead > 0) {
                const decoder = new TextDecoder();
                const inputStr = decoder.decode(buffer.subarray(0, bytesRead));
                try {
                    rawInput = JSON.parse(inputStr);
                } catch (parseErr) {
                    rawInput = { body: inputStr }; // Fallback to raw text
                }
            }

            // 3. Build the Guaranteed Request Object (Safe Request)
            // This prevents 'cannot read property of undefined' errors
            const req = {
                body: rawInput.body || rawInput, // If body field doesn't exist, use full input
                headers: rawInput.headers || {}, // Ensures headers always exist
                method: rawInput.method || "POST",
                uri: rawInput.uri || "/"
            };

            // 4. Execute User Handler
            const result = userHandler(req);

            // 5. Send Success Response
            sendResponse(
                result.status || 200, 
                result.body || "", 
                result.headers || {}
            );

        } catch (error) {
            // 6. Global Error Handling
            sendResponse(500, {
                error: "Runtime Panic in JS",
                message: error.message || error.toString(),
                stack: error.stack
            });
        }
    }
};
