# lazymongo

[![CI](https://github.com/saheersk/lazymongo/actions/workflows/ci.yml/badge.svg)](https://github.com/saheersk/lazymongo/actions/workflows/ci.yml)
[![Release](https://github.com/saheersk/lazymongo/actions/workflows/release.yml/badge.svg)](https://github.com/saheersk/lazymongo/actions/workflows/release.yml)

A fast, keyboard-driven terminal UI for MongoDB — inspired by lazygit and lazydocker.

![lazymongo screenshot](assets/screenshot.png)

---

## Features

### Navigation & browsing
- **Sidebar tree** — databases expand to show collections; `j`/`k` to move, `Enter` to select, `/` to search/filter the list
- **Document table** — paginated table view with column headers auto-built from the first page of results
- **Detail panel** — syntax-highlighted JSON viewer for the selected document with scroll support
- **Responsive layout** — adapts from 80 columns upward; full mouse support

### Documents
- **Insert** a new document in your `$EDITOR` (`n`)
- **Edit** the selected document in your `$EDITOR` (`e`)
- **Clone** a document (strips `_id`, opens editor) (`c`)
- **Delete** a single document with confirmation (`d` → `y`)
- **Multi-select** rows with `space`, bulk-delete with `D` → `y`

### Querying
- **Filter** with any MongoDB query expression — `{"status":"active","age":{"$gt":18}}` (`/`)
- **Filter history** — `↑`/`↓` in the filter bar to recall previous filters
- **Sort** by field name, `-field` for descending, or a full sort doc — `{"field":-1}` (`s`)
- **Reset** filter + sort in one keystroke (`r`)
- **Aggregate** — open a pipeline editor, run it, see results tagged `[AGG]` (`a`)
- **Explain plan** — see COLLSCAN/IXSCAN, index used, docs/keys examined, execution time (`E`)

### Schema & data tools
- **Schema inference** — samples up to 100 docs and shows per-field type breakdown with presence % (`S`)
- **Import** — bulk-insert from `.json` (array), `.jsonl`, `.ndjson`, or `.csv` with tab-completion for file paths (`i`)
- **Export** — export query results to JSON/CSV (`x`)
- **Copy** `_id` or full document JSON to clipboard (`y` / `Y`)

### Indexes
- **List** all indexes with keys, flags and stats (`I`)
- **Create** an index from a JSON template in `$EDITOR` (`n` inside index panel)
- **Drop** a selected index with confirmation (`d` inside index panel)

### Collections & databases
- **Create collection** directly from the sidebar (`c`)
- **Drop collection** with two-step confirmation (`D` on a collection in sidebar)
- **Drop database** with two-step confirmation (`D` on a database in sidebar)

### Live & connection features
- **Watch mode** — press `W` on a loaded collection to open a live change-stream overlay; INSERT/UPDATE/REPLACE/DELETE events appear in real time (requires a replica set)
- **Connection health** — periodic ping every 15 s; status bar shows latency (`◆ 2ms`) or offline indicator (`◇`) 
- **Connection switch** — press `P` to pick any saved profile without restarting (`P`)

### UI & themes
- **5 built-in themes** — `catppuccin`, `high-contrast`, `tokyo-night`, `nord`, `dracula`; cycle with `T`
- **Help overlay** — `?` shows a full keybinding reference at any time

---

## Install

### macOS

```bash
curl https://raw.githubusercontent.com/saheersk/lazymongo/main/scripts/install_update_darwin.sh | bash
```

Or manually:

```bash
# Apple Silicon (M1 / M2 / M3)
curl -fsSL https://github.com/saheersk/lazymongo/releases/latest/download/lazymongo_darwin_arm64.tar.gz | tar xz && sudo mv lazymongo /usr/local/bin/

# Intel
curl -fsSL https://github.com/saheersk/lazymongo/releases/latest/download/lazymongo_darwin_amd64.tar.gz | tar xz && sudo mv lazymongo /usr/local/bin/
```

Or with Homebrew:

```bash
brew tap saheersk/tap
brew install lazymongo
```

---

### Linux

```bash
curl https://raw.githubusercontent.com/saheersk/lazymongo/main/scripts/install_update_linux.sh | bash
```

Or manually:

```bash
# amd64
curl -fsSL https://github.com/saheersk/lazymongo/releases/latest/download/lazymongo_linux_amd64.tar.gz | tar xz && sudo mv lazymongo /usr/local/bin/

# arm64 (Raspberry Pi, AWS Graviton)
curl -fsSL https://github.com/saheersk/lazymongo/releases/latest/download/lazymongo_linux_arm64.tar.gz | tar xz && sudo mv lazymongo /usr/local/bin/
```

---

### Windows

```powershell
Invoke-WebRequest https://github.com/saheersk/lazymongo/releases/latest/download/lazymongo_windows_amd64.zip -OutFile lazymongo.zip
Expand-Archive lazymongo.zip -DestinationPath "$HOME\bin"
```

Add `$HOME\bin` to your `PATH` via **System Properties → Environment Variables → Path → New**.

---

### Go

```bash
go install github.com/saheersk/lazymongo@latest
```

Requires Go 1.21+. Binary lands in `$(go env GOPATH)/bin`.

---

### Build from source

```bash
git clone https://github.com/saheersk/lazymongo
cd lazymongo
go build -o lazymongo .
```

---

## Quick start

```bash
# Local MongoDB (default: mongodb://localhost:27017)
lazymongo

# Explicit URI
lazymongo --uri "mongodb://localhost:27017"

# Atlas / remote cluster
lazymongo --uri "mongodb+srv://user:pass@cluster.mongodb.net"

# Host and port separately
lazymongo --host 192.168.1.10 --port 27017

# Named profile shorthand
lazymongo local
```

---

## Configuration

On first run, lazymongo writes `~/.config/lazymongo/config.yaml`:

```yaml
connections:
  - name: local
    uri: mongodb://localhost:27017
    default: true
    theme: catppuccin   # per-profile theme override

ui:
  theme: catppuccin     # catppuccin | high-contrast | tokyo-night | nord | dracula
  mouse: true
  pageSize: 50
  editor: ""            # leave empty to use $EDITOR / $VISUAL / vim
```

### Named profiles

Save a connection and give it a name:

```bash
# Save profiles
lazymongo --uri mongodb://localhost:27017 --save local
lazymongo --uri "mongodb+srv://user:pass@cluster.mongodb.net" --save atlas

# Connect by name
lazymongo local
lazymongo --profile atlas
```

When more than one profile exists and none is specified, a picker appears on launch. Inside the app, press `P` at any time to switch profiles without restarting.

### Themes

Cycle through all themes with `T`, or set one per profile:

```yaml
connections:
  - name: production
    uri: mongodb+srv://...
    theme: high-contrast
  - name: local
    uri: mongodb://localhost:27017
    theme: catppuccin
```

Available themes: `catppuccin` · `high-contrast` · `tokyo-night` · `nord` · `dracula`

---

## Keyboard reference

### Global

| Key | Action |
|-----|--------|
| `h` / `←` | Focus sidebar |
| `l` / `→` | Focus documents |
| `?` | Toggle help overlay |
| `T` | Cycle theme |
| `P` | Switch connection profile |
| `esc` | Close overlay / go back |
| `q` / `Ctrl+C` | Quit |

### Sidebar

| Key | Action |
|-----|--------|
| `j` / `↓` | Move down |
| `k` / `↑` | Move up |
| `Enter` | Expand database / select collection |
| `/` | Search / filter sidebar list |
| `c` | Create collection |
| `D` | Drop collection or database (2-step confirm) |
| `R` | Refresh list |

### Document list

| Key | Action |
|-----|--------|
| `j` / `↓` | Next row |
| `k` / `↑` | Previous row |
| `g` | First row |
| `G` | Last row |
| `Ctrl+D` | Next page |
| `Ctrl+U` | Previous page |
| `Enter` | Open detail panel |
| `n` | New document (`$EDITOR`) |
| `e` | Edit document (`$EDITOR`) |
| `c` | Clone document (`$EDITOR`, `_id` stripped) |
| `d` | Delete document (`y` to confirm) |
| `space` | Toggle row selection (multi-select) |
| `D` | Bulk-delete selected rows (`y` to confirm) |
| `/` | Filter — any MongoDB query JSON |
| `↑` / `↓` | (in filter bar) Browse filter history |
| `s` | Sort — `field`, `-field`, or `{"field": -1}` |
| `r` | Reset filter and sort |
| `a` | Aggregate pipeline editor |
| `E` | Explain plan overlay |
| `S` | Schema inference overlay |
| `i` | Import from file (JSON / JSONL / CSV) |
| `x` | Export results |
| `W` | Watch collection — live change stream |
| `I` | Toggle index panel |
| `y` | Copy `_id` to clipboard |
| `Y` | Copy full document JSON to clipboard |
| `R` | Refresh current page |

### Filter / sort bar

| Key | Action |
|-----|--------|
| `Enter` | Apply |
| `Esc` | Cancel |
| `Ctrl+U` | Clear input |
| `↑` / `↓` | Browse filter history |

### Aggregate mode

Press `a` to open your `$EDITOR` with a pipeline template:

```json
[
  { "$match": {} }
]
```

Save and close to run. Results appear tagged `[AGG]`.

| Key | Action |
|-----|--------|
| `a` | Re-open editor (last pipeline pre-filled) |
| `esc` | Exit aggregate mode, return to live view |

Pipelines without `$limit`, `$out`, or `$merge` automatically get `{"$limit": 1000}` appended.

### Index panel (`I`)

| Key | Action |
|-----|--------|
| `j` / `k` | Navigate indexes |
| `g` / `G` | First / last |
| `n` | Create index (`$EDITOR` opens with template) |
| `d` | Drop selected index (`y` to confirm) |
| `R` | Refresh |
| `esc` / `h` | Close panel |

Index creation template:

```json
{
  "keys": { "fieldName": 1 },
  "unique": false,
  "sparse": false
}
```

Use `1` / `-1` for ascending/descending, `"text"` for full-text indexes.

### Explain plan overlay (`E`)

Shows the winning plan for the current query:

- **IXSCAN** — index name, keys examined, selectivity
- **COLLSCAN** — warning for missing index
- Execution time and docs returned

Press any key to close.

### Schema overlay (`S`)

Samples up to 100 documents and shows:

- Every field found, sorted by frequency
- BSON type breakdown (string, int32, objectId, …)
- Presence percentage

`j` / `k` to scroll, any other key to close.

### Import overlay (`i`)

| Key | Action |
|-----|--------|
| `Tab` | Autocomplete file path (shell-style, `~/` supported) |
| `Enter` | Run import |
| `Esc` | Cancel |

Supported formats: `.json` (array) · `.jsonl` · `.ndjson` · `.csv`

Inserts in batches of 500. Duplicate-key errors are skipped and counted; the rest still insert.

### Watch overlay (`W`)

Requires a MongoDB replica set (standalone instances don't support change streams).

| Key | Action |
|-----|--------|
| `j` / `k` | Scroll event list |
| `W` / `esc` | Stop watching and close |

Events show operation type (`INSERT` / `UPDATE` / `REPLACE` / `DELETE`), document ID, a field preview, and a relative timestamp. Newest events appear at the top; up to 100 events are kept in the buffer.

### Connection picker (`P`)

| Key | Action |
|-----|--------|
| `j` / `k` | Navigate profiles |
| `Enter` | Connect to selected profile |
| `Esc` | Cancel |

Selecting a profile disconnects the current client and reconnects without restarting the app. The sidebar and document list reset automatically.

### Detail panel

| Key | Action |
|-----|--------|
| `j` / `↓` | Scroll down |
| `k` / `↑` | Scroll up |
| `esc` / `h` | Close |

---

## Editor integration

lazymongo opens documents and pipelines in `$EDITOR` (fallback: `$VISUAL`, then `vim`). Multi-word commands work:

```yaml
ui:
  editor: "code --wait"
  # editor: "nvim"
  # editor: "nano"
```

Temp files are created as `/tmp/lazymongo-*.json` in MongoDB Extended JSON format. Save and close to apply; delete all content or quit without saving to cancel.

---

## Connection health

A background ping runs every 15 seconds:

- `◆ localhost:27017  2ms` — connected, latency shown
- `◇ localhost:27017` — connection lost

On reconnect the indicator returns to `◆` automatically.

---

## Watch mode

Watch mode uses MongoDB [change streams](https://www.mongodb.com/docs/manual/changeStreams/) and requires a **replica set** (or Atlas). A standalone `mongod` will show an error immediately.

To start a local single-node replica set for testing:

```bash
mongod --replSet rs0 --dbpath /tmp/rs0 --port 27017 --fork --logpath /tmp/rs0.log
mongosh --eval "rs.initiate()"
```

Then press `W` on any collection to start watching.

---

## Compatibility

| MongoDB | Status |
|---------|--------|
| 4.x | Supported |
| 5.x | Supported |
| 6.x | Supported |
| 7.x | Supported |
| Atlas | Supported |

| Platform | Status |
|----------|--------|
| macOS | Tested |
| Linux | Tested |
| Windows (WSL) | Tested |

Requires MongoDB Go driver v2.

---

## Development

```bash
# Run all tests
go test ./...

# Integration tests (requires MongoDB on localhost:27017)
go test ./internal/mongo/... -v

# Build
go build -o lazymongo .
```

Tests use the `lazymongo_test` database and clean up after themselves.

---

## Project layout

```
.
├── main.go
├── cmd/
│   ├── root.go          # cobra CLI, flag parsing, config loading
│   └── picker.go        # startup profile picker
├── internal/
│   ├── config/          # viper-backed YAML config, named profiles
│   ├── mongo/           # MongoDB client, CRUD, aggregate, indexes,
│   │                    # explain, schema, import, health, watch
│   ├── util/            # BSON↔JSON, clipboard, syntax highlight, export, import parse
│   └── tui/
│       ├── app.go       # root bubbletea model, message routing, overlays
│       ├── msg/         # shared message types (no import cycles)
│       ├── keymap/      # all key bindings
│       ├── style/       # lipgloss themes (5 built-in)
│       └── panels/
│           ├── sidebar/    # database + collection tree, search, create/drop
│           ├── documents/  # paginated table, filter/sort/agg/multi-select
│           ├── detail/     # single-document JSON viewer
│           ├── indexes/    # index list, create, drop
│           └── statusbar/  # bottom status line, health indicator
└── assets/
    └── screenshot.png
```

---

## Contributing

Bug reports and pull requests are welcome. Please open an issue first for significant changes.

---

## License

MIT — see [LICENSE](LICENSE).
