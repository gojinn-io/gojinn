# Governance of Gojinn

## 1. Purpose

This document defines how Gojinn is governed, how technical decisions are made, and which principles are non-negotiable.

Gojinn is an open-source project, but it is not directionless.
Its governance exists to protect the integrity, security, and long-term vision of the runtime.

## 2. Project Philosophy

Gojinn is built around a small set of strict, opinionated principles.

Gojinn does not aim to be flexible.
Gojinn aims to be correct, deterministic, and sovereign.

We explicitly reject complexity that emerges from:

- implicit state
- global permissions
- opaque abstractions
- infrastructure leakage

## 3. Core Runtime Invariant (Non-Negotiable)

Every change to Gojinn must preserve the following invariant:

> **All user code executes inside a deterministic, isolated, ephemeral WASM sandbox, and is never trusted by default.**

This means:

- **Deterministic**: Identical inputs must produce identical observable behavior.
- **Isolated**: Crashes, panics, or resource exhaustion must not affect the host or other executions.
- **Ephemeral**: No implicit state survives execution unless explicitly persisted via approved host interfaces.
- **Capability-based**: Access to host features (KV, DB, FS, AI, network) must be explicitly granted.

Any proposal that violates this invariant will be rejected, regardless of performance or convenience.

## 4. Decision-Making Model

### 4.1 Benevolent Dictator for Life (BDFL)

Gojinn follows a BDFL governance model.

The Project Founder has final authority over:

- architecture
- invariants
- security model
- long-term roadmap

Community input is strongly encouraged, but final decisions are made to protect coherence and safety.

This model exists to:

- avoid design-by-committee
- prevent fragmentation
- maintain a clear technical vision

### 4.2 Maintainers

Maintainers may be appointed by the Founder based on:

- sustained high-quality contributions
- architectural understanding
- alignment with project principles

Maintainers may:

- review PRs
- propose changes
- participate in roadmap discussions

Maintainers do not override core invariants.

## 5. Public API Policy

### 5.1 API Stability Contract

Public APIs are long-term contracts.

Once an API is marked as public:

- breaking changes require a major version
- deprecations must be documented
- compatibility is preserved whenever possible

### 5.2 What Will NEVER Be a Public API

The following are explicitly out of scope for public APIs:

- internal scheduler mechanics
- VM pooling strategies
- execution heuristics
- internal context representations
- memory layouts
- threading or concurrency internals

These details are implementation-specific and may change at any time.

### 5.3 What MAY Become a Public API

- WASM host bindings (KV, DB, Queue, AI, FS)
- event contracts (HTTP, Cron, Async)
- language SDK interfaces
- CLI commands and flags

## 6. Feature Acceptance Criteria

All new features must answer YES to the following:

- Does this preserve determinism?
- Does this avoid implicit state?
- Does this avoid exposing the host?
- Does this reduce, not increase, operational complexity?

Features that require:

- extensive configuration
- YAML-heavy workflows
- global mutable state
- privileged execution

do not belong in the core runtime.

## 7. Scope Control: Avoiding Kubernetes 2.0

Gojinn explicitly is not:

- a container orchestrator
- a VM manager
- a network fabric
- a storage orchestrator

Gojinn only understands:

- code
- events
- capabilities

Infrastructure management is intentionally left outside the runtime.

## 8. Experimental Features

Experimental features may exist behind:

- feature flags
- unstable namespaces
- non-default builds

Experimental features:

- carry no stability guarantees
- may be removed without notice
- must never weaken the core invariant

## 9. Community Expectations

We value contributors who:

- think in systems, not shortcuts
- prioritize correctness over convenience
- understand the cost of public APIs

We do not prioritize:

- feature requests that mimic existing clouds
- requests for vendor compatibility
- abstractions that hide execution cost

## 10. Amendments

This governance document may evolve, but:

- the core invariant
- the security model
- the scope boundaries

are considered foundational and require exceptional justification to change.