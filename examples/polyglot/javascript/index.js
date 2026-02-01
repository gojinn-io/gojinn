import { handle } from "../../sdk/js/gojinn.js";

function myHandler(req) {
    return {
        body: `Hello from JavaScript! You sent: ${req.body}`,
        headers: { "X-Powered-By": "Gojinn-JS" },
        status: 200
    };
}

handle(myHandler);
