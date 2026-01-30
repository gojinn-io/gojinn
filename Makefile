# Gojinn Makefile
# Automates building the Host (Caddy) and WASM Functions

# Host configuration
CADDY_BIN := ./gojinn-server
XCADDY_CMD := xcaddy build --with github.com/pauloappbr/gojinn=.

# Build flags (production)
# -s -w: strip debug symbols to reduce binary size
LDFLAGS := -ldflags "-s -w"

.PHONY: all clean run dev build-host build-funcs

# Default target
all: build-funcs build-host

# --- 1. Build Host (Caddy + Gojinn plugin) ---
build-host:
	@echo "Building Caddy host..."
	@$(XCADDY_CMD) --output $(CADDY_BIN)
	@echo "Host binary generated at $(CADDY_BIN)"

# --- 2. Build WASM Functions ---
build-funcs:
	@echo "Building WASM functions..."
	@GOOS=wasip1 GOARCH=wasm go build -o functions/sql.wasm functions/sql/main.go
	@GOOS=wasip1 GOARCH=wasm go build -o functions/counter.wasm functions/counter/main.go
	@# Add additional functions here as needed
	@echo "WASM functions build completed."

# --- 3. Development Mode (build and run) ---
dev: build-funcs
	@echo "Starting development mode..."
	@$(XCADDY_CMD)
	@./caddy run

# --- 4. Production Mode (run optimized binary) ---
run:
	@echo "Starting production server..."
	@$(CADDY_BIN) run

# --- 5. Cleanup ---
clean:
	@rm -f $(CADDY_BIN) caddy
	@rm -f functions/*.wasm
	@echo "Build artifacts removed."
