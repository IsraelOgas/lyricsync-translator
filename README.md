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

El `config.yaml` en la raíz del repo es un **template sin secrets**. La app lo usa como fallback inicial.  
Las keys y configuraciones sensibles se persisten automáticamente en `~/.config/lyricsync/config.yaml`.

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
    api_key: ""                     # configurable desde el panel de Settings
    model: "deepseek-chat"

cache:
  db_path: "~/.lyricsync/cache.db"
```

### Configuración desde la UI

La API key de DeepSeek se puede configurar desde el panel de **Settings → DeepSeek API Key** sin reiniciar la app. La key se guarda en `~/.config/lyricsync/config.yaml` y se aplica en caliente (hot-reload del cliente).

### Variables de entorno

| Variable | Default | Uso |
|---|---|---|
| `LIBRETRANSLATE_URL` | `http://127.0.0.1:5000` | URL de LibreTranslate |
| `DEEPSEEK_API_KEY` | — | API key de DeepSeek (fallback si no hay en UI) |
| `LYRIC_HOST` | `127.0.0.1` | Host del servidor |
| `LYRIC_PORT` | `8090` | Puerto del servidor |
| `LYRIC_TARGET_LANG` | `es` | Idioma de traducción |
| `LYRIC_DB_PATH` | `~/.lyricsync/cache.db` | Ruta de la DB |

> **Nota de seguridad**: `GET /api/config` no expone las API keys reales — devuelve `••••••••`. Las keys nunca se escriben en el `config.yaml` del repo, solo en `~/.config/lyricsync/`.

## Endpoints

| Método | Ruta | Descripción |
|---|---|---|
| GET | `/api/now-playing` | Track actual + estado + posición |
| GET | `/api/lyrics/stream` | SSE: track, letras, traducciones, posición |
| POST | `/api/lyrics/retry` | Reintentar traducción del track actual |
| GET | `/api/songs` | Listar canciones guardadas (con búsqueda `?search=`) |
| GET | `/api/songs/{hash}/lyrics` | Letras cacheadas por hash |
| GET | `/api/songs/{hash}/offset` | Offset de sincronización |
| PUT | `/api/songs/{hash}/offset` | Actualizar offset |
| POST | `/api/player/toggle` | Play/pause del reproductor |
| POST | `/api/player/next` | Siguiente track |
| POST | `/api/player/previous` | Track anterior |
| POST | `/api/player/seek` | Seek a posición `{position_ms}` |
| GET | `/api/player/volume` | Volumen actual |
| POST | `/api/player/volume` | Ajustar volumen `{delta}` o `{absolute}` |
| GET | `/api/player/shuffle` | Estado de shuffle |
| POST | `/api/player/shuffle` | Toggle shuffle |
| GET | `/api/player/loop` | Estado de loop |
| POST | `/api/player/loop` | Ciclar loop (none → playlist → track) |
| GET | `/api/config` | Configuración actual (API keys sanitizadas) |
| PUT | `/api/config` | Actualizar `target_lang` |
| PUT | `/api/config/provider` | Actualizar API key de provider + hot-reload |

## Eventos SSE

| Tipo | Dirección | Contenido |
|---|---|---|
| `track` | servidor → cliente | Artista, título, álbum, duración, cover art |
| `status` | servidor → cliente | `playing`, `paused`, `stopped`, `no_player` |
| `position` | servidor → cliente | Posición en ms (cada 500ms) |
| `lyrics_loading` | servidor → cliente | Búsqueda de letras iniciada |
| `lyrics` | servidor → cliente | Letras + flag `translating` + `not_found` |
| `lyrics_error` | servidor → cliente | Error al cargar letras o traducir (`error`, `retry`) |
| `translations` | servidor → cliente | Traducciones completadas (merge con líneas existentes) |

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
│       ├── components/      # LyricsViewer, NowPlayingBar, PlayerBar, SettingsPanel, SavedSongsView, HelpDialog, ErrorBoundary
│       ├── hooks/           # useSSE, usePlayerState, useSettings, useCoverColor, useKeyboardShortcuts
│       ├── App.tsx          # Estado global, cinema mode, view transitions
│       ├── App.module.css   # Estilos del layout + cinema track info flotante
│       ├── main.tsx         # Entry point React
│       ├── api.ts           # Helper apiUrl() para CORS/Wails
│       └── types.ts         # Tipos compartidos + defaults de settings
├── openspec/                # Artefactos SDD (specs, changes)
├── Dockerfile
├── docker-compose.yml
└── config.yaml
```

## Features

- **App nativa**: empaquetado Wails v2, single binary, sin navegador
- **Cinema mode**: fullscreen nativo, oculta barras de UI, widget flotante con info del track, View Transitions API para animación suave
- **Panel de Settings**: fuente, tema (4 temas), colores, espaciado, idioma, offset de sync, API key de DeepSeek con toggle revelar/ocultar
- **Configuración de API key desde la UI**: hot-reload del cliente DeepSeek sin reiniciar, persistencia en `~/.config/lyricsync/config.yaml`
- Detección automática de **cualquier reproductor MPRIS** (Spotify, Brave, Chrome, apps)
- Letras sincronizadas (LRC) con highlight en tiempo real + click-to-seek
- Traducción EN→ES (LibreTranslate o DeepSeek) con romanización de japonés, chino y coreano
- **Romanización prominente**: cuando la canción tiene transliteración, se muestra más grande que el texto original
- **Toast de error**: notificación flotante con auto-dismiss cuando falla el provider de traducción + botón Retry
- **Retry inteligente**: reintenta traducciones vacías (API key mala → corregir → siguiente reproducción)
- Controles del reproductor: play/pause, next/prev, seek, shuffle, loop, volumen con mute
- Biblioteca de canciones guardadas con búsqueda
- Atajos de teclado (`?` para ayuda)
- SSE con replay de estado al reconectar + merge atómico de traducciones
- Cache SQLite de canciones y traducciones
- Persistencia de estado de ventana (posición, tamaño, fullscreen)
- Docker: multi-stage build, LibreTranslate incluido
