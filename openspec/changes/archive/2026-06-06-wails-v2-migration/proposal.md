# Proposal: Wails v2 Desktop Migration

## Intent

Ship a self-contained native desktop app instead of requiring users to open `http://127.0.0.1:8090` in a browser. Single binary per platform with window management, fullscreen, and embedded assets. All existing features preserved.

## Scope

### In Scope
- Wails v2 integration (`wails.json`, `wails dev`/`wails build`)
- `go:embed` assets replacing `http.FileServer` disk serving
- Native window: title, close, fullscreen (`WindowFullscreen()`), position/size persistence
- Keep `web/` directory (configure `frontend:dir: "web"`)
- `cmd/lyricsync/main.go` → Wails lifecycle
- Cross-platform builds: Linux, macOS, Windows with documented deps

### Out of Scope
- Wails v3, Go↔JS bindings for API (future: MAY migrate SSE → Wails Events)
- Notifications, tray icons, system menus, auto-updater
- OS packaging (`.deb`, `.AppImage`, `.dmg`)

## Capabilities

### New Capabilities
- `wails-desktop-app`: Lifecycle, window management, fullscreen, window-state persistence
- `embedded-frontend`: `go:embed` asset serving replacing disk-based `FileServer`

### Modified Capabilities
None — no baseline specs in `openspec/specs/`.

## Approach

**HTTP API**: Chi router runs on `127.0.0.1:8090` (configurable via `server.port`). Frontend injected with `window.__API_BASE__`. Zero Wails bindings for v1.

**Lifecycle**: `wails.Run()` replaces `http.ListenAndServe()`. Server starts in `OnStartup`, shuts down in `OnShutdown`. Window position/size persisted to `config.yaml` `window:` section on close.

**Assets**: `embed.FS` replaces `http.FileServer`. Vite outputs to `web/dist/` → `//go:embed web/dist/*`. SPA routing via `fs.Sub()`.

**Fullscreen**: `WindowFullscreen()` replaces CSS toggle.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `cmd/lyricsync/main.go` | Modified | Wails lifecycle replaces raw HTTP server |
| `internal/api/server.go` | Modified | Drop `FileServer`; expose callbacks |
| `wails.json` | New | Frontend dir, build/install commands |
| `web/src/`, `web/index.html` | Modified | `window.__API_BASE__`; fullscreen via Wails Runtime |
| `web/vite.config.ts` | Modified | Base path for embedded serving |
| `go.mod` | Modified | Add `github.com/wailsapp/wails/v2` |
| `config.yaml` | Modified | New `window:` section |
| `internal/` (all pkgs), `web/` dir | Unchanged | All existing code preserved |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| SSE fails in webview | Low | EventSource supported; test day one |
| Cross-platform WebView quirks | Med | Test on all three OSes; document |
| `embed.FS` path vs `http.Dir` mismatch | Low | `fs.Sub()` for correct root |
| `window.__API_BASE__` unset | Med | Inject via Vite `define` + `index.html` |
| playerctl/D-Bus in Wails context | Low | Same process, no sandbox |

## Rollback Plan

1. Remove `wails.json`; revert `main.go` to standalone HTTP server
2. Restore `http.FileServer` in `server.go`
3. Remove Wails from `go.mod`; `go mod tidy`
4. `pnpm build` — app works as web-only

## Dependencies

| Platform | Build | Runtime |
|----------|-------|---------|
| Linux | `libgtk-3-dev`, `libwebkit2gtk-4.1-dev` | `libgtk-3-0`, `libwebkit2gtk-4.1-0` |
| macOS | Xcode CLI tools | None |
| Windows | GCC (MSYS2/TDM-GCC) | WebView2 (preinstalled Win 10+) |

- Go: `github.com/wailsapp/wails/v2 v2.10.0`
- CLI: `go install github.com/wailsapp/wails/v2/cmd/wails@latest`

## Success Criteria

- [ ] `wails dev` works on Linux, macOS, Windows (Go + Vite hot-reload)
- [ ] `wails build` produces single binary per platform with embedded assets
- [ ] All 14 API endpoints + SSE streaming work in webview
- [ ] Cinema mode = native `WindowFullscreen()` (no chrome)
- [ ] Window position/size persist across restarts
- [ ] Player controls, settings, help dialog, saved songs unchanged
- [ ] Window close → graceful shutdown (DB, goroutines)
