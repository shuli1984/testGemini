package i18n_test

import (
	"gemini-demo/internal/i18n"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func createTempLangFiles(t *testing.T) (string, func()) {
	tempDir, err := ioutil.TempDir("", "i18n_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	enContent := []byte(`{
		"hello": "Hello",
		"world": "World",
		"only_en": "English Only"
	}`)
	
	zhContent := []byte(`{
		"hello": "你好",
		"world": "世界",
		"only_zh": "仅中文"
	}`)

	if err := ioutil.WriteFile(filepath.Join(tempDir, "en.json"), enContent, 0644); err != nil {
		t.Fatalf("Failed to write en.json: %v", err)
	}
	if err := ioutil.WriteFile(filepath.Join(tempDir, "zh.json"), zhContent, 0644); err != nil {
		t.Fatalf("Failed to write zh.json: %v", err)
	}

	return tempDir, func() {
		os.RemoveAll(tempDir)
	}
}

func TestNewTranslator(t *testing.T) {
	translator := i18n.NewTranslator("path/to/langs", "en")

	if translator == nil {
		t.Error("NewTranslator returned nil")
	}
}

func TestLoadTranslations(t *testing.T) {
	tempDir, cleanup := createTempLangFiles(t)
	defer cleanup()

	translator := i18n.NewTranslator(tempDir, "en")
	err := translator.LoadTranslations()
	if err != nil {
		t.Fatalf("LoadTranslations failed: %v", err)
	}

	if !translator.IsValidLanguage("en") {
		t.Error("English language not loaded")
	}
	if !translator.IsValidLanguage("zh") {
		t.Error("Chinese language not loaded")
	}
	if translator.IsValidLanguage("fr") {
		t.Error("French language unexpectedly loaded")
	}
}

func TestGetTranslation(t *testing.T) {
	tempDir, cleanup := createTempLangFiles(t)
	defer cleanup()

	translator := i18n.NewTranslator(tempDir, "en")
	err := translator.LoadTranslations()
	if err != nil {
		t.Fatalf("LoadTranslations failed: %v", err)
	}

	// Test existing translation in requested language
	if trans := translator.GetTranslation("en", "hello"); trans != "Hello" {
		t.Errorf("Expected 'Hello', got '%s' for en/hello", trans)
	}
	if trans := translator.GetTranslation("zh", "hello"); trans != "你好" {
		t.Errorf("Expected '你好', got '%s' for zh/hello", trans)
	}

	// Test fallback to default language
	if trans := translator.GetTranslation("zh", "only_en"); trans != "English Only" {
		t.Errorf("Expected 'English Only', got '%s' for zh/only_en (fallback)", trans)
	}

	// Test key not found in any language
	if trans := translator.GetTranslation("en", "non_existent_key"); trans != "non_existent_key" {
		t.Errorf("Expected 'non_existent_key', got '%s' for non_existent_key", trans)
	}

	// Test invalid language, should fallback to default
	if trans := translator.GetTranslation("fr", "hello"); trans != "Hello" {
		t.Errorf("Expected 'Hello', got '%s' for fr/hello (invalid lang fallback)", trans)
	}
}

func TestGetAvailableLanguages(t *testing.T) {
	tempDir, cleanup := createTempLangFiles(t)
	defer cleanup()

	translator := i18n.NewTranslator(tempDir, "en")
	err := translator.LoadTranslations()
	if err != nil {
		t.Fatalf("LoadTranslations failed: %v", err)
	}

	langs := translator.GetAvailableLanguages()
	if len(langs) != 2 {
		t.Errorf("Expected 2 languages, got %d", len(langs))
	}

	foundEn := false
	foundZh := false
	for _, lang := range langs {
		if lang == "en" {
			foundEn = true
		}
		if lang == "zh" {
			foundZh = true
		}
	}

	if !foundEn || !foundZh {
		t.Error("Expected 'en' and 'zh' in available languages")
	}
}

func TestIsValidLanguage(t *testing.T) {
	tempDir, cleanup := createTempLangFiles(t)
	defer cleanup()

	translator := i18n.NewTranslator(tempDir, "en")
	err := translator.LoadTranslations()
	if err != nil {
		t.Fatalf("LoadTranslations failed: %v", err)
	}

	if !translator.IsValidLanguage("en") {
		t.Error("Expected 'en' to be valid")
	}
	if !translator.IsValidLanguage("zh") {
		t.Error("Expected 'zh' to be valid")
	}
	if translator.IsValidLanguage("fr") {
		t.Error("Expected 'fr' to be invalid")
	}
}

func TestDefaultLanguage(t *testing.T) {
	translator := i18n.NewTranslator("path/to/langs", "es")
	if translator.DefaultLanguage() != "es" {
		t.Errorf("Expected default language 'es', got '%s'", translator.DefaultLanguage())
	}
}
