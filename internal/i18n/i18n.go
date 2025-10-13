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
	t.mu.Lock()
	defer t.mu.Unlock()

	// Clear existing translations before reloading
	t.translations = make(map[string]map[string]string)

	langDirs, err := ioutil.ReadDir(t.basePath)
	if err != nil {
		return fmt.Errorf("failed to read i18n base directory: %w", err)
	}

	for _, langDir := range langDirs {
		if !langDir.IsDir() {
			continue // Skip files directly in basePath, only process language directories
		}

		langCode := langDir.Name()
		langPath := filepath.Join(t.basePath, langCode)

		moduleFiles, err := ioutil.ReadDir(langPath)
		if err != nil {
			return fmt.Errorf("failed to read language directory %s: %w", langPath, err)
		}

		t.translations[langCode] = make(map[string]string) // Initialize map for this language

		for _, moduleFile := range moduleFiles {
			if moduleFile.IsDir() || filepath.Ext(moduleFile.Name()) != ".json" {
				continue // Skip subdirectories and non-JSON files
			}

			moduleName := moduleFile.Name()[:len(moduleFile.Name())-len(filepath.Ext(moduleFile.Name()))] // e.g., "common"
			filePath := filepath.Join(langPath, moduleFile.Name())
			content, err := ioutil.ReadFile(filePath)
			if err != nil {
				return fmt.Errorf("failed to read translation file %s: %w", filePath, err)
			}

			var moduleMap map[string]string
			if err := json.Unmarshal(content, &moduleMap); err != nil {
				return fmt.Errorf("failed to unmarshal translation file %s: %w", filePath, err)
			}

			// Merge module translations into the language map with a prefix
			for key, value := range moduleMap {
				// Use "moduleName.key" as the new key to avoid conflicts and provide structure
				t.translations[langCode][fmt.Sprintf("%s.%s", moduleName, key)] = value
			}
		}
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
