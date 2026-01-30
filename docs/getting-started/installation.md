# 🛠️ Installation

**Gojinn** is a plugin for the Caddy web server. To use it, you need a version of the Caddy binary that includes the `http.handlers.gojinn` module compiled.

## ✅ Prerequisites

- **Go (Golang):** Version 1.25 or higher installed
- **Make:** Standard build tool (usually pre-installed on Linux/Mac)
- **Terminal Access**

---

## 🚀 Method 1: Using Makefile (Recommended)

If you have cloned the repository, this is the easiest way to build the entire stack (Caddy Host + WASM Functions) with a single command.

### 1. Clone the repository

```bash
git clone https://github.com/pauloappbr/gojinn.git
cd gojinn
```

### 2. Build Everything

This command installs xcaddy if missing, compiles the Caddy binary with the plugin, and compiles all example functions (SQL, Counter, etc).

```bash
make all
```

### 3. Run the Server

```bash
make run
```

## 🛠️ Method 2: Using xcaddy (Manual)

If you prefer to build only the binary manually or integrate it into a custom build pipeline.

### 1. Install xcaddy

```bash
go install github.com/caddyserver/xcaddy/cmd/xcaddy@latest
```

### 2. Compile Caddy with Gojinn

Run the command below to generate the caddy binary in the current directory:

```bash
xcaddy build \
    --with github.com/pauloappbr/gojinn
```

### 3. Verify the installation

Confirm that the module was installed correctly:

```bash
./caddy list-modules | grep gojinn
```

Expected output:

```text
http.handlers.gojinn
```

## 💻 Method 3: Local Development

If you are contributing to the Gojinn source code and want to test changes live without pushing to Git:

### 1. Clone the repository

```bash
git clone https://github.com/pauloappbr/gojinn.git
```

### 2. Compile with local replacement

Use replace to point to your local folder:

```bash
xcaddy build \
    --with github.com/pauloappbr/gojinn=./
```

(Note: The Makefile included in the repo already handles this relative path logic automatically).

## 📚 Next Steps

Now that you have the Gojinn binary installed, the next step is to create your first function.

👉 [Quickstart (5 min)](../quickstart.md)