package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// createTempConfigFile is a helper function to create a temporary config file for testing.
func createTempConfigFile(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	filePath := filepath.Join(dir, "config.yml")
	err := os.WriteFile(filePath, []byte(content), 0644)
	assert.NoError(t, err)
	return filePath
}

func TestLoadConfig(t *testing.T) {
	t.Run("with valid config file and env vars", func(t *testing.T) {
		configContent := `
server:
  address: ":8080"
database:
  type: "sqlite"
  dsn: "test.db"
auth:
  session_key: "file_session_key"
  csrf_key: "file_csrf_key"
`
		filePath := createTempConfigFile(t, configContent)

		// Set environment variables that should override file values
		t.Setenv("SESSION_KEY", "env_session_key")
		t.Setenv("CSRF_KEY", "env_csrf_key")
		t.Setenv("ADMIN_USERNAME", "env_admin_user")
		t.Setenv("ADMIN_PASSWORD", "env_admin_pass")
		t.Setenv("TRANSLATOR_API_KEY", "env_translator_api_key")
		t.Setenv("TRANSLATOR_TYPE", "env_translator_type")
		t.Setenv("I18N_DEFAULT_LANGUAGE", "env_i18n_lang")

		cfg, err := LoadConfig(filePath)

		// Assertions
		assert.NoError(t, err)
		assert.NotNil(t, cfg)
		assert.Equal(t, ":8080", cfg.Server.Address)
		assert.Equal(t, "sqlite", cfg.Database.Type)
		assert.Equal(t, "test.db", cfg.Database.DSN)
		assert.Equal(t, "env_session_key", cfg.Auth.SessionKey)
		assert.Equal(t, "env_csrf_key", cfg.Auth.CSRFKey)
		assert.Equal(t, "env_translator_api_key", cfg.Translator.APIKey)
		assert.Equal(t, "env_translator_type", cfg.Translator.Type)
		assert.Equal(t, "env_i18n_lang", cfg.I18n.DefaultLanguage)
	})

	t.Run("when config file is invalid yaml", func(t *testing.T) {
		filePath := createTempConfigFile(t, `server: address: ":8080"`)
		_, err := LoadConfig(filePath)
		assert.Error(t, err)
	})

	t.Run("when config file has unmarshal error", func(t *testing.T) {
		filePath := createTempConfigFile(t, "server: 1234")
		_, err := LoadConfig(filePath)
		assert.Error(t, err)
	})

	t.Run("when config file not found but env vars are set", func(t *testing.T) {
		// Set environment variables
		t.Setenv("SESSION_KEY", "env_session_key_only")
		t.Setenv("CSRF_KEY", "env_csrf_key_only")
		t.Setenv("TRANSLATOR_API_KEY", "env_translator_api_key_only")
		t.Setenv("TRANSLATOR_TYPE", "env_translator_type_only")
		t.Setenv("I18N_DEFAULT_LANGUAGE", "env_i18n_lang_only")
		t.Setenv("DATABASE_DSN", "env_database_dsn")

		// Pass a non-existent file path. LoadConfig should ignore the not-found error
		// and proceed to load from environment variables.
		cfg, err := LoadConfig("non-existent-config.yml")

		assert.NoError(t, err)
		assert.NotNil(t, cfg)
		assert.Equal(t, "env_session_key_only", cfg.Auth.SessionKey)
		assert.Equal(t, "env_csrf_key_only", cfg.Auth.CSRFKey)
		assert.Equal(t, "env_translator_api_key_only", cfg.Translator.APIKey)
		assert.Equal(t, "env_translator_type_only", cfg.Translator.Type)
		assert.Equal(t, "env_i18n_lang_only", cfg.I18n.DefaultLanguage)
		assert.Equal(t, "env_database_dsn", cfg.Database.DSN)
	})

	t.Run("when loading config from env vars with replacer", func(t *testing.T) {
		// Set environment variables that rely on the replacer
		t.Setenv("SERVER_ADDRESS", "localhost:9090")
		t.Setenv("ADMIN_USERNAME", "test_admin")
		t.Setenv("ADMIN_PASSWORD", "test_password")
		t.Setenv("DATABASE_TYPE", "postgres")

		// Pass an empty path, which tells LoadConfig to only use env vars.
		cfg, err := LoadConfig("")

		assert.NoError(t, err)
		assert.NotNil(t, cfg)
		assert.Equal(t, "localhost:9090", cfg.Server.Address)
		assert.Equal(t, "postgres", cfg.Database.Type)
		assert.Equal(t, "test_admin", cfg.Auth.Username)
		assert.Equal(t, "test_password", cfg.Auth.Password)
	})

	t.Run("with no config file and no env vars", func(t *testing.T) {
		// Temporarily unset environment variables to ensure a clean test.
		// t.Setenv automatically restores the original values after the test.
		t.Setenv("SESSION_KEY", "")
		t.Setenv("CSRF_KEY", "")
		t.Setenv("ADMIN_USERNAME", "")
		t.Setenv("ADMIN_PASSWORD", "")
		t.Setenv("TRANSLATOR_API_KEY", "")
		t.Setenv("TRANSLATOR_TYPE", "")
		t.Setenv("I18N_DEFAULT_LANGUAGE", "")
		t.Setenv("DATABASE_DSN", "")
		t.Setenv("SERVER_ADDRESS", "")
		t.Setenv("DATABASE_TYPE", "")

		// Pass an empty path, which tells LoadConfig to only use env vars.
		cfg, err := LoadConfig("")

		assert.NoError(t, err)
		assert.NotNil(t, cfg)

		// Assert that the fields have their zero values
		assert.Equal(t, "", cfg.Server.Address)
		assert.Equal(t, "", cfg.Database.Type)
		assert.Equal(t, "", cfg.Database.DSN)
		assert.Equal(t, "", cfg.Auth.SessionKey)
		assert.Equal(t, "", cfg.Auth.CSRFKey)
		assert.Equal(t, "", cfg.Translator.APIKey)
		assert.Equal(t, "", cfg.Translator.Type)
		assert.Equal(t, "", cfg.I18n.DefaultLanguage)
		assert.False(t, cfg.DebugMode)
	})
}
