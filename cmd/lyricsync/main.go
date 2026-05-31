package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/imov/lyricsync-translator/internal/api"
	"github.com/imov/lyricsync-translator/internal/cache"
	"github.com/imov/lyricsync-translator/internal/config"
	"github.com/imov/lyricsync-translator/internal/lyrics"
	"github.com/imov/lyricsync-translator/internal/player"
	"github.com/imov/lyricsync-translator/internal/translate"
)

func main() {
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
	defer store.Close()

	if err := store.Migrate(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running migrations: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Database ready at %s\n", cfg.Cache.DBPath)

	tracker := player.NewTracker(cfg.Player.PlayerctlPath)
	fmt.Println("Player tracker started")

	tranClient := translate.NewClient(
		cfg.Translation.LibreTranslate.BaseURL,
		cfg.Translation.LibreTranslate.TimeoutSec,
		cfg.Translation.LibreTranslate.APIKey,
	)
	tranSvc := translate.NewService(tranClient, cfg.Translation.Romanization.Languages)
	fmt.Println("Translation service ready")

	lyricsProvider := lyrics.NewProvider(cfg.Lyrics.Provider, cfg.Lyrics.LRCLib.BaseURL, cfg.Lyrics.LRCLib.TimeoutSec)
	if lyricsProvider == nil {
		fmt.Fprintf(os.Stderr, "Unknown lyrics provider: %s\n", cfg.Lyrics.Provider)
		os.Exit(1)
	}
	lyricsSvc := lyrics.NewService(lyricsProvider, store, tranSvc)
	fmt.Println("Lyrics service ready")

	srv := api.NewServer(cfg, store, tracker, tranSvc, lyricsSvc)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		fmt.Println("\nShutting down...")
		cancel()
		srv.Shutdown(context.Background())
	}()

	if err := srv.Start(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}
}
