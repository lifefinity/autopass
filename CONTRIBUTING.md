# Contributing

Thanks for your interest in contributing to autopass!

## Getting Started

1. Fork the repository
2. Clone your fork: `git clone https://github.com/<you>/autopass.git`
3. Install dependencies: `go mod download`
4. Create a branch: `git checkout -b my-feature`

## Development

```bash
make build    # Build
make test     # Run tests
make check    # Full check: fmt + vet + lint + sec + vuln + test
```

See [docs/development.md](docs/development.md) for detailed setup instructions.

## Pull Request Guidelines

- Keep PRs focused on a single change
- Add tests for new functionality
- Run `make check` before submitting
- Follow existing code style (enforced by `go fmt` and `golangci-lint`)
- Update documentation if behavior changes

## Commit Messages

Use conventional commit style:

```
feat: add timeout flag to update command
fix: handle empty pattern in matcher
docs: update examples in user guide
test: add property tests for crypto module
```

## Reporting Issues

- Check existing issues first
- Include OS, Go version, and steps to reproduce
- For security issues, email directly instead of opening a public issue

## Code of Conduct

Be respectful and constructive. We're all here to build something useful.
