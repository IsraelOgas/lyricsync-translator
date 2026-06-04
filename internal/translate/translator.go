package translate

// Translator is the abstraction for translation providers (LibreTranslate, DeepSeek, etc.).
// TranslateBatch returns both romanized (transliterated to ASCII) and translated text.
// LibreTranslate returns empty romanized since it doesn't handle romanization.
// Output slices MUST match input length.
type Translator interface {
	Translate(text, sourceLang, targetLang string) (string, error)
	TranslateBatch(lines []string, sourceLang, targetLang string) (romanized, translated []string, err error)
}
