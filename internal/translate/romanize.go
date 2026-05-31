package translate

import (
	"unicode"
)

// DetectLanguage returns an ISO 639-1 language code for the given text
// based on Unicode character ranges. Returns "auto" if undetermined.
func DetectLanguage(text string) string {
	hasHiragana := false
	hasKatakana := false
	hasKanji := false
	hasHangul := false
	hasHan := false // Chinese

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

// ShouldRomanize returns true if the language is in the enabled list.
func ShouldRomanize(lang string, enabledLanguages []string) bool {
	for _, l := range enabledLanguages {
		if l == lang {
			return true
		}
	}
	return false
}

// RomanizeText performs romanization for supported languages.
// Currently supports basic Japanese kana romanization.
// For other languages, returns the original text unchanged.
func RomanizeText(text, lang string) (string, error) {
	if lang == "ja" {
		return romanizeJapanese(text), nil
	}
	// For other languages, romanization is more complex
	// Return original text for now
	return text, nil
}

// romanizeJapanese converts Japanese kana to romaji using a simplified mapping.
func romanizeJapanese(text string) string {
	var result []rune
	for _, r := range text {
		if romaji, ok := kanaToRomaji[r]; ok {
			result = append(result, []rune(romaji)...)
		} else {
			result = append(result, r)
		}
	}
	return string(result)
}

// kanaToRomaji maps hiragana characters to Hepburn romaji.
// This is a simplified mapping for the MVP.
var kanaToRomaji = map[rune]string{
	// Hiragana
	0x3042: "a", 0x3044: "i", 0x3046: "u", 0x3048: "e", 0x304A: "o",
	0x304B: "ka", 0x304D: "ki", 0x304F: "ku", 0x3051: "ke", 0x3053: "ko",
	0x3055: "sa", 0x3057: "shi", 0x3059: "su", 0x305B: "se", 0x305D: "so",
	0x305F: "ta", 0x3061: "chi", 0x3064: "tsu", 0x3066: "te", 0x3068: "to",
	0x306A: "na", 0x306B: "ni", 0x306C: "nu", 0x306D: "ne", 0x306E: "no",
	0x306F: "ha", 0x3072: "hi", 0x3075: "fu", 0x3078: "he", 0x307B: "ho",
	0x307E: "ma", 0x307F: "mi", 0x3080: "mu", 0x3081: "me", 0x3082: "mo",
	0x3084: "ya", 0x3086: "yu", 0x3088: "yo",
	0x3089: "ra", 0x308A: "ri", 0x308B: "ru", 0x308C: "re", 0x308D: "ro",
	0x308F: "wa", 0x3092: "wo", 0x3093: "n",
	// Katakana
	0x30A2: "a", 0x30A4: "i", 0x30A6: "u", 0x30A8: "e", 0x30AA: "o",
	0x30AB: "ka", 0x30AD: "ki", 0x30AF: "ku", 0x30B1: "ke", 0x30B3: "ko",
	0x30B5: "sa", 0x30B7: "shi", 0x30B9: "su", 0x30BB: "se", 0x30BD: "so",
	0x30BF: "ta", 0x30C1: "chi", 0x30C4: "tsu", 0x30C6: "te", 0x30C8: "to",
	0x30CA: "na", 0x30CB: "ni", 0x30CC: "nu", 0x30CD: "ne", 0x30CE: "no",
	0x30CF: "ha", 0x30D2: "hi", 0x30D5: "fu", 0x30D8: "he", 0x30DB: "ho",
	0x30DE: "ma", 0x30DF: "mi", 0x30E0: "mu", 0x30E1: "me", 0x30E2: "mo",
	0x30E4: "ya", 0x30E6: "yu", 0x30E8: "yo",
	0x30E9: "ra", 0x30EA: "ri", 0x30EB: "ru", 0x30EC: "re", 0x30ED: "ro",
	0x30EF: "wa", 0x30F2: "wo", 0x30F3: "n",
	// Common dakuten (voiced) characters
	0x304C: "ga", 0x304E: "gi", 0x3050: "gu", 0x3052: "ge", 0x3054: "go",
	0x3056: "za", 0x3058: "ji", 0x305A: "zu", 0x305C: "ze", 0x305E: "zo",
	0x3060: "da", 0x3062: "ji", 0x3065: "zu", 0x3067: "de", 0x3069: "do",
	0x3070: "ba", 0x3073: "bi", 0x3076: "bu", 0x3079: "be", 0x307C: "bo",
	0x3071: "pa", 0x3074: "pi", 0x3077: "pu", 0x307A: "pe", 0x307D: "po",
}
