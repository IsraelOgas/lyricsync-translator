# Lyricsync Translator

Overlay sincronizado de letras traducidas sobre cualquier reproductor MPRIS (Spotify, YouTube Music, Brave, Chrome, etc.). Detecta automáticamente lo que estás escuchando, busca las letras, las traduce al español y las muestra en tiempo real.

Ahora como **app de escritorio nativa** con Wails v2 — un solo binario, sin abrir el navegador.

## Requisitos

| Dependencia | Paquete | Obligatorio | Notas |
|---|---|---|---|
| Go | `snap install go --classic` | ✅ | 1.26+ (probado con 1.27.2) |
| Node.js + pnpm | `nodejs`, `pnpm` | ✅ | 22+ |
| `playerctl` | `sudo apt install playerctl` | ✅ | Detección MPRIS |
| GTK 3 dev | `sudo apt install libgtk-3-dev` | ✅ | Wails — UI nativa |
| WebKit2 GTK dev | `sudo apt install libwebkit2gtk-4.1-dev` | ✅ | Wails — motor web |
| Wails CLI | `go install github.com/wailsapp/wails/v2/cmd/wails@v2.16.0` | ✅ | v2.12 no compila con Go 1.27 (ver Problemas frecuentes) |
| LibreTranslate | Docker `libretranslate/libretranslate` | ⬜ | Solo si usás `libretranslate` como provider |

### Instalación rápida de dependencias (Ubuntu/Debian)

```bash
sudo apt install -y playerctl libgtk-3-dev libwebkit2gtk-4.1-dev
snap install go --classic
```

### Instalación desde cero (paso a paso)

1. Dependencias del sistema (apt):

   ```bash
   sudo apt install -y playerctl libgtk-3-dev libwebkit2gtk-4.1-dev
   ```

2. Go (el proyecto usa `go 1.26.3` en `go.mod`; si ya tenés Go de tu distro, salta este paso):

   ```bash
   snap install go --classic
   ```

3. PATH de binarios de Go:

   ```bash
   export PATH="$HOME/go/bin:$PATH"
   ```

4. Instalar la CLI de Wails:

   ```bash
   go install github.com/wailsapp/wails/v2/cmd/wails@v2.16.0
   ```

5. Verificar el entorno (opcional pero recomendado):

   ```bash
   wails doctor
   ```

   Muestra versión de Wails, Go, OS, GPU y si encuentra los paquetes de GTK/WebKit.

6. Compilar:

   ```bash
   mkdir -p ~/tmp/wails
   TMPDIR=~/tmp/wails wails build
   ```

7. Ejecutar:

   ```bash
   ./bin/lyricsync
   ```

Ojo: tener instalada solo la librería runtime de GTK/WebKit (`libgtk-3-0`, `libwebkit2gtk-4.1-0`) permite ejecutar un binario ya compilado, pero no compilar: para `wails build` hacen falta los paquetes `-dev`.

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
| `internal error: package "fmt" without types was imported from ...` | CLI de Wails ≤ v2.12 usa `golang.org/x/tools` viejo y no puede leer los datos de exportación de Go 1.27; `-skipbindings` no lo evita | `go install github.com/wailsapp/wails/v2/cmd/wails@v2.16.0` |
| `Package 'gtk+-3.0' not found` / `Package 'webkit2gtk-4.1' not found` durante la compilación CGO | Faltan los headers de desarrollo (la librería runtime no alcanza) | `sudo apt install libgtk-3-dev libwebkit2gtk-4.1-dev` |

### Verificar el build

Binario enlazado contra las libs correctas:

```bash
ldd bin/lyricsync | grep -E 'webkit2gtk|gtk-3'
```

Debe listar `libwebkit2gtk-4.1.so.0` y `libgtk-3.so.0`.

Al arrancar, la app imprime en consola:

```text
LyricsSync Translator starting on 127.0.0.1:8090
Database ready at ~/.lyricsync/cache.db
Server listening on 127.0.0.1:8090
App started — launching server
```

API viva:

```bash
curl -s http://127.0.0.1:8090/api/now-playing
```

Cierre limpio: cerrar con la X de la ventana debe terminar el proceso al instante, imprimir `Shutting down...` y `Shutdown complete`, y `pgrep -af 'playerctl --follow'` no debe devolver nada (el follower no debe quedar huérfano).

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

## Testing

```bash
go test ./...          # tests unitarios + de integración (sin red)
go test -short ./...   # salta los tests que llaman APIs externas
```

- `internal/api` — `inflightSet` (dedupe de resoluciones en vuelo), apagado (broker SSE y `Shutdown` con un stream SSE abierto) y el estado de fullscreen reportado por la UI (`GET`/`PUT /api/window/fullscreen`).
- `internal/player` — ciclo de vida de `playerctl --follow`: al cancelar el contexto el proceso hijo muere y los canales se cierran.
- `internal/lyrics` — fallback chain (error vs. not-found, orden de providers) y cliente lrcmux.
- El test `TestLrcMux_RealAPI` hace llamadas reales a `https://api.lrcmux.dev` y se omite con `-short`.

Recomendado: `go test ./... -race` — los tests de concurrencia y apagado se escribieron para pasar con el detector de carreras.

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
            (MPRIS)     lrcmux       (traducción)
                        (letras)
```

> **Letras**: LRCLib y lrcmux se pueden combinar en cadena (`lyrics.fallback`). lrcmux agrega varios orígenes (LRCLIB, Kugou, NetEase…) y devuelve timestamps a nivel de palabra + cover art.

| Capa | Tecnología | Rol |
|---|---|---|
| Desktop | Wails v2 + WebKit2GTK | Ventana nativa, cinema mode, empaquetado |
| Frontend | React 19, TypeScript 5.9, Vite 8 | UI con letras sincronizadas |
| Backend | Go 1.26, chi v5 | API REST + SSE, resolución de letras |
| Player | playerctl + MPRIS/D-Bus | Detección automática del reproductor |
| Letras | LRCLib API + lrcmux (fallback en cadena) | Letras sincronizadas (LRC), timestamps word-level y plain text |
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
  provider: "lrclib"            # "lrclib" o "lrcmux" (agregador con timestamps word-level + cover art)
  fallback: []                  # providers de respaldo en orden (ver abajo)
  level: "line"                 # "line" (default) o "word" (karaoke palabra por palabra)
  lrclib:
    base_url: "https://lrclib.net/api"
    timeout_sec: 15
  lrcmux:
    base_url: "https://api.lrcmux.dev"
    timeout_sec: 15

translation:
  provider: "libretranslate"       # o "deepseek"
  enabled: true                    # master switch: apagado = cero llamadas a API
  target_lang: "es"
  libretranslate:
    base_url: "http://127.0.0.1:5000"
  deepseek:
    api_key: ""                     # configurable desde el panel de Settings
    model: "deepseek-chat"

cache:
  db_path: "~/.lyricsync/cache.db"
```

Podés encadenar providers en orden con `lyrics.fallback`: si el principal falla o no encuentra la letra, se prueba el siguiente de la lista. Por ejemplo, `lrcmux` como principal con `lrclib` de respaldo:

```yaml
lyrics:
  provider: "lrcmux"
  fallback: ["lrclib"]
```

### Configuración desde la UI

La API key de DeepSeek se puede configurar desde el panel de **Settings → DeepSeek API Key** sin reiniciar la app. La key se guarda en `~/.config/lyricsync/config.yaml` y se aplica en caliente (hot-reload del cliente).

El toggle **Translate Lyrics** (Settings, o atajo `T`) activa/desactiva toda la tubería de traducción. Cuando está apagado no se hace NINGUNA llamada a DeepSeek/LibreTranslate (costo cero), y al encenderlo se re-resuelve el track actual para arrancar las traducciones que faltan. El estado se persiste en `~/.config/lyricsync/config.yaml` y sobrevive reinicios.

El toggle **Karaoke** (Settings) activa/desactiva el karaoke completo. Cuando está apagado (default) la app pide `level=line`, que lrcmux cachea y no consume rate-limit; al encenderlo pide `level=word` y pinta palabra por palabra si el provider trae timestamps word-level, o cae al fill por línea si no (p. ej. LRCLIB nunca los tiene).

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
| PUT | `/api/config` | Actualizar `target_lang` y/o `translation_enabled` |
| PUT | `/api/config/provider` | Actualizar API key de provider + hot-reload |
| GET | `/api/window/fullscreen` | Último estado de fullscreen reportado por la UI |
| PUT | `/api/window/fullscreen` | Reportar fullscreen `{fullscreen}`; el cierre de ventana guarda este valor |

## Eventos SSE

| Tipo | Dirección | Contenido |
|---|---|---|
| `track` | servidor → cliente | Artista, título, álbum, duración, cover art |
| `status` | servidor → cliente | `playing`, `paused`, `stopped`, `no_player` |
| `position` | servidor → cliente | Posición en ms (cada 500ms) |
| `lyrics_loading` | servidor → cliente | Búsqueda de letras iniciada |
| `lyrics` | servidor → cliente | Letras + flag `translating` + `not_found`; `song.source` (badge de origen), `cover_art`, `isrc` y `end_ms` por línea se incluyen cuando el provider los trae |
| `lyrics_error` | servidor → cliente | Error al cargar letras o traducir (`error`, `retry`) |
| `translations` | servidor → cliente | Traducciones completadas (merge con líneas existentes) |

> **Badge de origen**: el frontend muestra de qué provider/origen salieron las letras (ej. `LRCLIB`, `lrcmux · Kugou`) en la barra Now Playing, el viewer y el modo cinema.

## Estructura del proyecto

```
lyricsync-translator/
├── main.go                  # Entry point (Wails + chi + server)
├── assets.go                # go:embed del frontend compilado
├── wails.json               # Configuración de Wails v2
├── internal/
│   ├── api/                 # HTTP server, SSE broker, handlers, inflight dedupe
│   ├── cache/               # SQLite store (cover art + word-level timestamps)
│   ├── config/              # Config loading + window state
│   ├── lyrics/              # Providers (LRCLib, lrcmux), fallback chain, LRC parser, orchestrator
│   ├── player/              # playerctl wrapper, MPRIS tracker
│   └── translate/           # LibreTranslate + DeepSeek clients, romanizer, master toggle
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

## Cierre de la app (shutdown)

Cuando cerrás la ventana con la X, la app no se va de golpe: hace un apagado ordenado antes de terminar el proceso.

- Se detiene el broker SSE y se cierran los streams abiertos, para que el servidor HTTP pueda drenar; el drenaje tiene timeout de 5 s y después fuerza `Close()`.
- Se cierra la base SQLite.
- Se mata el proceso hijo `playerctl --follow` (cancelando su contexto), así que no quedan procesos huérfanos.
- Se guarda posición, tamaño y fullscreen de la ventana en `~/.lyricsync/window-state.yaml`.
- El fullscreen se toma del último estado reportado por la UI (`PUT /api/window/fullscreen`), no se lee de GTK: Wails ejecuta el callback de cierre en otra goroutine y su getter de fullscreen no es seguro fuera del hilo principal. Consecuencia: si cambiás fullscreen por fuera de la app (por ejemplo desde el window manager), ese cambio no se persiste.

## Features

- **App nativa**: empaquetado Wails v2, single binary, sin navegador
- **Cinema mode**: fullscreen nativo, oculta barras de UI, widget flotante con info del track, View Transitions API para animación suave
- **Panel de Settings**: fuente, tema (4 temas), colores, espaciado, idioma, offset de sync, toggle **Translate Lyrics**, API key de DeepSeek con toggle revelar/ocultar
- **Configuración de API key desde la UI**: hot-reload del cliente DeepSeek sin reiniciar, persistencia en `~/.config/lyricsync/config.yaml`
- Detección automática de **cualquier reproductor MPRIS** (Spotify, Brave, Chrome, apps)
- Letras sincronizadas (LRC) con highlight en tiempo real + click-to-seek
- **Doble fuente de letras**: LRCLib y **lrcmux** (agregador multi-origen), con **fallback en cadena** configurable
- **Karaoke unificado**: un solo toggle en Settings (default OFF) que pide timestamps word-level a lrcmux y pinta palabra por palabra cuando están disponibles; si no, cae al fill por línea. Apagado = pide line-level (cacheado, sin gastar rate-limit)
- **Nivel de sync visible**: el badge de fuente muestra el nivel real servido (\`· word\` / \`· line\` / \`· none\`) y avisa con \`word N/D\` cuando Word Karaoke está activo pero el provider no trajo timestamps por palabra (p. ej. LRCLIB nunca los tiene)
- **Badge de origen de letras**: muestra de qué provider salieron (ej. `lrcmux · Kugou`) en la barra Now Playing, el viewer y cinema mode
- **Cover art desde lrcmux**: portadas de Deezer CDN cacheadas en SQLite y servidas con la canción
- **ISRC + duración reales**: lrcmux devuelve el ISRC del track y la duración canónica; se cachean y se reutilizan en re-fetch (matching por ISRC tiene prioridad)
- **Toggle de traducción con costo cero**: apagado = ninguna llamada a DeepSeek/LibreTranslate; atajo `T`
- **Dedupe en vuelo**: evita resoluciones duplicadas cuando playerctl emite múltiples track events
- Traducción EN→ES (LibreTranslate o DeepSeek) con romanización de japonés, chino y coreano
- **Romanización prominente**: cuando la canción tiene transliteración, se muestra más grande que el texto original
- **Toast de error**: notificación flotante con auto-dismiss cuando falla el provider de traducción + botón Retry
- **Retry inteligente**: reintenta traducciones vacías (API key mala → corregir → siguiente reproducción)
- Controles del reproductor: play/pause, next/prev, seek, shuffle, loop, volumen con mute
- Biblioteca de canciones guardadas con búsqueda
- Atajos de teclado (`?` para ayuda, `T` para alternar traducción, `C` para cinema mode)
- SSE con replay de estado al reconectar + merge atómico de traducciones
- Cache SQLite de canciones y traducciones
- Persistencia de estado de ventana (posición, tamaño, fullscreen)
  El fullscreen se guarda desde el flag que reporta la UI (`PUT /api/window/fullscreen`), no leyendo el estado de GTK: Wails ejecuta `OnBeforeClose` en otra goroutine y su getter de fullscreen no es seguro fuera del hilo principal.
- Docker: multi-stage build, LibreTranslate incluido
