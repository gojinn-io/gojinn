<?php
// 1. Disable error display in output (VERY IMPORTANT)
error_reporting(E_ALL);
ini_set('display_errors', '0'); 
ini_set('log_errors', '1');     // Send errors to internal logs, not to output

// 2. Try to safely read standard input
// The @ operator suppresses warnings if it fails
$input = @file_get_contents('php://stdin');

// If it failed or came empty, try reading from php://input (common fallback)
if ($input === false || empty($input)) {
    $input = @file_get_contents('php://input');
}

$request = json_decode($input, true);

// Fallback if nothing worked
if (!$request) {
    $request = ['method' => 'UNKNOWN', 'headers' => []];
}

$userAgent = $request['headers']['User-Agent'][0] ?? 'Unknown';

$response = [
    "status" => 200,
    "headers" => [
        "X-Gojinn-Lang" => "PHP " . phpversion(),
        "X-Elephant-Power" => "True",
        "Content-Type" => "application/json"
    ],
    "body" => json_encode([
        "message" => "Hello from PHP via WebAssembly! 🐘",
        "runtime" => "PHP " . phpversion(),
        "your_agent" => $userAgent,
        "server_time" => date('Y-m-d H:i:s'),
        "debug_input_len" => strlen($input)
    ])
];

echo json_encode($response);
?>
