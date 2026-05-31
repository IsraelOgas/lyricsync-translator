# Lyricsync Translator

Overlay sincronizado de letras traducidas sobre cualquier reproductor MPRIS (Spotify, YouTube Music, Brave, Chrome, etc.). Detecta automáticamente lo que estás escuchando, busca las letras, las traduce al español y las muestra en tiempo real.

## Quick path — Modo local

```bash
# 1. LibreTranslate (traducción)
docker run -ti --rm -p 5000:5000 libretranslate/libretranslate --load-only en,es

# 2. Backend Go
go build -o lyricsync ./cmd/lyricsync/
./lyricsync

# 3. Frontend (otra terminal)
cd web && pnpm install && pnpm dev
```

Abrí `http://localhost:5173`.

## Quick path — Docker (todo junto)

```bash
docker compose up -d
```

Abrí `http://localhost:8090` — el backend sirve el frontend.

**Requisito Ubuntu/Debian**: AppArmor bloquea D-Bus en contenedores. El `docker-compose.yml` ya incluye `apparmor:unconfined`. Si tu UID no es 1000, ajustalo en el `Dockerfile`.

## Arquitectura

```
┌─────────────┐     SSE      ┌──────────────┐
│  React 19   │◄─────────────│   Go 1.26    │
│  Vite 8     │              │   chi v5     │
│  :5173 dev  │              │   :8090      │
└─────────────┘              └──────┬───────┘
                                    │
                      ┌─────────────┼─────────────┐
                      │             │             │
                 playerctl      LRCLib      LibreTranslate
                  (MPRIS)     (letras)      (traducción)
```

| Capa | Tecnología | Rol |
|------|-----------|-----|
| Frontend | React 19, TypeScript 5.9, Vite 8 | UI con letras sincronizadas |
| Backend | Go 1.26, chi v5 | API REST + SSE, resolución de letras |
| Player | playerctl + MPRIS/D-Bus | Detección automática del reproductor |
| Letras | LRCLib API | Letras sincronizadas (LRC) y plain text |
| Traducción | LibreTranslate | Traducción EN→ES + romanización (JP/ZH/KO) |
| Cache | SQLite | Persistencia de canciones y traducciones |

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
  provider: "libretranslate"
  libretranslate:
    base_url: "http://127.0.0.1:5000"

cache:
  db_path: "~/.lyricsync/cache.db"
```

### Variables de entorno

| Variable | Default | Uso |
|----------|---------|-----|
| `LIBRETRANSLATE_URL` | `http://127.0.0.1:5000` | URL de LibreTranslate |
| `LYRIC_HOST` | `127.0.0.1` | Host del servidor |
| `LYRIC_PORT` | `8090` | Puerto del servidor |
| `LYRIC_DB_PATH` | `~/.lyricsync/cache.db` | Ruta de la DB |
| `WEB_DIR` | `web/dist` | Directorio de estáticos frontend |

## Endpoints

| Método | Ruta | Descripción |
|--------|------|-------------|
| GET | `/api/now-playing` | Track actual + estado + posición |
| GET | `/api/lyrics/stream` | SSE: track, letras, traducciones, posición |
| GET | `/api/songs/{hash}/lyrics` | Letras cacheadas por hash |
| POST | `/api/player/toggle` | Play/pause del reproductor |
| GET | `/api/config` | Configuración actual |
| PUT | `/api/config` | Actualizar configuración |

## Eventos SSE

| Tipo | Dirección | Contenido |
|------|-----------|-----------|
| `track` | servidor → cliente | Artista, título, álbum, duración |
| `status` | servidor → cliente | `playing`, `paused`, `stopped`, `no_player` |
| `position` | servidor → cliente | Posición en ms (cada 500ms) |
| `lyrics_loading` | servidor → cliente | Búsqueda de letras iniciada |
| `lyrics` | servidor → cliente | Letras + flag `translating` |
| `translations` | servidor → cliente | Traducciones completadas |

## Estructura del proyecto

```
lyricsync-translator/
├── cmd/lyricsync/          # Entry point Go
├── internal/
│   ├── api/                # HTTP server, SSE broker, handlers
│   ├── cache/              # SQLite store
│   ├── config/             # Config loading + env vars
│   ├── lyrics/             # LRCLib client, LRC parser, orchestrator
│   ├── player/             # playerctl wrapper, MPRIS tracker
│   └── translate/          # LibreTranslate client + romanizer
├── web/
│   └── src/
│       ├── components/     # LyricsViewer, NowPlayingBar
│       ├── hooks/          # useSSE
│       ├── App.tsx         # Estado global, handler de eventos
│       ├── main.tsx        # Entry point React
│       └── types.ts        # Tipos compartidos
├── Dockerfile
├── docker-compose.yml
└── config.yaml
```

## Features

- Detección automática de **cualquier reproductor MPRIS** (Spotify, Brave, Chrome, apps)
- Letras sincronizadas (LRC) con highlight en tiempo real
- Traducción EN→ES vía LibreTranslate
- Romanización de japonés, chino y coreano
- Pausa sincronizada letras + reproductor (playerctl play-pause)
- Indicador de carga mientras busca letras
- Indicador de progreso de traducción por línea
- SSE con replay de estado al reconectar (recarga de página)
- Cache SQLite de canciones y traducciones
- Docker: multi-stage build, LibreTranslate incluido

## Requisitos

- **Linux** con D-Bus session bus
- `playerctl` instalado (`sudo apt install playerctl`)
- Go 1.26+
- Node.js 22+ / pnpm
- LibreTranslate (Docker o instalación local)
- Un reproductor MPRIS (Spotify, navegador Chromium/Brave, YouTube Music app, etc.)
