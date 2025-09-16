package translator

import (
	"context"
	"fmt"
)

// MockTranslator is a mock implementation of the Translator interface for testing.
type MockTranslator struct{}

func (m *MockTranslator) TranslateText(ctx context.Context, text, sourceLang, targetLang string) (string, error) {
	// For testing, just return the original text with language codes
	return fmt.Sprintf("%s (translated from %s to %s)", text, sourceLang, targetLang), nil
}

func (m *MockTranslator) Close() error {
	return nil
}
