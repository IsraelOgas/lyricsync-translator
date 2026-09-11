package lyrics

import (
	"errors"
	"strings"
	"testing"
)

type mockProvider struct {
	name   string
	result *LyricsResult
	err    error
	calls  *int // optional call counter to assert ordering
}

func (m *mockProvider) Name() string { return m.name }
func (m *mockProvider) SearchLyrics(artist, title string, durationMs int, level string, isrc string) (*LyricsResult, error) {
	if m.calls != nil {
		*m.calls++
	}
	return m.result, m.err
}

func resultWithLines() *LyricsResult {
	return &LyricsResult{
		Source: "test",
		Synced: true,
		Lines:  []LyricLine{{Text: "hello"}},
	}
}

func TestFallbackProvider_SearchLyrics(t *testing.T) {
	tests := []struct {
		name            string
		providers       func() ([]LyricsProvider, func(t *testing.T))
		wantLines       bool
		wantErr         bool
		wantErrContains string
	}{
		{
			name: "first provider succeeds, second never called",
			providers: func() ([]LyricsProvider, func(t *testing.T)) {
				secondCalls := new(int)
				providers := []LyricsProvider{
					&mockProvider{name: "first", result: resultWithLines()},
					&mockProvider{name: "second", result: resultWithLines(), calls: secondCalls},
				}
				verify := func(t *testing.T) {
					if *secondCalls != 0 {
						t.Errorf("second provider called %d times, want 0", *secondCalls)
					}
				}
				return providers, verify
			},
			wantLines: true,
		},
		{
			name: "first not found, second succeeds",
			providers: func() ([]LyricsProvider, func(t *testing.T)) {
				return []LyricsProvider{
					&mockProvider{name: "first", result: nil},
					&mockProvider{name: "second", result: resultWithLines()},
				}, nil
			},
			wantLines: true,
		},
		{
			name: "first errors, second succeeds",
			providers: func() ([]LyricsProvider, func(t *testing.T)) {
				return []LyricsProvider{
					&mockProvider{name: "first", err: errors.New("boom")},
					&mockProvider{name: "second", result: resultWithLines()},
				}, nil
			},
			wantLines: true,
		},
		{
			name: "all providers not found",
			providers: func() ([]LyricsProvider, func(t *testing.T)) {
				return []LyricsProvider{
					&mockProvider{name: "first", result: nil},
					&mockProvider{name: "second", result: nil},
				}, nil
			},
			wantLines: false,
		},
		{
			name: "all providers error",
			providers: func() ([]LyricsProvider, func(t *testing.T)) {
				return []LyricsProvider{
					&mockProvider{name: "first", err: errors.New("down-a")},
					&mockProvider{name: "second", err: errors.New("down-b")},
				}, nil
			},
			wantErr:         true,
			wantErrContains: "all lyrics providers failed",
		},
		{
			name: "empty provider list",
			providers: func() ([]LyricsProvider, func(t *testing.T)) {
				return []LyricsProvider{}, nil
			},
			wantLines: false,
		},
		{
			name: "zero-line result treated as not found",
			providers: func() ([]LyricsProvider, func(t *testing.T)) {
				return []LyricsProvider{
					&mockProvider{name: "first", result: &LyricsResult{Lines: []LyricLine{}}},
					&mockProvider{name: "second", result: resultWithLines()},
				}, nil
			},
			wantLines: true,
		},
		{
			name: "first not found, second errors",
			providers: func() ([]LyricsProvider, func(t *testing.T)) {
				return []LyricsProvider{
					&mockProvider{name: "first", result: nil},
					&mockProvider{name: "second", err: errors.New("rate limited")},
				}, nil
			},
			wantErr:         true,
			wantErrContains: "all lyrics providers failed",
		},
		{
			name: "first errors, second not found",
			providers: func() ([]LyricsProvider, func(t *testing.T)) {
				return []LyricsProvider{
					&mockProvider{name: "first", err: errors.New("boom")},
					&mockProvider{name: "second", result: nil},
				}, nil
			},
			wantErr:         true,
			wantErrContains: "all lyrics providers failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			providers, verify := tt.providers()
			fb := NewFallbackProvider(providers...)

			result, err := fb.SearchLyrics("artist", "title", 0, "line", "")

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.wantErrContains != "" && !strings.Contains(err.Error(), tt.wantErrContains) {
					t.Errorf("error %q does not contain %q", err.Error(), tt.wantErrContains)
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.wantLines {
				if result == nil {
					t.Fatal("expected result, got nil")
				}
				if len(result.Lines) == 0 {
					t.Error("expected at least one line")
				}
			} else if result != nil {
				t.Errorf("expected nil result, got %d lines", len(result.Lines))
			}

			if verify != nil {
				verify(t)
			}
		})
	}
}
