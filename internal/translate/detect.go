package translate

import "unicode"

// DetectLanguage returns an ISO 639-1 language code for the given text
// based on Unicode character ranges. Returns "auto" if undetermined.
func DetectLanguage(text string) string {
	hasHiragana := false
	hasKatakana := false
	hasKanji := false
	hasHangul := false
	hasHan := false

	for _, r := range text {
		switch {
		case unicode.Is(unicode.Hiragana, r):
			hasHiragana = true
		case unicode.Is(unicode.Katakana, r):
			hasKatakana = true
		case r >= 0x4E00 && r <= 0x9FFF:
			hasKanji = true
			hasHan = true
		case r >= 0xAC00 && r <= 0xD7AF:
			hasHangul = true
		case r >= 0x3400 && r <= 0x4DBF:
			hasHan = true
		}
	}

	if hasHiragana || hasKatakana || hasKanji {
		return "ja"
	}
	if hasHangul {
		return "ko"
	}
	if hasHan {
		return "zh"
	}
	return "auto"
}
