# Deployment & Operations

This guide covers how to take Gojinn from a development laptop to a production Linux server using best practices (Systemd, Log Rotation, and CI/CD).

## 1. Build Pipeline (Makefile)

In production, we want optimized binaries (smaller size, no debug symbols). Use the included `Makefile` to automate compilation.

```bash
# Clean previous builds
make clean

# Build optimized binaries for Host (Caddy) and Guests (WASM)
make all
```

This generates the `gojinn-server` binary with `-ldflags "-s -w"` to strip debug information.

## 2. Linux Service (Systemd)

Never run Gojinn inside a tmux or screen session. Use Systemd to manage the process lifecycle.

### Service File (`/etc/systemd/system/gojinn.service`)

```ini
[Unit]
Description=Gojinn - In-Process Serverless Runtime
After=network.target network-online.target
Requires=network-online.target

[Service]
Type=notify
User=gojinn
Group=gojinn
WorkingDirectory=/home/gojinn/app
ExecStart=/usr/local/bin/gojinn run --environ --config /home/gojinn/app/Caddyfile
ExecReload=/usr/local/bin/gojinn reload --config /home/gojinn/app/Caddyfile
TimeoutStopSec=5s
LimitNOFILE=1048576
PrivateTmp=true
ProtectSystem=full
AmbientCapabilities=CAP_NET_BIND_SERVICE

[Install]
WantedBy=multi-user.target
```

### Enable and Start

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now gojinn
```

## 3. Log Rotation & Management

By default, Caddy logs to stderr, which is captured by Journald. For high-traffic production, configure Caddy to write to a dedicated file with auto-rotation.

### Directory Setup

```bash
sudo mkdir -p /var/log/gojinn
sudo chown -R gojinn:gojinn /var/log/gojinn
sudo chmod 700 /var/log/gojinn
```

### Caddyfile Configuration

Add the log directive to the Global Options block:

```caddy
{
    order gojinn last
    
    log {
        output file /var/log/gojinn/access.log {
            roll_size 100mb
            roll_keep 5
            roll_keep_for 720h
        }
        format json
        level INFO
    }
}
```

## 4. Continuous Deployment (Git-Push-to-Deploy)

To automate updates without downtime, you can use a simple shell script (`deploy.sh`) on the server.

### Workflow

1. Push code to main branch.
2. SSH into server.
3. Run `./deploy.sh`.

### The Script

The script performs the following actions:

- `git pull` updates the code.
- `make all` recompiles binaries.
- Stops the service.
- Replaces the binary in `/usr/local/bin`.
- Restarts the service.

```bash
#!/bin/bash
set -e

git pull origin main
make all

sudo systemctl stop gojinn
sudo mv ./gojinn-server /usr/local/bin/gojinn
sudo systemctl start gojinn

echo "[deploy] Deployment completed successfully"
```

---
