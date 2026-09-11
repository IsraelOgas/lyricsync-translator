package lyrics

import (
	"fmt"
	"testing"
)

func TestLrcMux_RealAPI(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping live API test in short mode")
	}

	client := NewLrcMuxClient("https://api.lrcmux.dev", 15)

	tests := []struct {
		artist, title string
		wantSynced    bool
	}{
		{"Coldplay", "Yellow", true},
		{"Soda Stereo", "De Musica Ligera", true},
		{"Bad Bunny", "Titi Me Pregunto", true},
		{"InventedArtistXYZ", "NonexistentSong", false},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s-%s", tt.artist, tt.title), func(t *testing.T) {
			result, err := client.SearchLyrics(tt.artist, tt.title, 0, "word", "")
			if err != nil {
				t.Fatalf("SearchLyrics error: %v", err)
			}

			if tt.wantSynced {
				if result == nil {
					t.Fatal("expected result, got nil")
				}
				if !result.Synced {
					t.Error("expected synced lyrics")
				}
				if len(result.Lines) == 0 {
					t.Error("expected at least one line")
				}
				t.Logf("Artist: %s - Title: %s", tt.artist, tt.title)
				t.Logf("  Source: %s", result.Source)
				t.Logf("  ISRC: %s", result.ISRC)
				t.Logf("  DurationMs: %d", result.DurationMs)
				t.Logf("  Lines: %d", len(result.Lines))

				// Verify cover art.
				if result.CoverArt != nil {
					t.Logf("  Cover: small=%s, medium=%s, big=%s",
						result.CoverArt.Small, result.CoverArt.Medium, result.CoverArt.Big)
				} else {
					t.Log("  Cover: nil")
				}

				// Verify first line.
				if len(result.Lines) > 0 {
					first := result.Lines[0]
					t.Logf("  First line: %q", first.Text)
					if first.TimeMs != nil {
						t.Logf("  First timestamp: %dms", *first.TimeMs)
					}
					// Verify word-level timestamps.
					if len(first.Words) > 0 {
						t.Logf("  Words: %d (first=%q start=%dms end=%dms)",
							len(first.Words), first.Words[0].Text,
							first.Words[0].StartMs, first.Words[0].EndMs)
					} else {
						t.Log("  Words: 0")
					}
				}
			} else {
				if result != nil {
					t.Errorf("expected nil for nonexistent song, got %d lines", len(result.Lines))
				}
			}
		})
	}
}
