package main

import (
	"bytes"
	"crypto/rand"
	"errors"
	"fmt"
	"gemini-demo/internal/config"
	"gemini-demo/internal/testutil"
	"gemini-demo/internal/util"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gorilla/csrf"
)

// mockReader is a mock implementation of io.Reader for testing purposes.
type mockReader struct{}

func (r *mockReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("mock error")
}

func TestGenerateRandomKey(t *testing.T) {
	key, err := generateRandomKey(32, rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate random key: %v", err)
	}
	if len(key) != 64 {
		t.Errorf("expected key length of 64, but got %d", len(key))
	}
}

func TestGenerateRandomKey_Error(t *testing.T) {
	_, err := generateRandomKey(32, &mockReader{})
	if err == nil {
		t.Fatal("expected an error, but got nil")
	}
	if err.Error() != "mock error" {
		t.Errorf("expected error to be 'mock error', but got %v", err)
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
	loadConfig = func(filePath string) (*config.Config, error) {
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
	// This test checks if the application correctly handles a failure in loading translations.
	// We'll set up a test environment *without* the i18n data to trigger the error.

	// Create a temporary directory to act as the project root
	tmpDir, err := ioutil.TempDir("", "test-project-root-i18n-error-")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Set the environment variable to point to the temporary directory
	os.Setenv("GEMINI_TEST_ROOT", tmpDir)
	defer os.Unsetenv("GEMINI_TEST_ROOT")
	util.ResetProjectRootCacheForTesting()

	// We also need a valid config for this test
	oldLoadConfig := loadConfig
	defer func() { loadConfig = oldLoadConfig }()
	loadConfig = func(filePath string) (*config.Config, error) {
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

	err = run([]string{"-debug"})
	if err == nil {
		t.Fatal("expected an error, but got nil")
	}

	if !strings.Contains(err.Error(), "Failed to load translations") {
		t.Errorf("expected error to contain 'Failed to load translations', but got %v", err)
	}
}

func TestRun_TranslatorError(t *testing.T) {
	_, cleanup, err := testutil.SetupTestEnv(t)
	if err != nil {
		t.Fatalf("SetupTestEnv failed: %v", err)
	}
	defer cleanup()

	oldLoadConfig := loadConfig
	defer func() { loadConfig = oldLoadConfig }()
	loadConfig = func(filePath string) (*config.Config, error) {
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

	err = run([]string{"-debug"})
	if err == nil {
		t.Fatal("expected an error, but got nil")
	}

	if !strings.Contains(err.Error(), "Failed to create translator") {
		t.Errorf("expected error to contain 'Failed to create translator', but got %v", err)
	}
}

func TestRun_DatabaseError(t *testing.T) {
	_, cleanup, err := testutil.SetupTestEnv(t)
	if err != nil {
		t.Fatalf("SetupTestEnv failed: %v", err)
	}
	defer cleanup()

	oldLoadConfig := loadConfig
	defer func() { loadConfig = oldLoadConfig }()
	loadConfig = func(filePath string) (*config.Config, error) {
		return &config.Config{
			I18n: config.I18nConfig{
				DefaultLanguage: "en",
			},
			Auth: config.AuthConfig{
				SessionKey: "12345678901234567890123456789012",
				CSRFKey:    "12345678901234567890123456789012",
			},
			Translator: config.TranslatorConfig{
				Type:   "deepl", // a valid translator type
				APIKey: "dummy-key",
			},
			Database: config.DatabaseConfig{
				Type: "invalid-database",
			},
		}, nil
	}

	err = run([]string{"-debug"})
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

func TestRun_DebugMissingSessionKeyError(t *testing.T) {
	_, cleanup, err := testutil.SetupTestEnv(t)
	if err != nil {
		t.Fatalf("SetupTestEnv failed: %v", err)
	}
	defer cleanup()

	oldLoadConfig := loadConfig
	defer func() { loadConfig = oldLoadConfig }()
	loadConfig = func(filePath string) (*config.Config, error) {
		return &config.Config{
			Auth: config.AuthConfig{
				SessionKey: "", // Missing session key
				CSRFKey:    "a-valid-csrf-key-of-sufficient-length",
			},
		}, nil
	}

	err = run([]string{"-debug"})
	if err == nil {
		t.Fatal("expected an error, but got nil")
	}
	if !strings.Contains(err.Error(), "Session key not found") {
		t.Errorf("expected error about session key, but got %v", err)
	}
}

func TestRun_DebugMissingCSRFKeyError(t *testing.T) {
	_, cleanup, err := testutil.SetupTestEnv(t)
	if err != nil {
		t.Fatalf("SetupTestEnv failed: %v", err)
	}
	defer cleanup()

	oldLoadConfig := loadConfig
	defer func() { loadConfig = oldLoadConfig }()
	loadConfig = func(filePath string) (*config.Config, error) {
		return &config.Config{
			Auth: config.AuthConfig{
				SessionKey: "a-valid-session-key-of-sufficient-length",
				CSRFKey:    "", // Missing CSRF key
			},
		}, nil
	}

	err = run([]string{"-debug"})
	if err == nil {
		t.Fatal("expected an error, but got nil")
	}
	if !strings.Contains(err.Error(), "CSRF key not found") {
		t.Errorf("expected error about CSRF key, but got %v", err)
	}
}

func TestRun_CSRFKeyTooShortError(t *testing.T) {
	_, cleanup, err := testutil.SetupTestEnv(t)
	if err != nil {
		t.Fatalf("SetupTestEnv failed: %v", err)
	}
	defer cleanup()

	oldLoadConfig := loadConfig
	defer func() { loadConfig = oldLoadConfig }()
	loadConfig = func(filePath string) (*config.Config, error) {
		return &config.Config{
			I18n: config.I18nConfig{DefaultLanguage: "en"},
			Auth: config.AuthConfig{
				SessionKey: "a-valid-session-key-of-sufficient-length",
				CSRFKey:    "too-short",
			},
			Translator: config.TranslatorConfig{
				Type:   "deepl",
				APIKey: "dummy-key-for-test",
			},
			Database: config.DatabaseConfig{Type: "sqlite", DSN: ":memory:"},
		}, nil
	}

	// Need to mock the database part to avoid it failing before the CSRF check
	err = run([]string{"-debug"})
	if err == nil {
		t.Fatal("expected an error, but got nil")
	}
	if !strings.Contains(err.Error(), "CSRF key must be at least 32 bytes long") {
		t.Errorf("expected error about CSRF key length, but got %v", err)
	}
}

func TestRun_AutoMigrateAndSeedError(t *testing.T) {
	// Create a temporary read-only database file
	tmpfile, err := ioutil.TempFile("", "test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())
	tmpfile.Close()
	os.Chmod(tmpfile.Name(), 0444) // read-only

	_, cleanup, err := testutil.SetupTestEnv(t)
	if err != nil {
		t.Fatalf("SetupTestEnv failed: %v", err)
	}
	defer cleanup()

	oldLoadConfig := loadConfig
	defer func() { loadConfig = oldLoadConfig }()
	loadConfig = func(filePath string) (*config.Config, error) {
		return &config.Config{
			I18n: config.I18nConfig{DefaultLanguage: "en"},
			Auth: config.AuthConfig{
				SessionKey: "12345678901234567890123456789012",
				CSRFKey:    "12345678901234567890123456789012",
			},
			Translator: config.TranslatorConfig{Type: "deepl", APIKey: "dummy-key"},
			Database:   config.DatabaseConfig{Type: "sqlite", DSN: tmpfile.Name()},
		}, nil
	}

	err = run([]string{"-debug"})
	if err == nil {
		t.Fatal("expected an error, but got nil")
	}
	if !strings.Contains(err.Error(), "failed to auto migrate and seed models") {
		t.Errorf("expected error about auto migrate and seed, but got %v", err)
	}
}

func TestRun_ParseTemplatesError(t *testing.T) {
	// Create a temporary directory to act as the project root
	tmpDir, err := ioutil.TempDir("", "test-project-root-")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Set the environment variable to point to the temporary directory
	os.Setenv("GEMINI_TEST_ROOT", tmpDir)
	defer os.Unsetenv("GEMINI_TEST_ROOT")
	util.ResetProjectRootCacheForTesting()

	// Create dummy data directories and files
	dataDir := filepath.Join(tmpDir, "data")
	os.MkdirAll(dataDir, 0755)
	i18nDir := filepath.Join(dataDir, "i18n")
	os.MkdirAll(i18nDir, 0755)
	seedDataPath := filepath.Join(dataDir, "seed_data.json")
	ioutil.WriteFile(seedDataPath, []byte("{}"), 0644)

	// We also need a valid config for this test
	oldLoadConfig := loadConfig
	defer func() { loadConfig = oldLoadConfig }()
	loadConfig = func(filePath string) (*config.Config, error) {
		return &config.Config{
			I18n: config.I18nConfig{
				DefaultLanguage: "en",
			},
			Auth: config.AuthConfig{
				SessionKey: "12345678901234567890123456789012",
				CSRFKey:    "12345678901234567890123456789012",
			},
			Translator: config.TranslatorConfig{
				Type:   "deepl",
				APIKey: "dummy-key",
			},
			Database: config.DatabaseConfig{
				Type: "sqlite",
				DSN:  ":memory:",
			},
		}, nil
	}

	err = run([]string{"-debug"})
	if err == nil {
		t.Fatal("expected an error, but got nil")
	}

	if !strings.Contains(err.Error(), "failed to parse templates") {
		t.Errorf("expected error to contain 'failed to parse templates', but got %v", err)
	}
}

func TestLogRequestMiddleware(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	debugFlag = true
	defer func() {
		log.SetOutput(os.Stderr)
		debugFlag = false
	}()

	// Create a dummy handler
	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Create the middleware
	middleware := logRequestMiddleware(dummyHandler)

	// Create a request and response recorder
	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	// Call the middleware
	middleware.ServeHTTP(rr, req)

	// Check the log output
	if !strings.Contains(buf.String(), "Request before CSRF") {
		t.Errorf("expected log to contain 'Request before CSRF', but it didn't")
	}
}

func TestRun_ListenAndServeError(t *testing.T) {
	_, cleanup, err := testutil.SetupTestEnv(t)
	if err != nil {
		t.Fatalf("SetupTestEnv failed: %v", err)
	}
	defer cleanup()

	oldLoadConfig := loadConfig
	defer func() { loadConfig = oldLoadConfig }()
	loadConfig = func(filePath string) (*config.Config, error) {
		cfg := &config.Config{
			I18n: config.I18nConfig{DefaultLanguage: "en"},
			Auth: config.AuthConfig{
				SessionKey: "12345678901234567890123456789012",
				CSRFKey:    "12345678901234567890123456789012",
			},
			Translator: config.TranslatorConfig{Type: "deepl", APIKey: "dummy-key"},
			Database:   config.DatabaseConfig{Type: "sqlite", DSN: ":memory:"},
		}
		cfg.Server.Address = "invalid address"
		return cfg, nil
	}

	err = run([]string{"-debug"})
	if err == nil {
		t.Fatal("expected an error, but got nil")
	}
	if !strings.Contains(err.Error(), "invalid address") {
		t.Errorf("expected error about invalid address, but got %v", err)
	}
}

func TestRun_HTTPS(t *testing.T) {
	_, cleanup, err := testutil.SetupTestEnv(t)
	if err != nil {
		t.Fatalf("SetupTestEnv failed: %v", err)
	}
	defer cleanup()

	oldLoadConfig := loadConfig
	defer func() { loadConfig = oldLoadConfig }()
	loadConfig = func(filePath string) (*config.Config, error) {
		cfg := &config.Config{
			I18n: config.I18nConfig{DefaultLanguage: "en"},
			Auth: config.AuthConfig{
				SessionKey: "12345678901234567890123456789012",
				CSRFKey:    "12345678901234567890123456789012",
			},
			Translator: config.TranslatorConfig{Type: "deepl", APIKey: "dummy-key"},
			Database:   config.DatabaseConfig{Type: "sqlite", DSN: ":memory:"},
		}
		cfg.Server.Address = "https://localhost:8443"
		return cfg, nil
	}

	// We expect this to fail because we don't have certs, but it will test the https path
	err = run([]string{"-debug"})
	if err == nil {
		t.Fatal("expected an error, but got nil")
	}
}

func TestRun_DotEnvError(t *testing.T) {
	// Capture log output
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	// Mock godotenv.Load to return an error
	oldGodotenvLoad := godotenvLoad
	defer func() { godotenvLoad = oldGodotenvLoad }()
	godotenvLoad = func(filenames ...string) (err error) {
		return fmt.Errorf("mock dotenv error")
	}

	// We expect the run to continue, but log an error.
	// We need to mock the rest of the run function to avoid other errors.
	oldLoadConfig := loadConfig
	defer func() { loadConfig = oldLoadConfig }()
	loadConfig = func(filePath string) (*config.Config, error) {
		return nil, fmt.Errorf("mock config error")
	}

	run([]string{})

	logOutput := buf.String()
	if !strings.Contains(logOutput, "Error loading .env file") {
		t.Errorf("expected log to contain 'Error loading .env file', but it didn't. Log: %s", logOutput)
	}
}

func TestRun_Success(t *testing.T) {
	_, cleanup, err := testutil.SetupTestEnv(t)
	if err != nil {
		t.Fatalf("SetupTestEnv failed: %v", err)
	}
	defer cleanup()

	oldLoadConfig := loadConfig
	defer func() { loadConfig = oldLoadConfig }()
	loadConfig = func(filePath string) (*config.Config, error) {
		cfg := &config.Config{
			I18n: config.I18nConfig{DefaultLanguage: "en"},
			Auth: config.AuthConfig{
				SessionKey: "12345678901234567890123456789012",
				CSRFKey:    "12345678901234567890123456789012",
			},
			Translator: config.TranslatorConfig{Type: "deepl", APIKey: "dummy-key"},
			Database:   config.DatabaseConfig{Type: "sqlite", DSN: ":memory:"},
		}
		cfg.Server.Address = "localhost:8080"
		return cfg, nil
	}

	// Mock the server so it doesn't actually start
	oldListenAndServe := listenAndServe
	defer func() { listenAndServe = oldListenAndServe }()
	listenAndServe = func(srv *http.Server) error {
		// Return nil to simulate a successful server start that is immediately closed.
		return nil
	}

	err = run([]string{"-debug"})
	if err != nil {
		t.Fatalf("expected no error, but got %v", err)
	}
}

func TestCSRFErrorHandler(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	debugFlag = true
	defer func() {
		log.SetOutput(os.Stderr)
		debugFlag = false
	}()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Create a request that will fail CSRF
	req := httptest.NewRequest("POST", "/", nil)
	rr := httptest.NewRecorder()

	csrfKey := "12345678901234567890123456789012"
	csrfMiddleware := csrf.Protect(
		[]byte(csrfKey),
		csrf.ErrorHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			debugLog("CSRF Error: Handler triggered for request to %s", r.URL.Path)
			if err := csrf.FailureReason(r); err != nil {
				debugLog("CSRF Error: Failure Reason: %v", err)
			}
			w.WriteHeader(http.StatusForbidden)
		})),
	)

	csrfMiddleware(handler).ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected status code %d, but got %d", http.StatusForbidden, rr.Code)
	}

	logOutput := buf.String()
	if !strings.Contains(logOutput, "CSRF Error: Handler triggered") {
		t.Errorf("expected log to contain 'CSRF Error: Handler triggered', but it didn't")
	}
	if !strings.Contains(logOutput, "CSRF Error: Failure Reason:") {
		t.Errorf("expected log to contain 'CSRF Error: Failure Reason:', but it didn't. Log: %s", logOutput)
	}
}

func TestRun_HTTP(t *testing.T) {
	_, cleanup, err := testutil.SetupTestEnv(t)
	if err != nil {
		t.Fatalf("SetupTestEnv failed: %v", err)
	}
	defer cleanup()

	oldListenAndServe := listenAndServe
	defer func() { listenAndServe = oldListenAndServe }()
	listenAndServe = func(srv *http.Server) error {
		return http.ErrServerClosed
	}

	oldLoadConfig := loadConfig
	defer func() { loadConfig = oldLoadConfig }()
	loadConfig = func(filePath string) (*config.Config, error) {
		cfg := &config.Config{
			I18n: config.I18nConfig{DefaultLanguage: "en"},
			Auth: config.AuthConfig{
				SessionKey: "12345678901234567890123456789012",
				CSRFKey:    "12345678901234567890123456789012",
			},
			Translator: config.TranslatorConfig{Type: "deepl", APIKey: "dummy-key"},
			Database:   config.DatabaseConfig{Type: "sqlite", DSN: ":memory:"},
		}
		cfg.Server.Address = "http://localhost:8080" // Note the http scheme
		return cfg, nil
	}

	// We expect this to fail because we are not actually starting a server,
	// but it will test the http path for the wrapper.
	err = run([]string{"-debug"})
	if err != http.ErrServerClosed {
		t.Fatalf("expected http.ErrServerClosed, but got %v", err)
	}
}

func TestRun_EmptyTrustedOriginsWarning(t *testing.T) {
	_, cleanup, err := testutil.SetupTestEnv(t)
	if err != nil {
		t.Fatalf("SetupTestEnv failed: %v", err)
	}
	defer cleanup()

	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	oldLoadConfig := loadConfig
	defer func() { loadConfig = oldLoadConfig }()
	loadConfig = func(filePath string) (*config.Config, error) {
		cfg := &config.Config{
			I18n: config.I18nConfig{DefaultLanguage: "en"},
			Auth: config.AuthConfig{
				SessionKey: "12345678901234567890123456789012",
				CSRFKey:    "12345678901234567890123456789012",
				TrustedOrigins: []string{}, // Empty trusted origins
			},
			Translator: config.TranslatorConfig{Type: "deepl", APIKey: "dummy-key"},
			Database:   config.DatabaseConfig{Type: "sqlite", DSN: ":memory:"},
		}
		cfg.Server.Address = "localhost:8080"
		return cfg, nil
	}

	oldListenAndServe := listenAndServe
	defer func() { listenAndServe = oldListenAndServe }()
	listenAndServe = func(srv *http.Server) error {
		return nil
	}

	err = run([]string{"-debug"})
	if err != nil {
		t.Fatalf("expected no error, but got %v", err)
	}

	logOutput := buf.String()
	if !strings.Contains(logOutput, "Warning: No CSRF trusted origins configured") {
		t.Errorf("expected log to contain warning about empty trusted origins, but it didn't. Log: %s", logOutput)
	}
}

func TestRun_MissingCSRFKeyError_NoDebug(t *testing.T) {
	_, cleanup, err := testutil.SetupTestEnv(t)
	if err != nil {
		t.Fatalf("SetupTestEnv failed: %v", err)
	}
	defer cleanup()

	oldLoadConfig := loadConfig
	defer func() { loadConfig = oldLoadConfig }()
	loadConfig = func(filePath string) (*config.Config, error) {
		return &config.Config{
			Auth: config.AuthConfig{
				SessionKey: "a-valid-session-key-of-sufficient-length",
				CSRFKey:    "", // Missing CSRF key
			},
			Translator: config.TranslatorConfig{Type: "deepl", APIKey: "dummy-key"},
			Database:   config.DatabaseConfig{Type: "sqlite", DSN: ":memory:"},
			I18n:       config.I18nConfig{DefaultLanguage: "en"},
		}, nil
	}

	err = run([]string{}) // No -debug flag
	if err == nil {
		t.Fatal("expected an error, but got nil")
	}
	if !strings.Contains(err.Error(), "CSRF key not found in config") {
		t.Errorf("expected error about CSRF key, but got %v", err)
	}
}

func TestMain_RunError(t *testing.T) {
	if os.Getenv("BE_CRASHER") == "1" {
		// Mock run to return an error
		originalRun := run
		run = func(args []string) error {
			return fmt.Errorf("mock run error")
		}
		defer func() { run = originalRun }()

		main()
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestMain_RunError")
	cmd.Env = append(os.Environ(), "BE_CRASHER=1")
	var out bytes.Buffer
	cmd.Stderr = &out
	err := cmd.Run()

	// Check that the command exited with a non-zero status code
	if e, ok := err.(*exec.ExitError); ok && !e.Success() {
		if !strings.Contains(out.String(), "mock run error") {
			t.Errorf("expected stderr to contain 'mock run error', but got %q", out.String())
		}
		return
	}
	t.Fatalf("process ran with err %v, want exit status 1", err)
}