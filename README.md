# vaultquery

Query Obsidian vault files by YAML frontmatter using a DQL-like query language. Indexes `.md` files into SQLite and outputs results as JSON or [TOON](https://toon-format.org).

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

Every `query` command automatically syncs the index before executing (incremental, mtime+size based). Pass `--index-only` to skip syncing and use the existing index as-is.

## Vault-local storage

vaultquery stores its index database in a `.vaultquery/` directory inside each vault root:

```
my-vault/
├── .vaultquery/
│   ├── index.db        # SQLite database
│   ├── config.yaml     # Optional configuration
│   └── .gitignore      # Auto-created, ignores all files
├── Notes/
└── ...
```

This enables indexing multiple vaults independently. The `.vaultquery/` directory and `.gitignore` are created automatically on first index.

### Folder exclusion

Create `.vaultquery/config.yaml` to exclude folders from indexing:

```yaml
exclude:
  - .obsidian
  - .trash
  - Templates
```

Paths are relative to the vault root. Matching is prefix-based (e.g. `Archive/Old` excludes everything under that path). The `.vaultquery` directory itself is always excluded.

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

Default output is JSON:

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

### TOON format

Pass `--format toon` (or set `format: toon` in `.vaultquery/config.yaml`) for [TOON](https://toon-format.org) output:

```
mode TABLE
fields [
  customer
  kubectl_context
]
results [
  {
    path "Clients/Acme Corp/Production/CLUSTER.md"
    title "Acme Production Cluster"
    customer "Acme Corp"
    kubectl_context "acme-prod"
  }
]
```

### JSON vs TOON

| | JSON | TOON |
|---|---|---|
| **Ecosystem** | Universal, supported everywhere | Newer, lightweight |
| **Readability** | Verbose (colons, commas, quoting) | Minimal syntax, easy to scan |
| **Tooling** | `jq`, every language | Growing, Go library available |
| **Best for** | Pipelines, API integration | Human review, config files |

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

| Flag | Default | Description |
|------|---------|-------------|
| `--vault` | Current directory | Vault root path |
| `-v, --verbose` | false | Show detailed progress during indexing |
| `--format` | `json` | Output format: `json` or `toon` (query only, overrides config) |
| `--index-only` | false | Skip index sync, use existing index as-is (query only) |

## Development

```bash
# Run tests
go test -race ./...

# Build
go build ./cmd/vaultquery

# Cross-compile snapshot
goreleaser build --snapshot --clean
```

## Comparison with Obsidian MCP Server

| | vaultquery | Obsidian MCP Server |
|---|---|---|
| **Requires Obsidian** | No — headless, works on any `.md` vault | Yes — needs a running Obsidian instance |
| **Structured output** | JSON and TOON with typed frontmatter fields | Unstructured text |
| **Scriptable** | CLI tool, pipes into `jq`, shell scripts, CI | Designed for LLM tool-use via MCP |
| **Query language** | DQL (Dataview-compatible subset) | Natural language via LLM |
| **Index** | SQLite, incremental mtime+size sync | Obsidian's internal index |

vaultquery is designed for automation, scripting, and headless environments where Obsidian is not installed or running.

## Acknowledgements

vaultquery is heavily inspired by the [Dataview](https://github.com/blacksmithgu/obsidian-dataview) plugin for Obsidian by Michael Brenan. Dataview's query language (DQL) and its approach to treating frontmatter as queryable data were the foundation for this tool. Thank you for creating such a brilliant plugin.

## License

MIT
