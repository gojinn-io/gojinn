using System;
using System.Text.Json;
using System.Text.Json.Nodes;

public class Program
{
    public static void Main(string[] args)
    {
        // 1. Read input (STDIN)
        string input = "";
        try 
        {
            // Read the entire input stream
            input = Console.In.ReadToEnd();
        }
        catch { /* Ignore error if empty */ }

        // 2. Process JSON
        JsonNode? json = null;
        if (!string.IsNullOrWhiteSpace(input))
        {
            try { json = JsonNode.Parse(input); } catch { }
        }

        string userAgent = json?["headers"]?["User-Agent"]?[0]?.ToString() ?? "Unknown";
        string runtimeName = ".NET " + Environment.Version.ToString();

        // 3. Build Response (Anonymous Object)
        var response = new
        {
            status = 200,
            headers = new
            {
                X_Gojinn_Lang = "C# / " + runtimeName,
                X_Corporate_Power = "True", // Joke about the corporate world
                Content_Type = "application/json"
            },
            body = JsonSerializer.Serialize(new
            {
                message = "Hello from C# via WebAssembly! 🏢",
                runtime = runtimeName,
                your_agent = userAgent,
                server_time = DateTime.UtcNow.ToString("o")
            })
        };

        // 4. Write Output (STDOUT)
        // Important: .NET may try to write BOM or other data, but Console.WriteLine is usually safe.
        Console.WriteLine(JsonSerializer.Serialize(response));
    }
}
