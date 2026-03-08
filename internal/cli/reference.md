# vaultquery Reference

Queries Obsidian vault `.md` files by YAML frontmatter using Dataview Query Language (DQL).
Indexes into SQLite, queries with a hybrid SQL + Go evaluator.

## Invocation

**Always use the `query` subcommand.** The DQL string is the first positional argument:

```bash
vaultquery query "<DQL>" --vault ~/base --format toon
```

Sync happens automatically before each query. Use `--index-only` to skip sync (e.g. multiple queries in quick succession). There is no `--sync` flag.

### Options

| Flag | Description | Default |
|---|---|---|
| `--vault <PATH>` | Vault root directory | current directory |
| `--format <FORMAT>` | Output format: `toon` or `json` | from config or `json` |
| `-v` | Verbose (show indexing progress) | off |
| `--index-only` | Skip sync, use existing index | off |

**Recommended:** Use `--format toon` for structured output optimized for both humans and LLMs (see [TOON format](https://github.com/toon-format/toon)).

### Shell Quoting

DQL queries contain double quotes (for strings and folder paths). **Always use double-quoted shell strings with escaped inner quotes:**

```bash
# CORRECT — double quotes with escaped inner quotes
vaultquery query "TABLE customer WHERE type = \"System\"" --vault ~/base --format toon

# CORRECT — no inner quotes needed
vaultquery query "LIST FROM #project" --vault ~/base --format toon

# WRONG — single quotes break with negation (!) in zsh (BANG_HIST)
vaultquery query 'TABLE x WHERE status != "done"'

# WRONG — missing query subcommand
vaultquery "TABLE customer WHERE type = \"System\""

# WRONG — --sync does not exist
vaultquery query "..." --vault ~/base --sync
```

### Storage & Config

- Index: `.vaultquery/index.db` inside vault root
- Config: `.vaultquery/config.yaml` with `format:` (default output format) and `exclude:` (folder exclusion list)

## Subcommands

```bash
vaultquery query "<DQL>" --vault ~/base     # Execute DQL query
vaultquery index --vault ~/base             # Build/update index (incremental)
vaultquery reindex --vault ~/base           # Drop & rebuild from scratch
vaultquery status --vault ~/base            # File count, last update, vault path
vaultquery reference                        # Print this reference to stdout
```

## DQL Syntax

### Query Types

- `TABLE [fields] FROM <source> [WHERE ...] [SORT ...] [LIMIT n]`
- `LIST [expression] FROM <source> [WHERE ...] [SORT ...] [LIMIT n]`
- `TASK FROM <source> [WHERE ...]`
- `CALENDAR <date-field> FROM <source> [WHERE ...] [SORT ...]`

### FROM Sources

```
FROM "folder/path"                    # Folder
FROM #tag                             # Tag
FROM [[link]]                         # Backlinks to page
FROM outgoing([[link]])               # Outgoing links from page
FROM #tag AND "folder"                # Boolean combination
FROM #tag OR #other-tag
FROM !#excluded                       # Negation
```

### WHERE Clauses

```
WHERE status = "active"
WHERE due < date(today)
WHERE contains(tags, "important")
WHERE field != null                   # Field exists
WHERE rating >= 4 AND status = "done"
WHERE contains(file.name, "2026")
```

### Fields & Expressions

```
TABLE status, due, file.name                        # Multiple fields
TABLE (due - date(today)) AS "Days Left"            # Computed + alias
TABLE file.folder, file.tags, file.inlinks          # file.* metadata
TABLE WITHOUT ID status FROM #project               # Omit file link column
TABLE choice(status = "done", "Y", "N") AS Done     # Function in field
```

### file.* Metadata Fields

```
file.name          # Filename without extension
file.folder        # Parent folder path
file.path          # Full relative path
file.ext           # File extension (e.g. ".md")
file.link          # Link to the file
file.size          # File size in bytes
file.ctime         # Creation time
file.cday          # Creation date (day only)
file.mtime         # Modification time
file.mday          # Modification date (day only)
file.day           # Date parsed from filename (e.g. 2026-03-06.md)
file.tags          # All tags (frontmatter + inline)
file.etags         # Explicit tags
file.inlinks       # Incoming links (backlinks)
file.outlinks      # Outgoing links
file.aliases       # Aliases from frontmatter
file.frontmatter   # All frontmatter as object
```

### SORT, LIMIT, GROUP BY, FLATTEN

```
SORT due ASC
SORT status DESC, due ASC
LIMIT 10
GROUP BY status
FLATTEN tags
```

### Built-in Functions

**Constructors**: `date(today)`, `date("2026-03-06")`, `dur("1d")`, `number(val)`, `string(val)`, `link(path)`, `list(...)`, `object("k1", v1, ...)`, `typeof(val)`

**Numeric**: `round(n, decimals)`, `floor(n)`, `ceil(n)`, `min(a, b)`, `max(a, b)`, `sum(list)`, `product(list)`, `average(list)`, `minby(list, fn)`, `maxby(list, fn)`

**Arrays/Lists**: `contains(a, b)`, `icontains(a, b)`, `econtains(a, b)`, `length(list)`, `sort(list)`, `reverse(list)`, `flat(list)`, `slice(list, start, end)`, `unique(list)`, `join(list, sep)`, `nonnull(list)`, `all(list)`, `any(list)`, `none(list)`

**Higher-order (lambda)**: `all(list, (x) => cond)`, `any(list, (x) => cond)`, `none(list, (x) => cond)`

**Strings**: `lower(s)`, `upper(s)`, `split(s, sep)`, `replace(s, old, new)`, `startswith(s, prefix)`, `endswith(s, suffix)`, `substring(s, start, end)`, `truncate(s, len, suffix)`, `padleft(s, width, char)`, `padright(s, width, char)`, `regextest(pattern, s)`, `regexmatch(pattern, s)`, `regexreplace(s, pattern, repl)`

**Utility**: `default(field, fallback)`, `choice(cond, ifTrue, ifFalse)`, `dateformat(date, "yyyy-MM-dd")`, `durationformat(dur)`, `striptime(date)`, `meta(link)`, `currencyformat(n, "$")`

## Examples

```bash
# All Kubernetes clusters
vaultquery query "TABLE customer, kubectl_context WHERE type = \"Kubernetes Cluster\"" --vault ~/base --format toon

# All servers with health-check infrastructure
vaultquery query "TABLE server_id, ssh_host, customer WHERE type = \"System\" AND server_id != \"\"" --vault ~/base --format toon

# All initiatives for a customer
vaultquery query "TABLE project FROM \"Kunden und Organisationen\" WHERE type = \"Initiative\" AND customer = \"Memodo\"" --vault ~/base --format toon

# Journal entries for this month
vaultquery query "LIST FROM \"Journal/2026/03\" SORT file.name DESC" --vault ~/base --format toon

# Skip sync for rapid successive queries
vaultquery query "LIST FROM #project" --vault ~/base --format toon --index-only

# Update your local reference file after upgrading
vaultquery reference > ~/.claude/references/vaultquery.md
```
