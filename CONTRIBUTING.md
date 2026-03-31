# Contributing to API Cerberus

Thank you for your interest in contributing to API Cerberus! This document provides guidelines for contributing.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Workflow](#development-workflow)
- [Commit Guidelines](#commit-guidelines)
- [Testing](#testing)
- [Documentation](#documentation)
- [Security](#security)

## Code of Conduct

This project follows the [Contributor Covenant Code of Conduct](CODE_OF_CONDUCT.md). By participating, you are expected to uphold this code.

## Getting Started

### Prerequisites

- Go 1.26 or later
- Node.js 20 or later (for web UI)
- Docker (for integration tests)
- Make

### Setting Up Development Environment

1. Fork the repository
2. Clone your fork:
   ```bash
   git clone https://github.com/YOUR_USERNAME/APICerebrus.git
   cd APICerebrus
   ```

3. Install dependencies:
   ```bash
   go mod download
   cd web && npm install
   ```

4. Build the project:
   ```bash
   make build
   ```

5. Run tests:
   ```bash
   make test
   ```

## Development Workflow

### Branching Strategy

- `main` - Production-ready code
- `develop` - Development branch
- `feature/*` - Feature branches
- `bugfix/*` - Bug fix branches
- `hotfix/*` - Hotfix branches

### Creating a Feature

1. Create a new branch from `develop`:
   ```bash
   git checkout develop
   git pull origin develop
   git checkout -b feature/my-feature
   ```

2. Make your changes

3. Write or update tests

4. Update documentation

5. Run the full test suite:
   ```bash
   make test-all
   ```

6. Commit your changes (see [Commit Guidelines](#commit-guidelines))

7. Push to your fork:
   ```bash
   git push origin feature/my-feature
   ```

8. Open a Pull Request

## Commit Guidelines

We follow the [Conventional Commits](https://www.conventionalcommits.org/) specification.

### Format

```
<type>(<scope>): <description>

[optional body]

[optional footer(s)]
```

### Types

- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes (formatting, etc.)
- `refactor`: Code refactoring
- `perf`: Performance improvements
- `test`: Test changes
- `chore`: Build process or auxiliary tool changes

### Examples

```
feat(auth): add OAuth2 support

Implement OAuth2 authorization code flow for user authentication.

Closes #123
```

```
fix(router): correct radix tree traversal

Fixed an edge case where routes with common prefixes were incorrectly matched.

Fixes #456
```

```
docs: update installation instructions

Updated README with new Docker installation steps.
```

## Testing

### Unit Tests

```bash
go test ./...
```

### Integration Tests

```bash
go test -tags=integration ./test/...
```

### E2E Tests

```bash
go test -tags=e2e ./test/...
```

### Benchmark Tests

```bash
go test -bench=. -benchmem ./...
```

### Race Detection

```bash
go test -race ./...
```

### Coverage

```bash
make coverage
```

## Code Quality

### Linting

We use `golangci-lint`:

```bash
make lint
```

### Code Formatting

```bash
go fmt ./...
```

### Import Organization

Imports should be organized as:
1. Standard library
2. Third-party packages
3. Internal packages

## Documentation

### Code Documentation

- All public functions, types, and constants must have Go doc comments
- Use complete sentences
- Start with the name of the thing being documented

```go
// User represents a user in the system.
type User struct {
    // ID is the unique identifier for the user.
    ID string
}

// Validate checks if the user is valid.
func (u *User) Validate() error {
    // ...
}
```

### Architecture Documentation

Update relevant documentation in `/docs` when:
- Adding new features
- Changing existing architecture
- Deprecating features

### API Documentation

Update OpenAPI specs in `/docs/openapi.yaml` for API changes.

## Security

### Reporting Security Issues

Please do NOT report security issues in public GitHub issues. Instead, email security@apicerberus.com.

### Security Checklist

- [ ] Input validation on all endpoints
- [ ] Authentication checks
- [ ] Authorization checks
- [ ] Rate limiting
- [ ] SQL injection prevention
- [ ] XSS prevention
- [ ] CSRF protection

## Performance

### Benchmarking

When making performance-critical changes:

1. Run benchmarks before and after:
   ```bash
   go test -bench=. -benchmem -count=5 ./... > benchmark-old.txt
   # make changes
   go test -bench=. -benchmem -count=5 ./... > benchmark-new.txt
   ```

2. Use `benchstat` to compare:
   ```bash
   benchstat benchmark-old.txt benchmark-new.txt
   ```

### Profiling

```bash
go test -cpuprofile=cpu.prof -memprofile=mem.prof -bench=. ./...
go tool pprof cpu.prof
```

## Release Process

1. Update version in `internal/version/version.go`
2. Update CHANGELOG.md
3. Create a tag:
   ```bash
   git tag -a v1.x.x -m "Release v1.x.x"
   git push origin v1.x.x
   ```
4. GitHub Actions will automatically create a release

## Getting Help

- GitHub Discussions: https://github.com/APICerberus/APICerebrus/discussions
- Slack: https://apicerberus.slack.com
- Email: maintainers@apicerberus.com

## Recognition

Contributors will be recognized in:
- README.md
- CONTRIBUTORS.md
- Release notes

Thank you for contributing to API Cerberus!
