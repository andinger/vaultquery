# vaultquery

Query Obsidian vault files by YAML frontmatter using a DQL-like query language. Indexes `.md` files into SQLite and outputs results as JSON or [TOON](https://toon-format.org).

## Install

### Homebrew (macOS / Linux)

```bash
brew install andinger/tap/vaultquery
```

### Binary download

Download a binary from [Releases](https://github.com/andinger/vaultquery/releases), or build from source:

```bash
go install github.com/andinger/vaultquery/cmd/vaultquery@latest
```

## Usage

```bash
# Query with DQL (auto-syncs index before each query)
vaultquery query "TABLE customer, status FROM \"Clients\" WHERE type = \"Lead\" SORT customer ASC" --vault ~/my-vault

# List all files with a specific tag
vaultquery query "LIST FROM #project WHERE status != \"archived\"" --vault ~/my-vault

# Find files linking to a specific page
vaultquery query "TABLE file.name FROM [[My Note]]" --vault ~/my-vault

# TOON output (recommended for LLM/agent workflows)
vaultquery query "TABLE customer WHERE type = \"Server\"" --vault ~/my-vault --format toon
```

Every `query` command automatically syncs the index before executing (incremental, mtime+size based). Pass `--index-only` to skip syncing and use the existing index as-is.

## Commands

| Command | Description |
|---|---|
| `vaultquery query "<DQL>"` | Execute a DQL query against the vault |
| `vaultquery index` | Build or update the vault index (incremental) |
| `vaultquery reindex` | Drop and rebuild the vault index from scratch |
| `vaultquery status` | Show index status (file count, vault path) |
| `vaultquery reference` | Print the full reference documentation to stdout |

### Options

| Flag | Applies to | Description | Default |
|---|---|---|---|
| `--vault <PATH>` | all | Vault root directory | current directory |
| `--format <FORMAT>` | query | Output format: `json` or `toon` | from config or `json` |
| `-v, --verbose` | all | Show detailed progress during indexing | off |
| `--index-only` | query | Skip index sync, use existing index as-is | off |

## Built-in reference

vaultquery ships with a comprehensive reference document embedded in the binary. This reference covers the full DQL syntax, all built-in functions, `file.*` metadata fields, shell quoting rules, and usage examples.

The reference is designed as a machine-readable context file for AI coding assistants. Instead of each assistant session interpreting the tool's behavior on its own, every instance gets the exact same canonical documentation — matched to the installed version.

```bash
# Print reference to stdout
vaultquery reference

# Create a local reference file (e.g. for Claude Code)
vaultquery reference > ~/.claude/references/vaultquery.md

# Update after upgrading vaultquery
brew upgrade vaultquery && vaultquery reference > ~/.claude/references/vaultquery.md
```

The reference is rebuilt with each release, so re-running the command after an upgrade keeps your local copy in sync.

## Shell quoting

DQL uses double quotes for string values and folder paths. **Always use double-quoted shell strings with escaped inner quotes:**

```bash
# CORRECT — double quotes with escaped inner quotes
vaultquery query "TABLE customer WHERE type = \"System\"" --vault ~/base

# CORRECT — no inner quotes needed
vaultquery query "LIST FROM #project" --vault ~/base

# WRONG — single quotes break negation (!) in zsh (BANG_HIST)
vaultquery query 'TABLE x WHERE status != "done"'
```

## DQL overview

### Query types

| Type | Description |
|---|---|
| `TABLE field1, field2, ...` | Tabular output with specified frontmatter fields |
| `LIST [expression]` | List of file links with optional expression |
| `TASK` | Task items (`- [ ]` / `- [x]`) from file content |
| `CALENDAR <date-field>` | Calendar view grouped by date |

### FROM sources

```
FROM "folder/path"              # Folder
FROM "*/folder"                 # Wildcard: match folder under any prefix
FROM "Clients/*/Projects"       # Wildcard: match any intermediate path
FROM #tag                       # Tag
FROM [[link]]                   # Backlinks to page
FROM outgoing([[link]])         # Outgoing links from page
FROM #tag AND "folder"          # Boolean combination
FROM #tag OR #other-tag
FROM !#excluded                 # Negation
```

### WHERE operators

| Operator | Example |
|---|---|
| `=`, `!=` | `status = "active"`, `type != "archived"` |
| `<`, `>`, `<=`, `>=` | `rating >= 4` |
| `contains`, `!contains` | `tags contains "linux"` |
| `exists`, `!exists` | `kubectl_context exists` |
| `AND`, `OR`, `(...)` | `(type = "Server" OR type = "Cluster") AND status = "active"` |

### Clauses

```
SORT due ASC                    # Sort (ASC or DESC)
SORT status DESC, due ASC       # Multi-field sort
LIMIT 10                        # Limit results
GROUP BY status                 # Group results
FLATTEN tags                    # Flatten array fields into rows
WITHOUT ID                      # Omit file link column
```

### Expressions and aliases

```
TABLE status, due, file.name
TABLE (due - date(today)) AS "Days Left"
TABLE choice(status = "done", "Y", "N") AS Done
```

### file.* metadata fields

Every indexed file exposes metadata via `file.*`:

| Field | Description |
|---|---|
| `file.name` | Filename without extension |
| `file.folder` | Parent folder path |
| `file.path` | Full relative path |
| `file.link` | Link to the file |
| `file.size` | File size in bytes |
| `file.ctime` / `file.cday` | Creation time / date |
| `file.mtime` / `file.mday` | Modification time / date |
| `file.day` | Date parsed from filename (e.g. `2026-03-06.md`) |
| `file.tags` / `file.etags` | All tags / explicit tags |
| `file.inlinks` / `file.outlinks` | Incoming / outgoing links |
| `file.aliases` | Aliases from frontmatter |
| `file.frontmatter` | All frontmatter as object |

### Built-in functions

**Constructors:** `date()`, `dur()`, `number()`, `string()`, `link()`, `list()`, `object()`, `typeof()`

**Numeric:** `round()`, `floor()`, `ceil()`, `min()`, `max()`, `sum()`, `product()`, `average()`, `minby()`, `maxby()`

**Arrays:** `contains()`, `icontains()`, `econtains()`, `length()`, `sort()`, `reverse()`, `flat()`, `slice()`, `unique()`, `join()`, `nonnull()`, `all()`, `any()`, `none()`

**Strings:** `lower()`, `upper()`, `split()`, `replace()`, `startswith()`, `endswith()`, `substring()`, `truncate()`, `padleft()`, `padright()`, `regextest()`, `regexmatch()`, `regexreplace()`

**Utility:** `default()`, `choice()`, `dateformat()`, `durationformat()`, `striptime()`, `meta()`, `currencyformat()`

**Lambda support:** `all(list, (x) => cond)`, `any(list, (x) => cond)`, `none(list, (x) => cond)`

For full function signatures and detailed syntax, run `vaultquery reference`.

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

## Output formats

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
