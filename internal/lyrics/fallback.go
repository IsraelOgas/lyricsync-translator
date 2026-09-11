package lyrics

import (
	"fmt"
	"log"
)

// FallbackProvider tries a list of providers in order and returns the first
// successful result. Implements LyricsProvider.
type FallbackProvider struct {
	providers []LyricsProvider
}

// NewFallbackProvider creates a fallback chain from the given providers,
// tried in the order they are passed.
func NewFallbackProvider(providers ...LyricsProvider) *FallbackProvider {
	return &FallbackProvider{providers: providers}
}

// Name returns the chain identifier used in logs.
func (f *FallbackProvider) Name() string { return "fallback" }

// SearchLyrics tries each provider in order until one returns a result with at
// least one line. A provider that returns nil, nil (or a result with zero
// lines) counts as a clean "not found" and we move to the next provider. A
// provider that returns an error is remembered and we also move on, because a
// later provider may still succeed. After the loop, if any provider errored we
// surface that error: the chain could NOT confirm the track is absent, so the
// caller can retry (e.g. a transient 429). "Not found" (nil, nil) is only
// reported when every provider answered without error.
func (f *FallbackProvider) SearchLyrics(artist, title string, durationMs int, level string, isrc string) (*LyricsResult, error) {
	var lastErr error

	for _, p := range f.providers {
		result, err := p.SearchLyrics(artist, title, durationMs, level, isrc)
		if err != nil {
			log.Printf("fallback: provider %s failed: %v", p.Name(), err)
			lastErr = err
			continue
		}
		if result == nil || len(result.Lines) == 0 {
			log.Printf("fallback: provider %s returned no lyrics, trying next", p.Name())
			continue
		}
		return result, nil
	}

	// An error means the chain could NOT confirm the track is absent, so
	// surface it for the caller to retry. Report "not found" only when every
	// provider answered cleanly (no error), or when the list is empty.
	if lastErr != nil {
		return nil, fmt.Errorf("all lyrics providers failed: %w", lastErr)
	}
	return nil, nil
}
