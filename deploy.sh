#!/bin/bash

# Exit immediately if any command fails
set -e

echo "[deploy] Starting Gojinn update process..."

# 1. Fetch latest code from Git (if applicable)
# If you deploy manually or edit locally, this step can be skipped.
echo "[1/5] Updating repository..."
git pull origin main || echo "[warn] Git pull failed or not configured. Using local code."

# 2. Rebuild everything (Host + WASM)
echo "[2/5] Building binaries..."
make all

# 3. Stop the service to replace the binary
echo "[3/5] Stopping service..."
sudo systemctl stop gojinn

# 4. Install the new binary
echo "[4/5] Installing new binary to /usr/local/bin..."
sudo mv ./gojinn-server /usr/local/bin/gojinn
sudo chmod +x /usr/local/bin/gojinn

# 5. Restart the service
echo "[5/5] Starting service..."
sudo systemctl start gojinn

# 6. Show service status
echo "[deploy] Deployment completed successfully. Current status:"
sudo systemctl status gojinn --no-pager
