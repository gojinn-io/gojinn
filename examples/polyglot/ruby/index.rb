require 'json'

# Read from STDIN
input = $stdin.read
request = input.empty? ? {} : JSON.parse(input) rescue {}

user_agent = request.dig('headers', 'User-Agent', 0) || 'Unknown'

response = {
  status: 200,
  headers: {
    "X-Gojinn-Lang" => "Ruby #{RUBY_VERSION}",
    "X-Gem-Power" => "True",
    "Content-Type" => "application/json"
  },
  body: JSON.generate({
    message: "Hello from Ruby via WebAssembly! 💎",
    runtime: "Ruby #{RUBY_VERSION}",
    your_agent: user_agent,
    time: Time.now.to_s
  })
}

# Write to STDOUT
puts JSON.generate(response)
