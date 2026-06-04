package translate

import (
	"github.com/abadojack/whatlanggo"
)

// DetectLanguage returns an ISO 639-1 language code for the given text.
// Uses whatlanggo for accurate detection of 84 languages.
func DetectLanguage(text string) string {
	if text == "" {
		return "auto"
	}
	info := whatlanggo.Detect(text)
	lang := info.Lang.Iso6391()
	if lang == "" || !info.IsReliable() {
		return "auto"
	}
	// fmt.Printf("Detected language: %s (reliable: %t)\n", lang, info.IsReliable())
	return lang
}

// isLatinOnly returns true if the text contains only ASCII/Latin characters.
func isLatinOnly(text string) bool {
	for _, r := range text {
		if r > 0x024F { // beyond Latin Extended-B
			if r >= 0x2000 && r <= 0x206F { // common punctuation
				continue
			}
			return false
		}
	}
	return true
}
