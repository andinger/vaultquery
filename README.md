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
| **Token efficiency** | High overhead from punctuation | ~20-30% fewer tokens |
| **Tooling** | `jq`, every language | Growing, Go library available |
| **Best for** | Pipelines, API integration | LLM/agent consumption, human review |

### Why TOON is ideal for agentic workflows

When an LLM agent calls vaultquery as a tool, the output is fed back into the model's context window. TOON's minimal syntax — no commas, no colons, no redundant quoting — means the same data uses significantly fewer tokens than JSON. In practice this translates to:

- **More results per context window** — agents can retrieve larger datasets without hitting token limits
- **Lower cost** — fewer input tokens per tool call directly reduces API spend on token-priced models
- **Faster responses** — less input to process means lower latency for the agent's next reasoning step
- **Equally parseable** — LLMs read TOON at least as reliably as JSON; the structure is unambiguous and whitespace-delimited

If you're building an agent that queries an Obsidian vault (e.g. via MCP tool-use, Claude Code, or any LLM-driven pipeline), `--format toon` is the recommended output format. See the [TOON specification](https://github.com/toon-format/toon) for detailed benchmarks and design rationale.

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

## Comparison with Obsidian MCP

Obsidian MCP plugins like [obsidian-connect-mcp](https://github.com/joch/obsidian-connect-mcp) give LLM agents direct access to a running Obsidian instance — including Dataview queries, vault search, and structured JSON output. If you already have Obsidian running, they're an excellent choice for agent workflows.

vaultquery fills a different niche: it works **without Obsidian** and runs as a standalone CLI.

| | vaultquery | Obsidian MCP |
|---|---|---|
| **Requires Obsidian** | No — headless, works on any `.md` vault | Yes — needs a running Obsidian instance |
| **Query language** | DQL (Dataview-compatible subset) | Dataview queries (via Obsidian plugin) |
| **Output** | JSON and TOON | JSON |
| **Scriptable** | CLI tool, pipes into `jq`, shell scripts, CI | Designed for LLM tool-use via MCP |
| **Index** | SQLite, incremental mtime+size sync | Obsidian's internal index |

Use vaultquery when you need automation, scripting, CI pipelines, or headless environments where Obsidian is not installed or running. Use Obsidian MCP when you already have Obsidian open and want the richest possible integration.

## Acknowledgements

- [Dataview](https://github.com/blacksmithgu/obsidian-dataview) by Michael Brenan — Dataview's query language (DQL) and its approach to treating frontmatter as queryable data were the foundation for this tool.
- [Obsidian](https://obsidian.md) — the knowledge base that makes all of this worth building.
- [obsidian-connect-mcp](https://github.com/joch/obsidian-connect-mcp) by Jonas Chodorski — a great MCP server for connecting LLM agents directly to a running Obsidian instance.

## License

MIT
