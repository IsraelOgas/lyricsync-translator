# embedded-frontend Specification

## Purpose

Asset embedding via `go:embed`, SPA routing, API base injection, dev/prod build pipeline with pnpm and Vite.

## Requirements

| # | Requirement | Strength | Summary |
|---|-------------|----------|---------|
| R1 | Asset Embedding | MUST | `//go:embed web/dist/*` → `embed.FS`; no disk access at runtime |
| R2 | SPA Routing | MUST | Serve `index.html` for unmatched paths; `fs.Sub()` for path resolution |
| R3 | API Base Injection | MUST | Inject `window.__API_BASE__` at build time from `config.yaml` port |
| R4 | Dev Workflow | SHALL | `wails dev` with Vite HMR proxying; no embedded assets in dev |

### R1: Asset Embedding

The system MUST embed `web/dist/*` via `//go:embed` and serve it through `embed.FS`.

#### Scenario: Production build

- GIVEN `pnpm build` has produced `web/dist/`
- WHEN the Go binary is compiled
- THEN assets are served from embedded FS without disk reads

#### Scenario: Missing dist

- GIVEN `web/dist/` is empty or absent
- WHEN `go build` runs
- THEN the build MUST fail with a clear error

### R2: SPA Routing

The system MUST serve `index.html` for any path not matching an embedded asset.

#### Scenario: Deep link refresh

- GIVEN the user is at `/settings`
- WHEN the page is refreshed
- THEN `index.html` loads and React Router renders `/settings`

#### Scenario: Asset resolution

- GIVEN `index.html` references `/assets/bundle.js`
- WHEN the browser requests `/assets/bundle.js`
- THEN the embedded asset is served with correct MIME type

### R3: API Base Injection

The system MUST inject `window.__API_BASE__` into the frontend at build time.

> **Design note**: The injection mechanism (e.g., Vite `define` constant, `index.html` script tag, Wails runtime call) is deferred to the design phase. The design MUST select and document the chosen approach.

#### Scenario: Default port

- GIVEN the chi server runs on port 8090
- WHEN the frontend loads
- THEN `window.__API_BASE__` equals `"http://127.0.0.1:8090"`

#### Scenario: Custom port

- GIVEN `config.yaml` sets `server.port: 3000`
- WHEN the app starts
- THEN `window.__API_BASE__` equals `"http://127.0.0.1:3000"`

### R4: Dev Workflow

The system SHALL support `wails dev` with Vite dev server proxying.

#### Scenario: HMR in dev

- GIVEN the developer runs `wails dev`
- WHEN a `.tsx` file is modified
- THEN Vite HMR updates the webview without full reload
