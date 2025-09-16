package translator

import (
	"context"
	"fmt"
	"log"
	"strings"

	"cloud.google.com/go/translate"
	"golang.org/x/text/language"
	"google.golang.org/api/option"
)

// Translator defines the interface for a translation service.
type Translator interface {
	TranslateText(ctx context.Context, text, sourceLang, targetLang string) (string, error)
	Close() error
}

// GoogleCloudTranslator implements the Translator interface using Google Cloud Translation API.
type GoogleCloudTranslator struct {
	client *translate.Client
}

// NewGoogleCloudTranslator creates a new GoogleCloudTranslator instance.
// It configures the client based on debugMode and provided API key or default credentials.
func NewGoogleCloudTranslator(ctx context.Context, apiKey string) (*GoogleCloudTranslator, error) {
	var opts []option.ClientOption

	if apiKey != "" {
		// Use API Key directly if provided
		log.Printf("Initializing Google Cloud Translation client with API Key.")
		opts = append(opts, option.WithAPIKey(apiKey))
	} else {
		// Use GOOGLE_APPLICATION_CREDENTIALS or default credentials
		log.Printf("Initializing Google Cloud Translation client with default credentials.")
		// No explicit option needed for default credentials (ADC)
	}

	client, err := translate.NewClient(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create Google Cloud Translation client: %w", err)
	}
	return &GoogleCloudTranslator{client: client}, nil
}

// TranslateText translates the given text from sourceLang to targetLang using Google Cloud Translation API.
func (t *GoogleCloudTranslator) TranslateText(ctx context.Context, text, sourceLang, targetLang string) (string, error) {
	if t.client == nil {
		return "", fmt.Errorf("Google Cloud Translation client is not initialized")
	}

	if text == "" {
		return "", nil // Return empty string for empty input
	}

	target, err := language.Parse(targetLang)
	if err != nil {
		return "", fmt.Errorf("failed to parse target language: %w", err)
	}

	source, err := language.Parse(sourceLang)
	if err != nil {
		return "", fmt.Errorf("failed to parse source language: %w", err)
	}

	resp, err := t.client.Translate(ctx, []string{text}, target, &translate.Options{
		Source: source,
	})
	if err != nil {
		return "", fmt.Errorf("failed to translate text: %w", err)
	}

	if len(resp) == 0 {
		return "", fmt.Errorf("no translation results returned")
	}

	return resp[0].Text, nil
}

// Close closes the underlying client connection.
func (t *GoogleCloudTranslator) Close() error {
	if t.client != nil {
		return t.client.Close()
	}
	return nil
}

// MockTranslator implements the Translator interface for testing or local development.
type MockTranslator struct{}

// NewMockTranslator creates a new MockTranslator instance.
func NewMockTranslator() *MockTranslator {
	log.Println("Using MockTranslator.")
	return &MockTranslator{}
}

// TranslateText returns a mock translation.
func (t *MockTranslator) TranslateText(ctx context.Context, text, sourceLang, targetLang string) (string, error) {
	return fmt.Sprintf("Translated from %s to %s: %s", sourceLang, targetLang, text), nil
}

// Close does nothing for the mock translator.
func (t *MockTranslator) Close() error {
	return nil
}

// New selects the appropriate translator based on the configuration.
func New(ctx context.Context, debugMode bool, apiKey string) (Translator, error) {
	if debugMode {
		// If an API key is provided in debug mode, use the real translator.
		if apiKey != "" {
			return NewGoogleCloudTranslator(ctx, apiKey)
		}
		// Otherwise, use the mock translator.
		return NewMockTranslator(), nil
	}
	// In non-debug mode, always use the real translator.
	return NewGoogleCloudTranslator(ctx, apiKey)
}
