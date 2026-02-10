# Contributing to Gojinn

First off, thank you for considering contributing to Gojinn! It's people like you that make Gojinn such a great tool.

## 🛠️ Development Workflow

We use a standard **Fork & Pull Request** workflow.

### Prerequisites
- **Go 1.25+**
- **Make** (for build automation)
- **Git**

### Setting up your environment

1. **Fork** the repository on GitHub.
2. **Clone** your fork locally:
   ```bash
   git clone https://github.com/pauloappbr/gojinn.git
   cd gojinn
   ```

3. Build the project to ensure everything is working:

```bash
make all
```

### Making Changes

1. Create a new branch for your feature or fix:

```bash
git checkout -b feat/my-awesome-feature
```

2. Make your changes.

3. Run tests before pushing:

```bash
make test
```

### Commit Messages

We follow the Conventional Commits specification. Please structure your commit messages as follows:

- `feat`: for new features
- `fix`: for bug fixes
- `docs`: for documentation changes
- `chore`: for build tasks, dependencies, etc.
- `perf`: for performance improvements

Example:

```text
feat(runtime): implement new memory limit logic
```

## 🐞 Reporting Bugs

If you find a bug, please create an Issue using our template. Include:

- Gojinn version
- Caddy version
- Your Operating System
- A minimal reproduction example (Caddyfile + WASM)

## 🛡️ Vulnerability Reporting

DO NOT open an issue for security vulnerabilities. Please refer to SECURITY.md.
