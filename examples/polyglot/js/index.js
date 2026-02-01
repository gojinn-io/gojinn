Gojinn.run((req) => {
    const name = req.headers["User"] || "World";
    
    return {
        status: 200,
        headers: { "X-Gojinn-Lang": "JavaScript" },
        body: {
            message: `Hello, ${name}!`,
            original_body: req.body,
            timestamp: new Date().toISOString()
        }
    };
});
