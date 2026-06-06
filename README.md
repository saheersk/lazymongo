# lazymongo

A fast, keyboard-driven terminal UI for MongoDB — inspired by lazygit and lazydocker.

```
┌─ Databases ──────────────┐┌─ mydb > users ───────────────────────────────────────┐
│ mydb                     ││ _id                       name           email        │
│   users                  ││▶ 507f1f77bcf86cd799439011  Alice Smith    alice@ex.com │
│   orders                 ││  507f1f77bcf86cd799439012  Bob Jones      bob@ex.com   │
│   products               ││  507f1f77bcf86cd799439013  Carol White    carol@ex.com │
│ analytics                ││                                                        │
│   events                 ││  n new  e edit  d del  / filter  s sort  a agg  I idx │
│   sessions               │└────────────────────────────────────────────────────────┘
│                          │┌─ Document ─────────────────────────────────────────────┐
│                          ││ {                                                      │
│                          ││   "_id": ObjectId("507f1f77bcf86cd799439011"),         │
│                          ││   "name": "Alice Smith",                               │
│                          ││   "email": "alice@example.com",                        │
│                          ││   "createdAt": ISODate("2024-05-24T10:00:00Z")         │
│                          ││ }                                                      │
└──────────────────────────┘└────────────────────────────────────────────────────────┘
 mongodb://localhost:27017 │ mydb > users │ 1–50 of 2 841 │ page 1/57
```

## Features

- **Browse** databases and collections with keyboard navigation
- **View** documents in a paginated table with a syntax-highlighted detail panel
- **Filter** using any MongoDB query expression (`{"status": "active", "age": {"$gt": 18}}`)
- **Sort** by field name, `-field` for descending, or a JSON sort document
- **Insert / edit / delete** documents in your `$EDITOR`
- **Aggregate** — open a pipeline editor (`a`), run it, see results inline with an `[AGG]` badge
- **Indexes** — list indexes with keys and flags (`I`), create new ones, drop existing ones
- **Copy** `_id` or full document JSON to clipboard (`y` / `Y`)
- **Responsive** layout from 80 columns upward, full mouse support

## Install

### go install

```bash
go install github.com/saheersk/lazymongo@latest
```

Requires Go 1.21+. The binary lands in `$(go env GOPATH)/bin`.

### Build from source

```bash
git clone https://github.com/saheersk/lazymongo
cd lazymongo
go build -o lazymongo .
```

## Quick start

```bash
# Connect to local MongoDB (default: mongodb://localhost:27017)
lazymongo

# Explicit URI
lazymongo --uri "mongodb://localhost:27017"

# Atlas or remote cluster
lazymongo --uri "mongodb+srv://user:pass@cluster.mongodb.net"

# Host and port separately
lazymongo --host 192.168.1.10 --port 27017
```

## Configuration

On first run, lazymongo writes `~/.config/lazymongo/config.yaml`:

```yaml
connections:
  - name: local
    uri: mongodb://localhost:27017
    default: true

ui:
  theme: dark       # only theme currently supported
  mouse: true       # mouse cell-motion events
  pageSize: 50      # documents per page
```

You can add multiple connections; the one with `default: true` is used when no `--uri` flag is given.

Any key can be overridden with a `LAZYMONGO_` environment variable:

```bash
LAZYMONGO_UI_PAGESIZE=100 lazymongo
```

## Keyboard reference

### Global

| Key | Action |
|-----|--------|
| `h` / `←` | Focus left panel |
| `l` / `→` | Focus right panel |
| `esc` | Close panel / go back |
| `q` / `Ctrl+C` | Quit |

### Sidebar

| Key | Action |
|-----|--------|
| `j` / `↓` | Move down |
| `k` / `↑` | Move up |
| `Enter` | Expand database / select collection |
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
| `e` | Edit selected document (`$EDITOR`) |
| `d` | Delete selected document (`y` to confirm) |
| `/` | Filter — any MongoDB query JSON |
| `s` | Sort — `field`, `-field`, or `{"field": -1}` |
| `r` | Reset filter and sort |
| `a` | Open aggregate pipeline editor |
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

### Aggregate mode

Press `a` from anywhere while a collection is loaded. Your `$EDITOR` opens with a template:

```json
[
  { "$match": {} }
]
```

Save and close to run the pipeline. Results appear in the document list tagged `[AGG]`.

| Key | Action |
|-----|--------|
| `a` | Re-open editor (pipeline is pre-filled with last run) |
| `esc` | Exit aggregate mode, return to live collection |

Pipelines without a `$limit`, `$out`, or `$merge` stage automatically get `{ "$limit": 1000 }` appended to prevent runaway scans.

### Index panel (`I`)

| Key | Action |
|-----|--------|
| `j` / `k` | Navigate indexes |
| `g` / `G` | First / last index |
| `n` | Create index (`$EDITOR` opens with template) |
| `d` | Drop selected index (`y` to confirm) |
| `R` | Refresh list |
| `esc` / `h` | Close panel |

Index creation template:

```json
{
  "keys": { "fieldName": 1 },
  "unique": false,
  "sparse": false
}
```

Use `1` / `-1` for ascending / descending. Use `"text"` for full-text indexes. Set `"unique": true` or `"sparse": true` as needed.

### Detail panel

| Key | Action |
|-----|--------|
| `j` / `↓` | Scroll down |
| `k` / `↑` | Scroll up |
| `esc` / `h` | Close |

## Editor integration

lazymongo opens documents and pipelines in your `$EDITOR` (falling back to `$VISUAL`, then `vi`). Multi-word editor commands are supported:

```yaml
ui:
  editor: "code --wait"   # VS Code
  editor: "nvim"
  editor: "nano"
```

Files are created as `/tmp/lazymongo-*.json` in MongoDB Extended JSON format. Save and close to apply; delete all content or quit without saving to cancel.

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

Requires the MongoDB Go driver v2. The `primitive` package is not used; BSON types are accessed directly via `bson.ObjectID`, `bson.A`, `bson.M`, `bson.D`.

## Development

```bash
# Run all tests
go test ./...

# Integration tests (requires MongoDB on localhost:27017)
go test ./internal/mongo/... -v

# Build binary
go build -o lazymongo .
```

Tests use the `lazymongo_test` database and clean up after themselves.

## Project layout

```
.
├── main.go
├── cmd/root.go              # cobra CLI, flag parsing, config loading
├── internal/
│   ├── config/              # viper-backed YAML config
│   ├── mongo/               # MongoDB client, CRUD, aggregate, indexes
│   └── tui/
│       ├── app.go           # root bubbletea model, message routing
│       ├── msg/             # shared message types (no import cycles)
│       ├── keymap/          # all key bindings in one place
│       ├── style/           # lipgloss theme
│       └── panels/
│           ├── sidebar/     # database + collection tree
│           ├── documents/   # paginated document table + filter/sort
│           ├── detail/      # single-document viewer
│           ├── indexes/     # index list, create, drop
│           └── statusbar/   # bottom status line
└── internal/util/           # BSON↔JSON helpers, clipboard, syntax highlight
```

## Contributing

Bug reports and pull requests are welcome. Please open an issue first for significant changes.

## License

MIT — see [LICENSE](LICENSE).
