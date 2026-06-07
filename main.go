package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/imov/lyricsync-translator/internal/api"
	"github.com/imov/lyricsync-translator/internal/cache"
	"github.com/imov/lyricsync-translator/internal/config"
	"github.com/imov/lyricsync-translator/internal/lyrics"
	"github.com/imov/lyricsync-translator/internal/player"
	"github.com/imov/lyricsync-translator/internal/translate"
)

// IsProduction is set via build-tag-gated init() in main_prod.go / main_dev.go.
// Wails adds the "desktop" build tag for production builds (wails build)
// and omits it for dev mode (wails dev).
var IsProduction bool

func main() {
	// Load .env file if present (does not override existing env vars)
	if err := godotenv.Load(); err != nil {
		log.Printf("No .env file found, using system env vars only")
	}

	cfg, err := config.Load("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("LyricsSync Translator starting on %s\n", cfg.Server.Address())

	store, err := cache.NewStore(cfg.Cache.DBPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening database: %v\n", err)
		os.Exit(1)
	}

	if err := store.Migrate(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running migrations: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Database ready at %s\n", cfg.Cache.DBPath)

	tracker := player.NewTracker(cfg.Player.PlayerctlPath)
	fmt.Println("Player tracker started")

	var translator translate.Translator
	switch cfg.Translation.Provider {
	case "libretranslate":
		translator = translate.NewLibreTranslateClient(
			cfg.Translation.LibreTranslate.BaseURL,
			cfg.Translation.LibreTranslate.TimeoutSec,
			cfg.Translation.LibreTranslate.APIKey,
		)
	case "deepseek":
		translator = translate.NewDeepSeekClient(
			cfg.Translation.DeepSeek.APIKey,
			cfg.Translation.DeepSeek.Model,
			cfg.Translation.DeepSeek.BaseURL,
			cfg.Translation.DeepSeek.TimeoutSec,
		)
	default:
		log.Fatalf("Unknown translation provider: %s (expected 'libretranslate' or 'deepseek')", cfg.Translation.Provider)
	}
	tranSvc := translate.NewService(translator, cfg.Translation.TargetLang)
	fmt.Println("Translation service ready")

	lyricsProvider := lyrics.NewProvider(cfg.Lyrics.Provider, cfg.Lyrics.LRCLib.BaseURL, cfg.Lyrics.LRCLib.TimeoutSec)
	if lyricsProvider == nil {
		fmt.Fprintf(os.Stderr, "Unknown lyrics provider: %s\n", cfg.Lyrics.Provider)
		os.Exit(1)
	}
	lyricsSvc := lyrics.NewService(lyricsProvider, store, tranSvc)
	fmt.Println("Lyrics service ready")

	srv := api.NewServer(cfg, store, tracker, tranSvc, lyricsSvc)

	// In production, chi serves both API and embedded SPA assets.
	// In dev mode, chi serves API routes and reverse-proxies frontend
	// requests to the Vite dev server so HMR works correctly.
	// Wails v2 requires Handler to be set in AssetServer.Options.
	viteURL := ""
	if !IsProduction {
		viteURL = "http://localhost:5173"
	}
	apiBase := cfg.Server.APIBase()
	srv.EnableEmbeddedAssets(FS(), apiBase, viteURL)

	// Launch Wails desktop app.
	err = wails.Run(&options.App{
		Title:  "LyricSync",
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
			Handler: srv.Handler(), // Chi handles API + frontend (SPA or Vite proxy).
		},
		OnStartup: func(ctx context.Context) {
			fmt.Println("App started — launching server")
			srv.Start(ctx)

			// Restore previous window state.
			wc, err := config.LoadWindowState()
			if err != nil {
				log.Printf("Could not load window state: %v", err)
				return
			}
			if wc.X != 0 || wc.Y != 0 {
				runtime.WindowSetPosition(ctx, wc.X, wc.Y)
			}
			if wc.Width > 0 && wc.Height > 0 {
				runtime.WindowSetSize(ctx, wc.Width, wc.Height)
			}
			if wc.Fullscreen {
				runtime.WindowFullscreen(ctx)
			}
		},
		OnShutdown: func(ctx context.Context) {
			fmt.Println("\nShutting down...")
			srv.Shutdown(ctx)
			store.Close()
		},
		OnBeforeClose: func(ctx context.Context) bool {
			x, y := runtime.WindowGetPosition(ctx)
			w, h := runtime.WindowGetSize(ctx)
			isFullscreen := runtime.WindowIsFullscreen(ctx)

			wc := &config.WindowConfig{
				X:          x,
				Y:          y,
				Width:      w,
				Height:     h,
				Fullscreen: isFullscreen,
			}
			if err := config.SaveWindowState(wc); err != nil {
				log.Printf("Could not save window state: %v", err)
			}
			return false // allow close
		},
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
