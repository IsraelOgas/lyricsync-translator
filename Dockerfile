# Stage 1 — Build React frontend
FROM node:22-alpine AS frontend
RUN corepack enable && corepack prepare pnpm@latest --activate
WORKDIR /app/web
COPY web/package.json web/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile
COPY web/ ./
RUN pnpm build

# Stage 2 — Build Go backend
FROM golang:1.26-alpine AS backend
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
COPY --from=frontend /app/web/dist ./web/dist
RUN CGO_ENABLED=0 go build -o lyricsync ./cmd/lyricsync/

# Stage 3 — Final image
FROM alpine:3.22
RUN apk add --no-cache playerctl dbus ca-certificates tzdata

# Create non-root user matching typical host UID for D-Bus access
RUN adduser -D -u 1000 lyricsync && \
    mkdir -p /data && \
    chown lyricsync:lyricsync /data

COPY --from=backend --chown=lyricsync:lyricsync /app/lyricsync /usr/local/bin/lyricsync
COPY --from=backend --chown=lyricsync:lyricsync /app/web/dist /app/web/dist

ENV WEB_DIR=/app/web/dist
ENV LYRIC_HOST=0.0.0.0
ENV LYRIC_DB_PATH=/data/cache.db

USER lyricsync
EXPOSE 8090
CMD ["lyricsync"]
