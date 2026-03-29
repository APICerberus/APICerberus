# API Cerberus

[![Build](https://img.shields.io/badge/build-pending-lightgrey)](#)
[![Coverage](https://img.shields.io/badge/coverage-pending-lightgrey)](#)
[![License: MIT](https://img.shields.io/badge/license-MIT-green.svg)](./LICENSE)

API Cerberus is a full-stack API gateway and API management platform written in pure Go.

## Project Overview

API Cerberus aims to provide HTTP/HTTPS, gRPC, and GraphQL gateway capabilities,
including policy enforcement, billing, audit logging, analytics, and admin/user interfaces.

## Current Status

Initial scaffolding is ready (v0.0.1 / section 1.1).

## Quick Start

```bash
make build
./bin/apicerberus --version
```

## Repository Layout

- `cmd/apicerberus`: application entrypoint
- `internal`: private packages
- `web`: frontend assets and dashboard app
- `test`: integration and end-to-end test assets

## Roadmap

See `.project/TASKS.md` for implementation milestones.
