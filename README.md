# Lyricsync Translator

Overlay sincronizado de letras traducidas sobre cualquier reproductor MPRIS (Spotify, YouTube Music, Brave, Chrome, etc.). Detecta automáticamente lo que estás escuchando, busca las letras, las traduce al español y las muestra en tiempo real.

Ahora como **app de escritorio nativa** con Wails v2 — un solo binario, sin abrir el navegador.

## Requisitos

| Dependencia | Paquete | Obligatorio | Notas |
|---|---|---|---|
| Go | `snap install go --classic` | ✅ | 1.26+ |
| Node.js + pnpm | `nodejs`, `pnpm` | ✅ | 22+ |
| `playerctl` | `sudo apt install playerctl` | ✅ | Detección MPRIS |
| GTK 3 dev | `sudo apt install libgtk-3-dev` | ✅ | Wails — UI nativa |
| WebKit2 GTK dev | `sudo apt install libwebkit2gtk-4.1-dev` | ✅ | Wails — motor web |
| LibreTranslate | Docker `libretranslate/libretranslate` | ⬜ | Solo si usás `libretranslate` como provider |

### Instalación rápida de dependencias (Ubuntu/Debian)

```bash
sudo apt install -y playerctl libgtk-3-dev libwebkit2gtk-4.1-dev
snap install go --classic
```

## Build — App de escritorio (Wails)

```bash
# Si Go se instaló con snap, ~/go/bin puede no estar en el PATH:
export PATH="$HOME/go/bin:$PATH"

# Si /tmp tiene noexec (común en entornos con security hardening):
mkdir -p ~/tmp/wails
TMPDIR=~/tmp/wails wails build
```

El binario queda en `bin/lyricsync`.

### Tags de build requeridos

El `wails.json` ya incluye los tags necesarios para Ubuntu 24.04:

```json
"build:tags": "webkit2_41"
```

Si tu distro usa `webkit2gtk-4.0` (Ubuntu 22.04, Debian 12), cambiá el tag a `webkit2_40` o eliminalo.

### Problemas frecuentes de build

| Error | Causa | Solución |
|---|---|---|
| `fork/exec wailsbindings: permission denied` | Falta `package main` en raíz, o `/tmp` con `noexec` | Asegurate de que `main.go` esté en la raíz del proyecto y usá `TMPDIR` alternativo |
| `open wailsjs/runtime/package.json: permission denied` | `web/wailsjs/` pertenece a root (por `sudo wails build` previo) | `sudo chown -R $USER:$USER web/wailsjs/` |
| `webkit2gtk-4.0 was not found` | Ubuntu 24.04 usa 4.1, no 4.0 | Agregá `build:tags: webkit2_41` en `wails.json` |
| `libwebkit2gtk-4.1-dev` no encontrado | Falta el paquete dev | `sudo apt install libwebkit2gtk-4.1-dev` |
| `fatal error: gtk/gtk.h: No such file` | Falta GTK dev | `sudo apt install libgtk-3-dev` |

## Build — Solo backend (desarrollo rápido)

```bash
go build .
./lyricsync-translator
```

## Dev mode (frontend + backend)

```bash
# Terminal 1 — LibreTranslate (opcional)
docker run -ti --rm -p 5000:5000 libretranslate/libretranslate --load-only en,es

# Terminal 2 — Backend
go build . && ./lyricsync-translator

# Terminal 3 — Frontend (Vite HMR)
cd web && pnpm install && pnpm dev
```

Abrí `http://localhost:5173`.

## Docker (todo junto)

```bash
docker compose up -d
```

Abrí `http://localhost:8090`.

> **Ubuntu/Debian**: AppArmor bloquea D-Bus en contenedores. El `docker-compose.yml` ya incluye `apparmor:unconfined`. Si tu UID no es 1000, ajustalo en el `Dockerfile`.

## Arquitectura

```
┌──────────────────────────────────────┐
│           Wails Desktop              │
│  ┌─────────────┐     ┌────────────┐  │
│  │  React 19   │ SSE │  Go 1.26   │  │
│  │  Vite 8     │◄────│  chi v5    │  │
│  │  WebView    │     │  API+SPA   │  │
│  └─────────────┘     └─────┬──────┘  │
└─────────────────────────────┼────────┘
                              │
                ┌─────────────┼─────────────┐
                │             │             │
           playerctl      LRCLib      LibreTranslate
            (MPRIS)      (letras)      (traducción)
```

| Capa | Tecnología | Rol |
|---|---|---|
| Desktop | Wails v2 + WebKit2GTK | Ventana nativa, cinema mode, empaquetado |
| Frontend | React 19, TypeScript 5.9, Vite 8 | UI con letras sincronizadas |
| Backend | Go 1.26, chi v5 | API REST + SSE, resolución de letras |
| Player | playerctl + MPRIS/D-Bus | Detección automática del reproductor |
| Letras | LRCLib API | Letras sincronizadas (LRC) y plain text |
| Traducción | LibreTranslate o DeepSeek | Traducción EN→ES + romanización (JP/ZH/KO) |
| Cache | SQLite (modernc) | Persistencia de canciones y traducciones |

## Configuración

Creá `config.yaml` (opcional, usa defaults si no existe):

```yaml
server:
  host: "127.0.0.1"
  port: 8090

player:
  playerctl_path: "playerctl"

lyrics:
  provider: "lrclib"

translation:
  provider: "libretranslate"       # o "deepseek"
  target_lang: "es"
  libretranslate:
    base_url: "http://127.0.0.1:5000"
  deepseek:
    api_key: "${DEEPSEEK_API_KEY}"
    model: "deepseek-chat"

cache:
  db_path: "~/.lyricsync/cache.db"
```

### Variables de entorno

| Variable | Default | Uso |
|---|---|---|
| `LIBRETRANSLATE_URL` | `http://127.0.0.1:5000` | URL de LibreTranslate |
| `DEEPSEEK_API_KEY` | — | API key de DeepSeek |
| `LYRIC_HOST` | `127.0.0.1` | Host del servidor |
| `LYRIC_PORT` | `8090` | Puerto del servidor |
| `LYRIC_DB_PATH` | `~/.lyricsync/cache.db` | Ruta de la DB |

## Endpoints

| Método | Ruta | Descripción |
|---|---|---|
| GET | `/api/now-playing` | Track actual + estado + posición |
| GET | `/api/lyrics/stream` | SSE: track, letras, traducciones, posición |
| GET | `/api/songs` | Listar canciones guardadas (con búsqueda `?q=`) |
| GET | `/api/songs/{hash}/lyrics` | Letras cacheadas por hash |
| POST | `/api/player/toggle` | Play/pause del reproductor |
| GET | `/api/config` | Configuración actual |
| PUT | `/api/config` | Actualizar configuración |

## Eventos SSE

| Tipo | Dirección | Contenido |
|---|---|---|
| `track` | servidor → cliente | Artista, título, álbum, duración |
| `status` | servidor → cliente | `playing`, `paused`, `stopped`, `no_player` |
| `position` | servidor → cliente | Posición en ms (cada 500ms) |
| `lyrics_loading` | servidor → cliente | Búsqueda de letras iniciada |
| `lyrics` | servidor → cliente | Letras + flag `translating` |
| `translations` | servidor → cliente | Traducciones completadas |

## Estructura del proyecto

```
lyricsync-translator/
├── main.go                  # Entry point (Wails + chi + server)
├── assets.go                # go:embed del frontend compilado
├── wails.json               # Configuración de Wails v2
├── internal/
│   ├── api/                 # HTTP server, SSE broker, handlers
│   ├── cache/               # SQLite store
│   ├── config/              # Config loading + window state
│   ├── lyrics/              # LRCLib client, LRC parser, orchestrator
│   ├── player/              # playerctl wrapper, MPRIS tracker
│   └── translate/           # LibreTranslate + DeepSeek clients, romanizer
├── web/
│   └── src/
│       ├── components/      # LyricsViewer, NowPlayingBar, PlayerBar, SavedSongsView
│       ├── hooks/           # useSSE, usePlayerState, useSettings, useKeyboardShortcuts
│       ├── App.tsx          # Estado global, handler de eventos
│       ├── main.tsx         # Entry point React
│       └── types.ts         # Tipos compartidos
├── openspec/                # Artefactos SDD (specs, changes)
├── Dockerfile
├── docker-compose.yml
└── config.yaml
```

## Features

- **App nativa**: empaquetado Wails v2, single binary, sin navegador
- **Cinema mode**: fullscreen nativo con overlay de letras
- Detección automática de **cualquier reproductor MPRIS** (Spotify, Brave, Chrome, apps)
- Letras sincronizadas (LRC) con highlight en tiempo real
- Traducción EN→ES (LibreTranslate o DeepSeek)
- Romanización de japonés, chino y coreano
- Biblioteca de canciones guardadas con búsqueda
- Pausa sincronizada letras + reproductor
- SSE con replay de estado al reconectar
- Cache SQLite de canciones y traducciones
- Persistencia de estado de ventana (posición, tamaño, fullscreen)
- Docker: multi-stage build, LibreTranslate incluido
