function readInput() {
    const buffer = new Uint8Array(1024 * 1024); // 1MB buffer
    const n = Javy.IO.readSync(0, buffer); // 0 = stdin
    const inputStr = new TextDecoder().decode(buffer.subarray(0, n));
    return JSON.parse(inputStr);
}

function writeOutput(output) {
    const jsonStr = JSON.stringify(output);
    const encoder = new TextEncoder();
    const buffer = encoder.encode(jsonStr);
    Javy.IO.writeSync(1, buffer); // 1 = stdout
}

export function handle(handlerFunction) {
    try {
        const input = readInput();
        
        const result = handlerFunction(input);
        
        const response = {
            body: result.body || "",
            headers: result.headers || { "Content-Type": "text/plain" },
            status: result.status || 200
        };
        
        writeOutput(response);
    } catch (e) {
        writeOutput({
            body: `Function Error: ${e.message}`,
            headers: { "Content-Type": "text/plain" },
            status: 500
        });
    }
}