package i18n

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"sync"
)

// Translator holds the loaded translations for different languages.
type Translator struct {
	translations map[string]map[string]string
	mu           sync.RWMutex
	basePath     string
	defaultLang  string
}

// NewTranslator creates a new Translator instance.
func NewTranslator(basePath, defaultLang string) *Translator {
	return &Translator{
		translations: make(map[string]map[string]string),
		basePath:     basePath,
		defaultLang:  defaultLang,
	}
}

// LoadTranslations loads translation files from the specified base path.
func (t *Translator) LoadTranslations() error {
	files, err := ioutil.ReadDir(t.basePath)
	if err != nil {
		return fmt.Errorf("failed to read i18n directory: %w", err)
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		lang := file.Name()
		if filepath.Ext(lang) != ".json" {
			continue
		}
		lang = lang[:len(lang)-len(filepath.Ext(lang))] // Remove .json extension

		filePath := filepath.Join(t.basePath, file.Name())
		content, err := ioutil.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read translation file %s: %w", filePath, err)
		}

		var langMap map[string]string
		if err := json.Unmarshal(content, &langMap); err != nil {
			return fmt.Errorf("failed to unmarshal translation file %s: %w", filePath, err)
		}
		t.translations[lang] = langMap
	}
	return nil
}

// GetTranslation retrieves a translated string for a given key and language.
// It falls back to the default language if the key is not found in the requested language.
func (t *Translator) GetTranslation(lang, key string) string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if langMap, ok := t.translations[lang]; ok {
		if val, ok := langMap[key]; ok {
			return val
		}
	}

	// Fallback to default language
	if defaultLangMap, ok := t.translations[t.defaultLang]; ok {
		if val, ok := defaultLangMap[key]; ok {
			return val
		}
	}

	return key // Return the key itself if no translation is found
}

// GetAvailableLanguages returns a list of available languages.
func (t *Translator) GetAvailableLanguages() []string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	languages := make([]string, 0, len(t.translations))
	for lang := range t.translations {
		languages = append(languages, lang)
	}
	return languages
}

// IsValidLanguage checks if a language is loaded.
func (t *Translator) IsValidLanguage(lang string) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	_, ok := t.translations[lang]
	return ok
}

// DefaultLanguage returns the default language set for the translator.
func (t *Translator) DefaultLanguage() string {
	return t.defaultLang
}
