package main

import (
	"bytes"
	"fmt"
	"gemini-demo/internal/config"
	"gemini-demo/internal/util"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"testing"
)

func TestGenerateRandomKey(t *testing.T) {
	key, err := generateRandomKey(32)
	if err != nil {
		t.Fatalf("failed to generate random key: %v", err)
	}
	if len(key) != 64 {
		t.Errorf("expected key length of 64, but got %d", len(key))
	}
}

func TestDebugLog(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	debugFlag = true
	debugLog("test message")
	log.SetOutput(os.Stderr)

	if !strings.Contains(buf.String(), "test message") {
		t.Errorf("expected log to contain 'test message', but it didn't")
	}

	buf.Reset()
	log.SetOutput(&buf)
	debugFlag = false
	debugLog("another test message")
	log.SetOutput(os.Stderr)

	if buf.String() != "" {
		t.Errorf("expected log to be empty, but got '%s'", buf.String())
	}
}

func TestRun_ConfigError(t *testing.T) {
	oldLoadConfig := loadConfig
	defer func() { loadConfig = oldLoadConfig }()
	loadConfig = func() (*config.Config, error) {
		return nil, fmt.Errorf("mock config error")
	}

	err := run([]string{})
	if err == nil {
		t.Fatal("expected an error, but got nil")
	}

	if !strings.Contains(err.Error(), "mock config error") {
		t.Errorf("expected error to contain 'mock config error', but got %v", err)
	}
}

func TestRun_i18nError(t *testing.T) {
	// Set debug flag to true to trigger auth key checks
	debugFlag = true
	defer func() { debugFlag = false }()

	// Create a temporary directory to act as the project root
	tmpDir, err := ioutil.TempDir("", "test-project-root-")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Set the environment variable to point to the temporary directory
	os.Setenv("GEMINI_TEST_ROOT", tmpDir)
	defer os.Unsetenv("GEMINI_TEST_ROOT")

	// Reset the project root cache to ensure the new env var is used
	util.ResetProjectRootCacheForTesting()

	// We also need a valid config for this test
	oldLoadConfig := loadConfig
	defer func() { loadConfig = oldLoadConfig }()
	loadConfig = func() (*config.Config, error) {
		return &config.Config{
			I18n: config.I18nConfig{
				DefaultLanguage: "en",
			},
			Auth: config.AuthConfig{
				SessionKey: "12345678901234567890123456789012",
				CSRFKey:    "12345678901234567890123456789012",
			},
		}, nil
	}

	err = run([]string{})
	if err == nil {
		t.Fatal("expected an error, but got nil")
	}

	if !strings.Contains(err.Error(), "Failed to load translations") {
		t.Errorf("expected error to contain 'Failed to load translations', but got %v", err)
	}
}

func TestRun_TranslatorError(t *testing.T) {
	// Set debug flag to true to trigger auth key checks
	debugFlag = true
	defer func() { debugFlag = false }()

	// We need a valid i18n setup for this test
	// So we use the real project root
	util.ResetProjectRootCacheForTesting()

	oldLoadConfig := loadConfig
	defer func() { loadConfig = oldLoadConfig }()
	loadConfig = func() (*config.Config, error) {
		return &config.Config{
			I18n: config.I18nConfig{
				DefaultLanguage: "en",
			},
			Auth: config.AuthConfig{
				SessionKey: "12345678901234567890123456789012",
				CSRFKey:    "12345678901234567890123456789012",
			},
			Translator: config.TranslatorConfig{
				Type: "invalid-translator",
			},
		}, nil
	}

	err := run([]string{})
	if err == nil {
		t.Fatal("expected an error, but got nil")
	}

	if !strings.Contains(err.Error(), "Failed to create translator") {
		t.Errorf("expected error to contain 'Failed to create translator', but got %v", err)
	}
}

func TestRun_DatabaseError(t *testing.T) {
	// Set debug flag to true to trigger auth key checks
	debugFlag = true
	defer func() { debugFlag = false }()

	// We need a valid i18n and translator setup for this test
	util.ResetProjectRootCacheForTesting()

	oldLoadConfig := loadConfig
	defer func() { loadConfig = oldLoadConfig }()
	loadConfig = func() (*config.Config, error) {
		return &config.Config{
			I18n: config.I18nConfig{
				DefaultLanguage: "en",
			},
			Auth: config.AuthConfig{
				SessionKey: "12345678901234567890123456789012",
				CSRFKey:    "12345678901234567890123456789012",
			},
			Translator: config.TranslatorConfig{
				Type: "noop", // a valid translator type
			},
			Database: config.DatabaseConfig{
				Type: "invalid-database",
			},
		}, nil
	}

	err := run([]string{})
	if err == nil {
		t.Fatal("expected an error, but got nil")
	}

	if !strings.Contains(err.Error(), "failed to initialize database") {
		t.Errorf("expected error to contain 'failed to initialize database', but got %v", err)
	}
}

func TestRun_FlagParseError(t *testing.T) {
	err := run([]string{"-invalid-flag"})
	if err == nil {
		t.Fatal("expected an error, but got nil")
	}
	if !strings.Contains(err.Error(), "flag provided but not defined: -invalid-flag") {
		t.Errorf("expected error to contain 'flag provided but not defined: -invalid-flag', but got %v", err)
	}
}
