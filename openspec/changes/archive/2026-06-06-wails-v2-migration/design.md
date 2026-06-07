# Design: Wails v2 Desktop Migration

## Technical Approach

Wrap the existing chi HTTP server inside `wails.Run()` lifecycle. The chi router serves BOTH API routes and embedded frontend assets via custom `AssetServer.Handler`. Zero Wails Go↔JS bindings — the HTTP API stays unchanged. Native `WindowFullscreen()` replaces CSS cinema toggle. Window state persists to `config.yaml` `window:` section on close.

## Architecture Decisions

| Decision | Options considered | Choice | Rationale |
|----------|-------------------|--------|-----------|
| **API base injection** | A) Vite `define` (build-time), B) Go template in `index.html`, C) Wails runtime JS→Go | **B — Go template** | Port is runtime-configurable (`config.yaml`). Build-time constant breaks custom port. Wails bindings out of scope. Go handler substitutes `{{.APIBase}}` placeholder in `index.html` before serving. |
| **Asset serving** | A) Wails default `Assets` embed, B) Custom `http.Handler` via `AssetServer.Handler` | **B — Custom handler** | Chi must own all routing (API + SPA). Wails default asset server can't inject API base into HTML. Custom handler = chi router handles everything. |
| **go:embed location** | A) `cmd/lyricsync/embed.go` (`../../web/dist/*`), B) `internal/assets/embed.go` | **A — cmd/lyricsync/** | Simpler: embed with `main` package. No new package for one directive. Path `../../web/dist/*` is explicit. |
| **Fullscreen trigger** | A) Only JS → Wails Runtime, B) Wails Go API from backend event | **A — JS → Wails Runtime, with Escape fallback** | Cinema toggle initiated by frontend (key `C` or settings). Call `window.runtime.WindowFullscreen()` / `WindowUnfullscreen()` directly. **Escape**: add JS `keydown` listener as explicit fallback calling `WindowUnfullscreen()` — Wails handles Escape natively on most platforms, but the JS fallback covers WebKitGTK (Linux) edge cases. Frontend syncs `cinemaMode` state via `fullscreenchange` event. |
| **Dev mode API access** | A) Relative paths only, B) Conditional `window.__API_BASE__` | **A — Relative paths (dev), absolute (prod)** | In `wails dev`, webview loads from Vite dev server (port 5173), proxy forwards `/api`. Relative paths work. `window.__API_BASE__` injected only in production builds. |
| **Window state persistence** | A) Wails `WindowGetPosition`/`WindowGetSize` on close, B) Event-based save | **A — `OnBeforeClose` callback, regex edit of `config.yaml`** | `OnBeforeClose` reads `config.yaml` as text, regex-injects/updates only the `window:` block, preserving all comments and key order. If regex proves fragile during implementation, **fallback to Option B**: write to separate file `~/.lyricsync/window-state.yaml`, leaving `config.yaml` untouched. |

## Data Flow

```
                          wails.Run()
                     ┌──────────────────┐
                     │  OnStartup()     │
                     │  ├─ config.Load  │
                     │  ├─ cache.New    │
                     │  ├─ player.New   │
                     │  ├─ srv := api.New│
                     │  ├─ go srv.Start │──► chi :8090 ── API routes
                     │  └─ restore win  │       │
                     └────────┬─────────┘       ├─ embedded FS handler
                              │                  │   (template inject index.html)
                     ┌────────▼─────────┐       │
    Webview ◄────────│  AssetServer     │◄──────┘
    (wails://)       │  Handler = chi   │
                     └──────────────────┘
                     ┌──────────────────┐
                     │  OnBeforeClose() │──► regex-edit config.yaml (window:)
                     │  OnShutdown()    │──► srv.Shutdown() → store.Close()
                     └──────────────────┘
```

**Production**: Webview → chi router → API routes OR embedded assets (with API base injected).
**Dev**: Webview → `localhost:5173` (Vite) → proxy `/api` → chi `127.0.0.1:8090`.

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `cmd/lyricsync/main.go` | Modify | Replace `http.ListenAndServe` + signal handling with `wails.Run()` lifecycle. Init in `OnStartup`, graceful shutdown in `OnShutdown`. |
| `cmd/lyricsync/embed.go` | **Create** | `//go:embed ../../web/dist/*` + `EmbeddedAssets() http.FileSystem` helper using `fs.Sub()` |
| `internal/api/server.go` | Modify | Remove `http.FileServer` block (lines 114-124). `Start()` returns immediately (listen in goroutine). Add `ServeEmbedded(embed.FS)` method for SPA handler with template injection. |
| `wails.json` | **Create** | `frontend:dir: "web"`, `frontend:install: "pnpm install"`, `frontend:build: "pnpm build"`, `frontend:dev:watcher: "pnpm dev"`, `frontend:dev:serverUrl: "http://localhost:5173"` |
| `web/index.html` | Modify | Add `<script>window.__API_BASE__ = "{{.APIBase}}";</script>` placeholder before `</head>` |
| `web/vite.config.ts` | Modify | No structural changes. Ensure proxy target matches dev server port. |
| `web/src/types.ts` | Modify | Add `declare global { interface Window { __API_BASE__: string; runtime?: { WindowFullscreen: () => void; WindowUnfullscreen: () => void } } }` |
| `web/src/hooks/useSettings.ts` | Modify | `applySettings()`: replace `data-cinema` attribute toggle with `window.runtime.WindowFullscreen()` / `WindowUnfullscreen()` call |
| `web/src/hooks/useKeyboardShortcuts.ts` | Modify | Escape key handler: sync `cinemaMode` to `false` on fullscreen exit |
| `web/src/hooks/useSSE.ts` | Modify | `EventSource` URL: prepend `window.__API_BASE__` |
| `web/src/api.ts` | Modify | `fetch` URL: prepend `window.__API_BASE__` |
| `web/src/hooks/usePlayerState.ts` | Modify | All `fetch` calls: prepend `window.__API_BASE__` |
| `web/src/hooks/useSettings.ts` | Modify | `fetch('/api/config',...)` → prepend `window.__API_BASE__` |
| `config.yaml` | Modify | `window:` section added on first close (regex edit preserves existing comments/key order) |
| `internal/config/config.go` | Modify | Add `WindowConfig` struct. `Load()` adds `Window` field; regex-based `SaveWindow()` writes only the `window:` block to `config.yaml`. Fallback: `~/.lyricsync/window-state.yaml`. |
| `go.mod` | Modify | Add `github.com/wailsapp/wails/v2 v2.10.0` |

## Interfaces / Contracts

**WindowConfig struct** (`internal/config/config.go`):
```go
type WindowConfig struct {
    X          int  `yaml:"x"`
    Y          int  `yaml:"y"`
    Width      int  `yaml:"width"`
    Height     int  `yaml:"height"`
    Fullscreen bool `yaml:"fullscreen"`
}
```

**Config preservation** (`internal/config/config.go`):
```go
// SaveWindowState regex-edits only the window: block into config.yaml text.
// Preserves all comments, blank lines, and key ordering outside window:.
// Falls back to ~/.lyricsync/window-state.yaml if regex proves fragile.
func SaveWindowState(cfgPath string, w WindowConfig) error
```

**SPA handler contract**: `func (s *Server) spaHandler(assets http.FileSystem, apiBase string) http.HandlerFunc` — serves `index.html` with API base injected for unmatched paths; otherwise serves embedded file directly.

**Escape fallback** (`web/src/hooks/useKeyboardShortcuts.ts`): `keydown` listener for Escape — if `cinemaMode` is active, call `window.runtime.WindowUnfullscreen()` explicitly. Runs alongside Wails native Escape handling (belt and suspenders).

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Build | Binary compiles on Linux, macOS, Windows | `wails build` per platform; CI matrix (manual) |
| Smoke | App launches, API responds, SSE streams | `wails dev` manual smoke test |
| Build-time | `go build` fails if `web/dist/` missing | Verify `//go:embed` compile error on empty dir |

No unit/integration tests exist today. This change adds no new testable Go interfaces — only lifecycle wiring. Build verification across platforms is the minimum viable gate.

## Migration / Rollout

No data migration. Existing `config.yaml` gains `window:` section on first close. Rollback: remove `wails.json`, revert `main.go` to standalone HTTP server, restore `FileServer` block, `go mod tidy`.

## Resolved During Design

- **Escape key in fullscreen**: JS `keydown` fallback explicitly calls `WindowUnfullscreen()`. Runs alongside Wails native handling — covers WebKitGTK edge cases.
- **config.yaml preservation**: Regex minimal approach — inject/edit only `window:` block into `config.yaml` text, preserving comments. Fallback: separate `~/.lyricsync/window-state.yaml` file if regex proves fragile.

## Open Questions

None — all design decisions resolved.
