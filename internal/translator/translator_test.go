package translator

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	t.Run("with google translator type", func(t *testing.T) {
		// This will fail if you don't have Google Cloud credentials configured
		// We are not testing the actual client creation here, just the factory
		translator, err := New(context.Background(), "google", "dummy-key")
		assert.NoError(t, err)
		assert.IsType(t, &GoogleCloudTranslator{}, translator)
	})

	t.Run("with deepl translator type", func(t *testing.T) {
		translator, err := New(context.Background(), "deepl", "dummy-key")
		assert.NoError(t, err)
		assert.IsType(t, &DeepLTranslator{}, translator)
	})

	t.Run("with mock translator type", func(t *testing.T) {
		_, err := New(context.Background(), "mock", "")
		assert.Error(t, err)
		assert.EqualError(t, err, "mock translator can only be used in test environment")
	})

	t.Run("with unknown translator type", func(t *testing.T) {
		_, err := New(context.Background(), "unknown", "")
		assert.Error(t, err)
		assert.EqualError(t, err, "unknown translator type: unknown")
	})
}

func TestNewGoogleCloudTranslator(t *testing.T) {
	t.Run("with api key", func(t *testing.T) {
		translator, err := NewGoogleCloudTranslator(context.Background(), "dummy-key")
		assert.NoError(t, err)
		assert.NotNil(t, translator)
		assert.NotNil(t, translator.client)
	})

	t.Run("without api key and no credentials", func(t *testing.T) {
		// This test assumes no Google Cloud credentials are configured in the environment.
		_, err := NewGoogleCloudTranslator(context.Background(), "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create Google Cloud Translation client")
	})
}

func TestGoogleCloudTranslator_TranslateText(t *testing.T) {
	t.Run("with nil client", func(t *testing.T) {
		translator := &GoogleCloudTranslator{}
		_, err := translator.TranslateText(context.Background(), "hello", "en", "de")
		assert.Error(t, err)
		assert.EqualError(t, err, "Google Cloud Translation client is not initialized")
	})

	t.Run("with empty text", func(t *testing.T) {
		// We need a valid client to test this, but we can't create one without credentials.
		// So we create one with a dummy key, which will fail on translate, but not here.
		translator, err := NewGoogleCloudTranslator(context.Background(), "dummy-key")
		assert.NoError(t, err)
		result, err := translator.TranslateText(context.Background(), "", "en", "de")
		assert.NoError(t, err)
		assert.Equal(t, "", result)
	})

	t.Run("with invalid target language", func(t *testing.T) {
		translator, _ := NewGoogleCloudTranslator(context.Background(), "dummy-key")
		_, err := translator.TranslateText(context.Background(), "hello", "en", "invalid")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse target language")
	})

	t.Run("with invalid source language", func(t *testing.T) {
		translator, _ := NewGoogleCloudTranslator(context.Background(), "dummy-key")
		_, err := translator.TranslateText(context.Background(), "hello", "invalid", "de")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse source language")
	})
}

func TestGoogleCloudTranslator_Close(t *testing.T) {
	t.Run("with nil client", func(t *testing.T) {
		translator := &GoogleCloudTranslator{}
		err := translator.Close()
		assert.NoError(t, err)
	})

	t.Run("with non-nil client", func(t *testing.T) {
		translator, err := NewGoogleCloudTranslator(context.Background(), "dummy-key")
		assert.NoError(t, err)
		err = translator.Close()
		assert.NoError(t, err)
	})
}

func TestNewDeepLTranslator(t *testing.T) {
	t.Run("with api key", func(t *testing.T) {
		translator, err := NewDeepLTranslator("dummy-key")
		assert.NoError(t, err)
		assert.NotNil(t, translator)
		assert.NotNil(t, translator.client)
	})

	t.Run("without api key", func(t *testing.T) {
		_, err := NewDeepLTranslator("")
		assert.Error(t, err)
		assert.EqualError(t, err, "DeepL API key is required")
	})
}

func TestDeepLTranslator_TranslateText(t *testing.T) {
	t.Run("with nil client", func(t *testing.T) {
		translator := &DeepLTranslator{}
		_, err := translator.TranslateText(context.Background(), "hello", "en", "de")
		assert.Error(t, err)
		assert.EqualError(t, err, "DeepL client is not initialized")
	})

	t.Run("with empty text", func(t *testing.T) {
		translator, _ := NewDeepLTranslator("dummy-key")
		result, err := translator.TranslateText(context.Background(), "", "en", "de")
		assert.NoError(t, err)
		assert.Equal(t, "", result)
	})

	t.Run("with translation error", func(t *testing.T) {
		// This will cause an error because the API key is invalid
		translator, _ := NewDeepLTranslator("dummy-key")
		_, err := translator.TranslateText(context.Background(), "hello", "en", "de")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to translate text with DeepL")
	})
}

func TestDeepLTranslator_Close(t *testing.T) {
	translator, _ := NewDeepLTranslator("dummy-key")
	err := translator.Close()
	assert.NoError(t, err)
}
