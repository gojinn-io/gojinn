# Support for Gojinn

This document explains **how to get help**, **what kind of support is available**, and **what is out of scope** for the Gojinn project.

Gojinn is an **open-source project**, developed and maintained by the community.  
Support is provided on a **best-effort basis**.

---

## 🆘 How to Get Help

### 1. GitHub Issues (Primary Channel)

GitHub Issues are the main support channel for Gojinn.

You may open an issue for:
- bug reports
- reproducible crashes
- documentation issues
- runtime inconsistencies
- performance regressions

When opening an issue, please include:
- Gojinn version
- Caddy version
- Operating System
- minimal reproduction (Caddyfile + WASM module)
- expected vs actual behavior

Issues without sufficient information **may be closed without response**.

---

### 2. GitHub Discussions (Community Support)

GitHub Discussions are recommended for:
- usage questions
- architectural discussions
- design proposals
- roadmap feedback
- community ideas

This is the best place to:
- ask “how should I do X?”
- validate an idea before opening a PR
- discuss trade-offs

---

## 🔐 Security Issues

**DO NOT** report security vulnerabilities via GitHub Issues or Discussions.

If you discover a security vulnerability:
- refer to `SECURITY.md`
- follow the private disclosure process described there

Public disclosure of vulnerabilities may result in issue removal.

---

## ❌ What Gojinn Does NOT Provide Support For

To keep the project sustainable, the following are **explicitly out of scope**:

- Debugging user-written WASM code
- Teaching programming languages
- Migrating applications from other cloud providers
- Kubernetes, Docker, or VM configuration
- Infrastructure sizing or capacity planning
- Production SRE/on-call support

Gojinn provides a runtime — **not managed cloud services**.

---

## 🧪 Experimental Features

Some features may be marked as **experimental**.

Experimental features:
- are not guaranteed to be stable
- may change or be removed without notice
- are not eligible for support beyond basic bug reports

Use experimental features **at your own risk**.

---

## 🧭 Support Philosophy

Gojinn follows a **correctness-first** support philosophy:

- correctness over convenience
- reproducibility over speculation
- clarity over speed

We prioritize issues that:
- affect security
- break the core runtime invariant
- impact deterministic execution
- affect multiple users

---

## 🤝 Community Expectations

When requesting support, please:
- be respectful and precise
- avoid assumptions about internals
- understand that maintainers are volunteers
- read existing documentation before asking

Hostile, demanding, or entitled behavior will not be tolerated.

---

## 💼 Commercial Support

At this time, Gojinn does **not** offer:
- paid support
- SLAs
- enterprise contracts

This may change in the future, but today Gojinn remains a **community-driven project**.

---

## 📌 Final Note

If you are unsure whether something belongs in:
- Issues → reproducible problems
- Discussions → questions or ideas

When in doubt, start with **Discussions**.

Thank you for supporting Gojinn.
