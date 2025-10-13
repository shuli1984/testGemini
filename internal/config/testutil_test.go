package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestSetupViper(t *testing.T) {
	t.Run("successfully sets up viper and env var", func(t *testing.T) {
		// Reset viper and environment for a clean test
		viper.Reset()
		originalEnv := os.Getenv("GEMINI_TEST_ROOT")
		os.Unsetenv("GEMINI_TEST_ROOT")
		defer os.Setenv("GEMINI_TEST_ROOT", originalEnv)

		// Call the function to test
		SetupViper()

		// 1. Check if GEMINI_TEST_ROOT is set correctly
		wd, err := os.Getwd()
		assert.NoError(t, err)
		projectRoot := wd
		for {
			if _, err := os.Stat(filepath.Join(projectRoot, "go.mod")); err == nil {
				break
			}
			parent := filepath.Dir(projectRoot)
			if parent == projectRoot {
				t.Fatal("could not find project root to verify GEMINI_TEST_ROOT")
			}
			projectRoot = parent
		}
		assert.Equal(t, projectRoot, os.Getenv("GEMINI_TEST_ROOT"))

		// 2. Check if viper has loaded the default config
		assert.Equal(t, "sqlite", viper.GetString("database.type"))
		assert.Equal(t, "HelloHandler", viper.GetString("routes.0.handler"))
	})
}
