package testutil

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

func SetupViper() {
	// Set GEMINI_TEST_ROOT to the current working directory (project root)
	// This is crucial for tests to find files like seed_data.json
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	// Ensure the path is the actual project root, not a subdirectory like tests/go_tests/models
	// We need to go up until we find a known project file like go.mod or README.md
	projectRoot := wd
	for {
		if _, err := os.Stat(filepath.Join(projectRoot, "go.mod")); err == nil {
			break // Found go.mod, this is likely the project root
		}
		parent := filepath.Dir(projectRoot)
		if parent == projectRoot { // Reached file system root
			panic("Could not find project root (go.mod not found in parent directories)")
		}
		projectRoot = parent
	}
	os.Setenv("GEMINI_TEST_ROOT", projectRoot)

	viper.SetConfigType("yaml")
	var yamlExample = []byte(`
routes:
  - path: "/"
    handler: "HelloHandler"
    methods:
      - "GET"
  - path: "/about"
    handler: "AboutHandler"
    methods:
      - "GET"
database:
  type: "sqlite"
  dsn: "file::memory:?cache=shared"
`)
	viper.ReadConfig(strings.NewReader(string(yamlExample)))
}