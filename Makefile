# Gojinn Makefile (Fixed URLs)

# Host configuration
CADDY_BIN := ./gojinn-server
XCADDY_CMD := xcaddy build --with github.com/pauloappbr/gojinn=.

# URLs VMWare Labs
# Python 3.12
URL_PYTHON := "https://github.com/vmware-labs/webassembly-language-runtimes/releases/download/python%2F3.12.0%2B20231211-040d5a6/python-3.12.0.wasm"
# PHP 8.2.6 CGI
URL_PHP    := "https://github.com/vmware-labs/webassembly-language-runtimes/releases/download/php%2F8.2.6%2B20230714-11be424/php-cgi-8.2.6.wasm"
# Ruby 3.2.2
URL_RUBY   := "https://github.com/vmware-labs/webassembly-language-runtimes/releases/download/ruby%2F3.2.2%2B20230714-11be424/ruby-3.2.2.wasm"

LDFLAGS := -ldflags "-s -w"

.PHONY: all clean run dev build-host build-funcs build-polyglot download-runtimes

all: download-runtimes build-funcs build-polyglot build-host

# --- 0. Download Runtimes ---
download-runtimes:
	@echo "⬇️  Checking Polyglot Runtimes..."
	@mkdir -p lib
	@# Verifica se é um arquivo válido (> 1KB) para evitar o erro de "Not Found"
	@if [ ! -s lib/python.wasm ] || [ $$(stat -c%s lib/python.wasm) -lt 1000 ]; then \
		echo "   [PY] Downloading Python 3.12 (approx 20MB)..."; \
		curl -L -o lib/python.wasm $(URL_PYTHON); \
	fi
	@if [ ! -s lib/php.wasm ] || [ $$(stat -c%s lib/php.wasm) -lt 1000 ]; then \
		echo "   [PHP] Downloading PHP 8.2 (approx 15MB)..."; \
		curl -L -o lib/php.wasm $(URL_PHP); \
	fi
	@if [ ! -s lib/ruby.wasm ] || [ $$(stat -c%s lib/ruby.wasm) -lt 1000 ]; then \
		echo "   [RB] Downloading Ruby 3.2 (approx 30MB)..."; \
		curl -L -o lib/ruby.wasm $(URL_RUBY); \
	fi
	@echo "✅ Runtimes ready in ./lib"

# --- 1. Build Host ---
build-host:
	@echo "🏗️  Building Caddy host..."
	@$(XCADDY_CMD) --output $(CADDY_BIN)

# --- 2. Build Go Funcs ---
build-funcs:
	@echo "🐹 Building Go WASM functions..."
	@GOOS=wasip1 GOARCH=wasm go build -o functions/sql.wasm functions/sql/main.go || echo "⚠️ functions/sql not found, skipping"
	@GOOS=wasip1 GOARCH=wasm go build -o functions/counter.wasm functions/counter/main.go || echo "⚠️ functions/counter not found, skipping"
	@GOOS=wasip1 GOARCH=wasm go build -o functions/s3.wasm functions/s3/main.go || echo "⚠️ functions/s3 not found, skipping"
	@GOOS=wasip1 GOARCH=wasm go build -o functions/cron.wasm functions/cron/main.go || echo "⚠️ functions/cron skipping"
	@GOOS=wasip1 GOARCH=wasm go build -o functions/trigger.wasm functions/trigger/main.go || echo "⚠️ trigger skipping"
	@GOOS=wasip1 GOARCH=wasm go build -o functions/worker.wasm functions/worker/main.go || echo "⚠️ worker skipping"
	@GOOS=wasip1 GOARCH=wasm go build -o functions/fail.wasm functions/fail/main.go || echo "⚠️ fail skipping"
	@GOOS=wasip1 GOARCH=wasm go build -o functions/trigger-fail.wasm functions/trigger-fail/main.go || echo "⚠️ trigger-fail skipping"
	@GOOS=wasip1 GOARCH=wasm go build -o functions/iot.wasm functions/iot/main.go || echo "⚠️ iot skipping"

# --- 3. Build Polyglot ---
build-polyglot: download-runtimes
	@echo "📜 Building Polyglot functions..."
	@mkdir -p functions

	@# JS
	@if command -v javy >/dev/null 2>&1; then \
		cat sdk/js/shim.js examples/polyglot/js/index.js > functions/js_temp.js; \
		javy build functions/js_temp.js -o functions/js.wasm; \
		rm functions/js_temp.js; \
	else \
		echo "   ⚠️  Javy not found. Skipping JS."; \
	fi
	
	@# Copia os runtimes baixados para a pasta functions
	@cp lib/python.wasm functions/python.wasm
	@cp lib/php.wasm functions/php.wasm
	@cp lib/ruby.wasm functions/ruby.wasm
	@echo "✅ Polyglot build complete."

# --- 4. Dev Mode ---
dev: build-funcs build-polyglot
	@echo "🚀 Starting development mode..."
	@$(XCADDY_CMD)
	@./caddy run

clean:
	@rm -f $(CADDY_BIN) caddy functions/*.wasm