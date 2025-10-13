package util

import (
	"fmt"
	"gemini-demo/internal/i18n"
	"html/template"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ProjectRoot returns the project's root directory.
// It is designed to be safe for concurrent use.
func ProjectRoot(testExecutablePath string) string {
	if testRoot := os.Getenv("GEMINI_TEST_ROOT"); testRoot != "" {
		return testRoot
	}

	var startDir string
	var err error

	if testExecutablePath != "" {
		startDir = filepath.Dir(testExecutablePath) // For testing, use the directory of the provided executable path
	} else {
		// Use current working directory as the starting point
		startDir, err = os.Getwd()
		if err != nil {
			panic(err) // Or handle error more gracefully
		}
	}

	// Search upwards from the starting directory for a known project root marker (e.g., go.mod)
	currentDir := startDir
	for {
		if _, err := os.Stat(filepath.Join(currentDir, "go.mod")); err == nil {
			return currentDir
		}
		parent := filepath.Dir(currentDir)
		if parent == currentDir {
			break // Reached root directory, go.mod not found
		}
		currentDir = parent
	}

	// Fallback if go.mod is not found, with special handling for 'bin' directory
	if strings.HasSuffix(filepath.ToSlash(startDir), "/bin") {
		return filepath.Dir(startDir)
	}

	return startDir // Fallback to the starting directory
}

// ResetProjectRootCacheForTesting is now a no-op because the cache has been removed.
// This function is kept for backward compatibility with existing test code.
func ResetProjectRootCacheForTesting() {
	// No-op
}

func ParseTemplates(translator *i18n.Translator, projectRoot ...string) (map[string]*template.Template, error) {
	var actualProjectRoot string
	if len(projectRoot) > 0 && projectRoot[0] != "" {
		actualProjectRoot = projectRoot[0]
	} else {
		actualProjectRoot = ProjectRoot("")
	}

	templates := make(map[string]*template.Template)
	templateDir := filepath.Join(actualProjectRoot, "templates")

	// 1. Read all files into memory.
	rawTemplates := make(map[string]string)
	err := filepath.Walk(templateDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if strings.HasSuffix(info.Name(), ".html") {
			if info.IsDir() {
				return fmt.Errorf("template entry is a directory: %s", path)
			}
			relPath, _ := filepath.Rel(templateDir, path)
			relPath = strings.ReplaceAll(relPath, "\\", "/")
			content, err := os.ReadFile(path)
			if err != nil {
				return err // Propagate the error
			}
			rawTemplates[relPath] = string(content)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("error walking templates: %w", err)
	}

		funcMap := template.FuncMap{
		"T": func(lang, key string) template.HTML {
			return template.HTML(translator.GetTranslation(lang, key))
		},
		"hasPrefix": strings.HasPrefix,
	}

	// 2. For each file, create its own isolated template set.
	for name, content := range rawTemplates {
		tmpl := template.New(name).Funcs(funcMap)

		// To ensure {{define}} blocks are correctly overridden, we must parse the main template's content
		// LAST. However, we must also ensure that all other templates are available for inclusion.
		// A predictable parsing order is needed to reason about this.
		var otherNames []string
		for otherName := range rawTemplates {
			if name != otherName {
				otherNames = append(otherNames, otherName)
			}
		}
		sort.Strings(otherNames) // Sort for predictable parsing order

		for _, otherName := range otherNames {
			if _, err := tmpl.New(otherName).Parse(rawTemplates[otherName]); err != nil {
				return nil, fmt.Errorf("parsing sub-template %s for %s: %w", otherName, name, err)
			}
		}

		// Now, parse the main template's content. This will overwrite any conflicting {{define}} blocks.
		if _, err := tmpl.Parse(content); err != nil {
			return nil, fmt.Errorf("parsing main template %s: %w", name, err)
		}

		templates[name] = tmpl
	}

	if len(templates) == 0 {
		return nil, fmt.Errorf("no templates found")
	}

	return templates, nil
}