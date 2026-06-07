# wails-desktop-app Specification

## Purpose

Desktop lifecycle, window management, fullscreen, and state persistence. Cross-platform: Linux, macOS, Windows.

## Requirements

| # | Requirement | Strength | Summary |
|---|-------------|----------|---------|
| R1 | App Lifecycle | MUST | Chi server starts in `OnStartup`, graceful shutdown (DB close, drain requests) on window close |
| R2 | Window Management | MUST | Fixed title "LyricSync", `WindowClose()`, native `WindowFullscreen()` for cinema mode |
| R3 | Window State Persistence | MUST | Save position, size, and fullscreen state to `config.yaml` `window:` on close; restore on launch |
| R4 | Cross-Platform | SHALL | Identical behavior on Linux, macOS, Windows; quirks documented |

### R1: App Lifecycle

The system MUST start the chi HTTP server on `127.0.0.1:{port}` during Wails `OnStartup` and gracefully shut it down on window close.

#### Scenario: Graceful shutdown

- GIVEN the app is running with active SSE connections
- WHEN the user closes the window
- THEN the server drains in-flight requests, closes the SQLite DB, and cancels goroutines before exit

### R2: Window Management

The system MUST set window title to "LyricSync" and support native fullscreen toggle via `WindowFullscreen()`.

#### Scenario: Cinema mode enter

- GIVEN the app window is in normal mode
- WHEN the user activates cinema mode
- THEN the window enters native fullscreen with no decorations

#### Scenario: Cinema mode exit

- GIVEN the app is in native fullscreen
- WHEN the user presses Escape
- THEN the window restores to previous position and size

### R3: Window State Persistence

The system MUST persist window position (x,y), size (width,height), and fullscreen state to `config.yaml` `window:` on close and restore all three on next launch.

#### Scenario: Restore saved state

- GIVEN `config.yaml` has `window: {x: 100, y: 200, width: 1024, height: 768, fullscreen: false}`
- WHEN the app starts
- THEN the window appears at (100,200) with 1024×768 in normal mode

#### Scenario: Restore fullscreen on launch

- GIVEN the user closed the app while in cinema mode
- AND `config.yaml` `window:` has `fullscreen: true`
- WHEN the app starts
- THEN the window opens in native fullscreen immediately

#### Scenario: First launch defaults

- GIVEN `config.yaml` has no `window:` section
- WHEN the app starts for the first time
- THEN the window SHALL use sensible defaults (centered, 1024×768, not fullscreen)

### R4: Cross-Platform

The system SHALL build and run identically on Linux, macOS, and Windows.

#### Scenario: Cinema mode across OSes

- GIVEN binaries for Linux, macOS, and Windows
- WHEN cinema mode is toggled on each platform
- THEN all three enter/exit native fullscreen with identical frontend behavior
