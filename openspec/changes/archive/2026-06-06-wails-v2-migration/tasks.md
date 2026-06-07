# Tasks: Wails v2 Desktop Migration

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | 250-320 |
| 400-line budget risk | Medium |
| Chained PRs recommended | No |
| Delivery strategy | no-commit |
| Chain strategy | N/A |

Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: pending
400-line budget risk: Medium

## Guard Contract: `window.runtime`

`window.runtime` exists only inside a Wails WebView — absent during browser `pnpm dev`.
Any code touching it MUST guard with optional chaining (`?.`) or `typeof` check.
Existing hooks in Phase 3 use `?.` guards. Future Runtime code must follow same pattern.

## Phase 1: Foundation

- [x] 1.1 Install Wails CLI: `go install github.com/wailsapp/wails/v2/cmd/wails@latest` → installed v2.12.0
- [x] 1.2 Add `github.com/wailsapp/wails/v2 v2.12.0` to `go.mod`; run `go mod tidy`
- [x] 1.3 Create `wails.json`: `frontend:dir:"web"`, `install:"pnpm install"`, `build:"pnpm build"`, `dev:watcher:"pnpm dev"`, `dev:serverUrl:"http://localhost:5173"`

## Phase 2: Backend Core

- [x] 2.1 Create `cmd/lyricsync/embed.go`: `//go:embed ../../web/dist/*` + `EmbeddedAssetsFS() fs.FS` helper using `fs.Sub()`
- [x] 2.2 Add `WindowConfig{X,Y,Width,Height,Fullscreen}` struct + `LoadWindowState() (*WindowConfig, error)` to `internal/config/config.go` — reads `~/.lyricsync/window-state.yaml` (yaml unmarshal)
- [x] 2.3 Add `SaveWindowState(w *WindowConfig) error` to `internal/config/config.go` — writes `~/.lyricsync/window-state.yaml` (yaml marshal, create dirs)
- [x] 2.4 Modify `internal/api/server.go`: remove `FileServer` block; `Start()` returns immediately (listen goroutine); add `spaHandler(assetsFS fs.FS, apiBase string) http.HandlerFunc` with string substitution for `{{.APIBase}}` in `index.html`; store `router` + `cancel` in Server struct
- [x] 2.5 Refactor `cmd/lyricsync/main.go`: `wails.Run()` with `OnStartup` (config.Load→cache→player→srv.Start→LoadWindowState), `OnShutdown` (drain→close DB), `OnBeforeClose` (SaveWindowState). Signal handling removed — Wails owns lifecycle.

## Phase 3: Frontend Wiring

- [x] 3.1 Modify `web/index.html`: add `<script>window.__API_BASE__="{{.APIBase}}";</script>` before `</head>`
- [x] 3.2 Update `web/src/types.ts`: declare `Window.__API_BASE__` and `Window.runtime` globals
- [x] 3.3 Update `web/src/api.ts`: prepend `window.__API_BASE__` to all `fetch()` URLs
- [x] 3.4 Update `web/src/hooks/useSSE.ts`: prepend `window.__API_BASE__` to `EventSource` URL
- [x] 3.5 Update `web/src/hooks/usePlayerState.ts`: prepend `window.__API_BASE__` to `fetch()` calls
- [x] 3.6 Update `web/src/hooks/useSettings.ts`: in `applySettings()`, replace `data-cinema` attribute toggle with guarded `window.runtime?.WindowFullscreen()` / `WindowUnfullscreen()`; prepend `__API_BASE__` to config fetches
- [x] 3.7 Update `web/src/hooks/useKeyboardShortcuts.ts`: add Escape `keydown` fallback — if `cinemaMode` active, call `window.runtime?.WindowUnfullscreen()`; sync via `fullscreenchange` event. "C" shortcut unchanged — stays via settings→applySettings flow.
- [x] 3.8 Verify `web/vite.config.ts`: proxy port 5173→8090 already correct — no changes needed

## Phase 4: Build & Verification

- [x] 4.1 Run `pnpm build` in `web/` → produce `web/dist/` ✅ (1.17KB HTML, 224KB JS, 20KB CSS)
- [x] 4.2 Run `go build ./cmd/lyricsync/` — verify `//go:embed` compiles with embedded assets ✅ (16.7MB binary)
- [ ] 4.3 Run `wails build` — verify single binary per platform ⚠️ Wails CLI runs but fails with `fork/exec /tmp/wailsbindings: permission denied` (system /tmp noexec mount, not a code issue)
- [ ] 4.4 Smoke test: API endpoints + SSE, cinema mode (fullscreen+Escape exit), window state persists, graceful close on window exit (requires wails build to succeed first)
