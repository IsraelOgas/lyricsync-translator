package translate

import (
	"context"
	"sync"
)

// Service orchestrates romanization and translation.
type Service struct {
	translateClient *Client
	romanLanguages  []string
}

// NewService creates a new translation service.
func NewService(translateClient *Client, romanLanguages []string) *Service {
	return &Service{
		translateClient: translateClient,
		romanLanguages:  romanLanguages,
	}
}

// LineResult holds the processed output for one original line.
type LineResult struct {
	Original   string `json:"original"`
	Romanized  string `json:"romanized"`
	Translated string `json:"translated"`
	Lang       string `json:"lang"`
}

// ProcessLine detects language, romanizes if applicable, and translates to Spanish.
func (s *Service) ProcessLine(ctx context.Context, original string) (romanized, translated string, err error) {
	if original == "" {
		return "", "", nil
	}

	lang := DetectLanguage(original)
	if ShouldRomanize(lang, s.romanLanguages) {
		romanized, err = RomanizeText(original, lang)
		if err != nil {
			romanized = ""
		}
	}

	translated, err = s.translateClient.Translate(original, lang, "es")
	if err != nil {
		translated = ""
	}

	return romanized, translated, nil
}

// ProcessLines processes multiple lines concurrently with a worker pool.
func (s *Service) ProcessLines(ctx context.Context, lines []string) ([]LineResult, error) {
	if len(lines) == 0 {
		return nil, nil
	}

	results := make([]LineResult, len(lines))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 4) // max 4 concurrent API calls

	for i, line := range lines {
		select {
		case <-ctx.Done():
			break
		default:
		}

		wg.Add(1)
		sem <- struct{}{}
		go func(idx int, text string) {
			defer wg.Done()
			defer func() { <-sem }()

			romanized, translated, err := s.ProcessLine(ctx, text)
			lang := DetectLanguage(text)

			results[idx] = LineResult{
				Original:   text,
				Romanized:  romanized,
				Translated: translated,
				Lang:       lang,
			}
			_ = err // non-blocking: errors are captured in empty fields
		}(i, line)
	}

	wg.Wait()
	return results, nil
}
