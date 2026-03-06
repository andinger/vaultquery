# vaultquery

Query Obsidian vault files by YAML frontmatter using a DQL-like query language. Indexes `.md` files into SQLite and outputs results as JSON.

## Install

### Homebrew (macOS / Linux)

```bash
brew install andinger/vaultquery/vaultquery
```

### Binary download

Download a binary from [Releases](https://github.com/andinger/vaultquery/releases), or build from source:

```bash
go install github.com/andinger/vaultquery/cmd/vaultquery@latest
```

## Usage

```bash
# Index the vault (run from vault root, or use --vault)
vaultquery index --vault ~/my-vault

# Query with DQL
vaultquery query "TABLE customer, kubectl_context WHERE type = 'Kubernetes Cluster' SORT customer ASC"

# List all leads that aren't lost
vaultquery query "LIST FROM \"Sales\" WHERE type = 'Lead' AND status != 'lost'"

# Find files with a specific tag
vaultquery query "TABLE customer WHERE tags contains 'linux'"

# Check index status
vaultquery status

# Full reindex (drop + rebuild)
vaultquery reindex --vault ~/my-vault
```

Every `query` command automatically updates the index before executing (incremental, mtime+size based).

## DQL Reference

### Query Modes

| Mode | Description |
|------|-------------|
| `TABLE field1, field2, ...` | Returns specified frontmatter fields |
| `LIST` | Returns only path and title |

### Clauses

| Clause | Description | Example |
|--------|-------------|---------|
| `FROM "path"` | Filter by vault subdirectory | `FROM "Clients"` |
| `WHERE expr` | Filter by field conditions | `WHERE type = 'Server'` |
| `SORT field [ASC\|DESC]` | Sort results | `SORT customer ASC` |
| `LIMIT n` | Limit result count | `LIMIT 10` |
| `GROUP BY field` | Group results | `GROUP BY customer` |
| `FLATTEN field` | Flatten array fields | `FLATTEN tags` |

### WHERE Operators

| Operator | Description | Example |
|----------|-------------|---------|
| `=` | Equals | `status = 'active'` |
| `!=` | Not equals | `status != 'lost'` |
| `<`, `>`, `<=`, `>=` | Comparison | `value > '1000'` |
| `contains` | Array contains value | `tags contains 'linux'` |
| `!contains` | Array doesn't contain | `tags !contains 'deprecated'` |
| `exists` | Field exists | `kubectl_context exists` |
| `!exists` | Field doesn't exist | `notes !exists` |

### Logical Operators

Combine conditions with `AND`, `OR`, and parentheses:

```
WHERE (type = 'Server' OR type = 'Cluster') AND status = 'active'
```

## Output

All output is JSON:

```json
{
  "mode": "TABLE",
  "fields": ["customer", "kubectl_context"],
  "results": [
    {
      "path": "Clients/Acme Corp/Production/CLUSTER.md",
      "title": "Acme Production Cluster",
      "customer": "Acme Corp",
      "kubectl_context": "acme-prod"
    }
  ]
}
```

## Frontmatter

vaultquery indexes YAML frontmatter from `.md` files:

```markdown
---
type: Kubernetes Cluster
customer: Acme Corp
tags:
  - linux
  - production
---
# Acme Production Cluster
```

- All top-level fields are indexed
- Arrays are stored as separate rows (enabling `contains` queries)
- The title is extracted from the first `# heading` after frontmatter

## Configuration

| Flag | Env | Default | Description |
|------|-----|---------|-------------|
| `--vault` | - | Current directory | Vault root path |
| `--db` | - | `~/.local/share/vaultquery/index.db` | SQLite database path |

## Development

```bash
# Run tests
go test -race ./...

# Build
go build ./cmd/vaultquery

# Cross-compile snapshot
goreleaser build --snapshot --clean
```

## License

MIT
