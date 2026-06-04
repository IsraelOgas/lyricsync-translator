package translate

import (
	"context"
	"log"
)

// Service orchestrates translation via the Translator interface.
type Service struct {
	translator Translator
}

// NewService creates a new translation service.
func NewService(translator Translator) *Service {
	return &Service{translator: translator}
}

// LineResult holds the processed output for one original line.
type LineResult struct {
	Original   string `json:"original"`
	Romanized  string `json:"romanized"`
	Translated string `json:"translated"`
	Lang       string `json:"lang"`
}

// ProcessLine detects language and translates a single line to Spanish.
func (s *Service) ProcessLine(ctx context.Context, original string) (romanized, translated string, err error) {
	if original == "" {
		return "", "", nil
	}

	lang := DetectLanguage(original)
	translated, err = s.translator.Translate(original, lang, "es")
	if err != nil {
		translated = ""
	}

	return "", translated, nil
}

// ProcessLines sends all lines to the translator in a single batch call.
// The Translator handles both romanization and translation internally.
func (s *Service) ProcessLines(ctx context.Context, lines []string) ([]LineResult, error) {
	if len(lines) == 0 {
		return nil, nil
	}

	n := len(lines)

	log.Printf("translate: sending %d lines to provider", n)
	romanized, translated, err := s.translator.TranslateBatch(lines, "auto", "es")
	if err != nil {
		log.Printf("translate: batch failed: %v", err)
		romanized = make([]string, n)
		translated = make([]string, n)
	} else {
		log.Printf("translate: got %d romanized, %d translations back", len(romanized), len(translated))
	}

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
