package translator

import (
	"context"
	"fmt"
	"log"

	"cloud.google.com/go/translate"
	"github.com/lkretschmer/deepl-go"
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

// DeepLTranslator implements the Translator interface using the DeepL API.
type DeepLTranslator struct {
	client *deepl.Client
}

// NewDeepLTranslator creates a new DeepLTranslator instance.
func NewDeepLTranslator(apiKey string) (*DeepLTranslator, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("DeepL API key is required")
	}
	log.Println("Initializing DeepL client.")
	client := deepl.NewClient(apiKey)
	return &DeepLTranslator{client: client}, nil
}

// TranslateText translates the given text from sourceLang to targetLang using the DeepL API.
func (t *DeepLTranslator) TranslateText(ctx context.Context, text, sourceLang, targetLang string) (string, error) {
	if t.client == nil {
		return "", fmt.Errorf("DeepL client is not initialized")
	}

	if text == "" {
		return "", nil // Return empty string for empty input
	}

	translation, err := t.client.TranslateText(text, targetLang)
	if err != nil {
		return "", fmt.Errorf("failed to translate text with DeepL: %w", err)
	}

	return translation.Text, nil
}

// Close does nothing for the DeepL translator as the underlying client does not need to be closed.
func (t *DeepLTranslator) Close() error {
	return nil
}



// New selects the appropriate translator based on the configuration.
func New(ctx context.Context, translatorType, apiKey string) (Translator, error) {
	switch translatorType {
	case "google":
		return NewGoogleCloudTranslator(ctx, apiKey)
	case "deepl":
		return NewDeepLTranslator(apiKey)
	case "mock":
		return nil, fmt.Errorf("mock translator can only be used in test environment")
	default:
		return nil, fmt.Errorf("unknown translator type: %s", translatorType)
	}
}
