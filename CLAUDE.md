# Query Library — Claude Code Guidelines

## Overview

Pure Go query language library for Trazo. Handles lexing, parsing, AST construction,
and validation of a unified query syntax. Zero external dependencies (stdlib only).
WASM-compilable for client-side use.

## Tech Stack

| Library | Purpose |
|---------|---------|
| Go 1.25+ | Standard library only — no third-party deps |

## Project Structure

```
query/
├── token.go           # Token types and Position
├── lexer.go           # Lexer: string → token stream
├── ast.go             # AST node types, Walk(), String()
├── parser.go          # Recursive descent parser
├── validator.go       # AST validator against FieldConfig
├── field_config.go    # Field types, operators, config
├── errors.go          # Structured errors with position info
├── query.go           # Public API: Parse(), Validate()
├── *_test.go          # Table-driven tests for each component
└── wasm/              # WASM build target (Phase 4)
```

## Architecture Decisions — DO NOT DEVIATE

1. **Zero external dependencies** — stdlib only. No testify, no third-party.
2. **Single flat package** — all types in package `query` at module root.
3. **WASM-compatible** — no OS-specific code in the library.
4. **Consumers own SQL generation** — this library only parses and validates.

## Code Conventions

### Import Order

```go
import (
    // 1. Standard library
    "fmt"
    "strings"

    // 2. Internal (none expected — flat package)
)
```

### Error Handling

- `QueryError` with `Position` for all parse/validation errors.
- `ErrorList` collects multiple errors (validator does not stop at first).
- `fmt.Errorf("doing X: %w", err)` for internal wrapping.

### Test Naming

- `Test<Function>_<Scenario>` with `t.Run` subtests.
- AAA pattern: Arrange, Act, Assert.
- Table-driven for all token types, operators, grammar productions.
- Use stdlib `testing` only — no testify.

## Commit Format

```
<type>(<scope>): <description>
```

- Types: `feat`, `fix`, `refactor`, `test`, `chore`, `docs`
- Scopes: `lexer`, `parser`, `ast`, `validator`, `api`, `errors`, `wasm`, `ci`
- No `Co-authored-by` for Claude.

## Common Commands

```bash
make build      # compile
make test       # unit tests with -race
make lint       # golangci-lint
make fmt        # gofmt + goimports
make vet        # go vet
make check      # fmt + vet + lint + test
make coverage   # test with coverage report
make wasm       # build WASM target
```
