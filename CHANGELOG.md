# Changelog

All notable changes to lazymongo are documented here.

---

## [1.2.5] — 2026-06-07

### Added
- **Watch mode** (`W`) — live change-stream overlay showing INSERT / UPDATE / REPLACE / DELETE events in real time (requires a MongoDB replica set)
- **Connection switch** (`P`) — pick any saved profile from a picker overlay without restarting the app
- **Connection health** — background ping every 15 s; status bar shows latency (`◆ 2ms`) or offline indicator (`◇`)
- **Self-update** (`lazymongo --update`) — fetches the latest release from GitHub, downloads the correct OS/arch binary, and replaces itself in place
- **Multi-select** (`space`) — toggle row selection; `D` bulk-deletes all selected documents with confirmation
- **Filter history** — `↑` / `↓` in the filter bar to recall previous queries
- **Clone document** (`c`) — duplicate a document with `_id` stripped, opens in `$EDITOR`
- **Explain plan** (`E`) — overlay showing winning plan (COLLSCAN vs IXSCAN), index used, docs/keys examined, execution time
- **Schema inference** (`S`) — samples up to 100 documents and shows per-field type breakdown with presence percentage
- **Import** (`i`) — bulk-insert from `.json` (array), `.jsonl`, `.ndjson`, `.csv`; file-path input with tab completion and `~/` expansion
- **Export** (`x`) — export query results to JSON or CSV
- **Delete confirmation for all destructive operations** — single delete, bulk delete, drop collection, drop database all require `y` confirmation

### Changed
- Full README rewrite with complete keyboard reference and feature docs
- Help overlay updated with all new key bindings

### Fixed
- `space` key was incorrectly opening documents instead of toggling selection
- CI: removed ineffectual assignment lint warning (Windows build)
- CI: tilde tab-completion test now skips on Windows (Unix-only feature)

---

## [1.2.1] — 2026-05-XX

### Added
- **5 built-in themes** — `catppuccin`, `high-contrast`, `tokyo-night`, `nord`, `dracula`; cycle with `T`
- **Named connection profiles** — save and load connections by name (`--save`, `--profile`)
- **Dynamic sidebar** — databases and collections update without restart
- **Drop database** — two-step confirmation from the sidebar
- **Sidebar search** — `/` in sidebar to filter databases and collections live
- **Transparent help overlay** — `?` shows all key bindings without leaving the current view
- **Pinned status and hint bars** — always visible at the bottom regardless of scroll position
- **Export picker** — choose format (JSON / CSV) before exporting
- **Collection stats** — document count shown next to collection name in sidebar
- **Create collection** from sidebar (`c`)

---

## [1.1.2] — 2026-04-XX

### Fixed
- Homebrew formula step syntax for GoReleaser v2

---

## [1.1.1] — 2026-04-XX

### Added
- Install scripts for macOS and Linux (`scripts/install_update_darwin.sh`, `scripts/install_update_linux.sh`)

### Fixed
- Homebrew tap configuration for GoReleaser v2.16

---

## [1.1.0] — 2026-04-XX

### Added
- One-liner install commands in README
- Homebrew tap support (`brew tap saheersk/tap && brew install lazymongo`)

### Fixed
- golangci-lint warnings resolved
- `go.mod` minimum version lowered to Go 1.24 for CI compatibility

---

## [1.0.0] — 2026-03-XX

### Added
- **Aggregate pipeline editor** (`a`) — opens `$EDITOR` with a template; results shown inline tagged `[AGG]`; auto-appends `$limit: 1000` for safety
- **Index panel** (`I`) — list indexes with keys and flags; create (`n`) and drop (`d`) indexes
- **CRUD operations** — insert (`n`), edit (`e`), delete (`d`) documents via `$EDITOR`
- Full test suite for CRUD, filtering, sorting, pagination

---

## [0.1.0] — 2026-02-XX

### Added
- Initial release
- Sidebar tree — browse databases and collections
- Paginated document table with column headers
- Syntax-highlighted detail panel
- Filter with any MongoDB query expression
- Sort by field name or JSON sort document
- Copy `_id` or full document JSON to clipboard (`y` / `Y`)
- Mouse support
- Responsive layout from 80 columns upward
