# query

[![CI](https://github.com/trazo-lat/query/actions/workflows/ci.yml/badge.svg)](https://github.com/trazo-lat/query/actions/workflows/ci.yml)

Pure Go query language library for Trazo. Handles lexing, parsing, AST construction, and validation of a unified query syntax used across all clients (CLI, Web UI, API, VS Code, WASM).

Zero dependencies on core-service. This library is the shared baseline for query parsing and validation -- all semantic evaluation (SQL generation, execution) lives in the consumer.

## Install

```bash
go get github.com/trazo-lat/query
```

## Usage

```go
import "github.com/trazo-lat/query"

// Parse a query string into an AST.
expr, err := query.Parse("state=draft AND total>50000")

// Validate against field declarations.
fields := []query.FieldConfig{
    {Name: "state", Type: query.TypeText, AllowedOps: query.TextOps},
    {Name: "total", Type: query.TypeDecimal, AllowedOps: query.NumericOps},
}
err = query.Validate(expr, fields)

// Or do both in one call.
expr, err = query.ParseAndValidate("state=draft", fields)
```

## Query Syntax

```
state=draft                                        # equality
state!=cancelled                                   # not equal
year>2020                                          # comparison
name=John*                                         # wildcard (prefix, suffix, contains)
tire_size                                          # presence
state=draft AND customer_id=customer_john-doe      # logical AND
(state=draft OR state=issued) AND total>50000      # grouping
NOT state=cancelled                                # negation
created_at:2026-01-01..2026-03-31                  # range
ttl.duration>1d                                    # duration
labels.dev=jane                                    # nested fields
```

## Operators

| Operator | Types | Description |
|----------|-------|-------------|
| `=` | all | Equality or wildcard match |
| `!=` | all | Not equal |
| `>` `>=` `<` `<=` | number, date, duration | Comparison |
| `..` | number, date, duration | Inclusive range (`field:start..end`) |
| `NOT` | expression | Negation |
| `AND` | expressions | Logical AND |
| `OR` | expressions | Logical OR |

## License

Apache License 2.0
