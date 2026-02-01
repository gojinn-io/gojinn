# Gojinn Makefile
# Automates building the Host (Caddy) and WASM Functions

# Host configuration
CADDY_BIN := ./gojinn-server
XCADDY_CMD := xcaddy build --with github.com/pauloappbr/gojinn=.

# Build flags (production)
# -s -w: strip debug symbols to reduce binary size
LDFLAGS := -ldflags "-s -w"

.PHONY: all clean run dev build-host build-funcs build-polyglot

# Default target (Agora inclui o Polyglot)
all: build-funcs build-polyglot build-host

# --- 1. Build Host (Caddy + Gojinn plugin) ---
build-host:
	@echo "🏗️  Building Caddy host..."
	@$(XCADDY_CMD) --output $(CADDY_BIN)
	@echo "✅ Host binary generated at $(CADDY_BIN)"

# --- 2. Build WASM Functions (Go) ---
build-funcs:
	@echo "🐹 Building Go WASM functions..."
	@GOOS=wasip1 GOARCH=wasm go build -o functions/sql.wasm functions/sql/main.go
	@GOOS=wasip1 GOARCH=wasm go build -o functions/counter.wasm functions/counter/main.go
	@echo "✅ Go WASM functions build completed."

# --- 2.1 Build Polyglot Functions (JS/Python/PHP/Ruby) ---
build-polyglot:
	@echo "📜 Building Polyglot functions..."
	@mkdir -p functions

	@# --- JAVASCRIPT ---
	@echo "   [JS] Bundling..."
	@cat sdk/js/shim.js examples/polyglot/js/index.js > functions/js_temp.js
	@javy build functions/js_temp.js -o functions/js.wasm
	@rm functions/js_temp.js
    
	@# --- PYTHON ---
	@echo "   [PY] Copying Runtime..."
	@cp lib/python.wasm functions/python.wasm

	@# --- PHP ---
	@echo "   [PHP] Copying Runtime..."
	@cp lib/php.wasm functions/php.wasm

	@echo "   [RB] Copying Runtime..."
	@cp lib/ruby.wasm functions/ruby.wasm
    
	@echo "✅ Polyglot build complete."

# --- 3. Development Mode (build and run) ---
dev: build-funcs build-polyglot
	@echo "🚀 Starting development mode..."
	@$(XCADDY_CMD)
	@./caddy run

# --- 4. Production Mode (run optimized binary) ---
run:
	@echo "🚀 Starting production server..."
	@$(CADDY_BIN) run

# --- 5. Cleanup ---
clean:
	@rm -f $(CADDY_BIN) caddy
	@rm -f functions/*.wasm
	@echo "🧹 Build artifacts removed."