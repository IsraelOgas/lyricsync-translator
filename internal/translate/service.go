package translate

import (
	"context"
	"log"
	"sync"
)

// Service orchestrates translation via the Translator interface.
type Service struct {
	translator Translator
	targetLang string
	mu         sync.RWMutex
}

// NewService creates a new translation service with default target language.
func NewService(translator Translator, targetLang string) *Service {
	if targetLang == "" {
		targetLang = "es"
	}
	return &Service{translator: translator, targetLang: targetLang}
}

// SetTargetLang updates the target language for subsequent translations.
func (s *Service) SetTargetLang(lang string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.targetLang = lang
	log.Printf("translate: target language set to %s", lang)
}

// TargetLang returns the current target language.
func (s *Service) TargetLang() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.targetLang
}

// LineResult holds the processed output for one original line.
type LineResult struct {
	Original   string `json:"original"`
	Romanized  string `json:"romanized"`
	Translated string `json:"translated"`
	Lang       string `json:"lang"`
}

// ProcessLine detects language and translates a single line.
func (s *Service) ProcessLine(ctx context.Context, original string) (romanized, translated string, err error) {
	if original == "" {
		return "", "", nil
	}

	lang := DetectLanguage(original)
	target := s.TargetLang()
	translated, err = s.translator.Translate(original, lang, target)
	if err != nil {
		translated = ""
	}

	return "", translated, nil
}

// ProcessLines sends all lines to the translator in a single batch call.
// Skips romanization for Latin-only text and skips translation when source matches target.
func (s *Service) ProcessLines(ctx context.Context, lines []string) ([]LineResult, error) {
	if len(lines) == 0 {
		return nil, nil
	}

	n := len(lines)
	target := s.TargetLang()

	// Detect predominant source language and whether text is all Latin
	allLatin := true
	sourceLangs := make(map[string]int)
	for _, line := range lines {
		lang := DetectLanguage(line)
		sourceLangs[lang]++
		if !isLatinOnly(line) {
			allLatin = false
		}
	}

	// Find the predominant detected language
	predominant := "auto"
	maxCount := 0
	for lang, count := range sourceLangs {
		if count > maxCount {
			maxCount = count
			predominant = lang
		}
	}

	// Skip romanization if all text is already Latin
	if allLatin {
		log.Printf("translate: all lines are Latin script, skipping romanization")
	}

	// Skip translation if source language matches target
	skipTranslation := predominant != "auto" && predominant == target
	if skipTranslation {
		log.Printf("translate: source language matches target (%s), skipping translation", target)
	}

	if allLatin && skipTranslation {
		// Nothing to do — copy original to both fields
		results := make([]LineResult, n)
		for i, line := range lines {
			results[i] = LineResult{
				Original:   line,
				Romanized:  "",
				Translated: line,
				Lang:       DetectLanguage(line),
			}
		}
		return results, nil
	}

	log.Printf("translate: sending %d lines to provider (target=%s)", n, target)
	romanized, translated, err := s.translator.TranslateBatch(lines, "auto", target)
	if err != nil {
		log.Printf("translate: batch failed: %v", err)
	}
	romanized = padResults(romanized, n)
	translated = padResults(translated, n)

	if allLatin {
		// Clear romanized for Latin text
		for i := range romanized {
			romanized[i] = ""
		}
	}

	log.Printf("translate: got %d romanized, %d translations back", len(romanized), len(translated))

	results := make([]LineResult, n)
	for i := range lines {
		results[i] = LineResult{
			Original:   lines[i],
			Romanized:  romanized[i],
			Translated: translated[i],
			Lang:       DetectLanguage(lines[i]),
		}
	}

	return results, nil
}

// padResults pads a slice to length n with empty strings.
func padResults(s []string, n int) []string {
	for len(s) < n {
		s = append(s, "")
	}
	return s
}
