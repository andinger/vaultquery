# Contributing to vaultquery

## Development Setup

```bash
git clone https://github.com/andinger/vaultquery.git
cd vaultquery
go mod download
go test -race ./...
```

Requires Go 1.23+.

## Project Structure

```
cmd/vaultquery/         Entry point + E2E tests
internal/
  cli/                  cobra commands, CLI wiring
  config/               XDG paths, vault root resolution
  dql/                  DQL lexer, parser, AST
  executor/             AST-to-SQL translation, query execution
  index/                SQLite schema, store operations
  indexer/              Filesystem scanning, frontmatter parsing, change detection
```

## Running Tests

```bash
# All tests with race detector
go test -race ./...

# Specific package
go test -race ./internal/dql/...

# Verbose
go test -v -race ./internal/executor/...
```

All test data uses fictional companies (Acme Corp, Globex Inc, Initech). No real vault data is used in tests.

## Code Style

- Run `go vet ./...` before committing
- Follow standard Go conventions
- Keep dependencies minimal

## Pull Requests

1. Fork the repository
2. Create a feature branch
3. Write tests for new functionality
4. Ensure `go test -race ./...` passes
5. Submit a PR with a clear description
