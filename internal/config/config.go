package config

import (
	"strings"

	"github.com/spf13/viper"
)

// Route defines the structure for a single route in the config.
type Route struct {
	Path         string   `mapstructure:"path"`
	Handler      string   `mapstructure:"handler"`
	Methods      []string `mapstructure:"methods"`
	AuthRequired bool     `mapstructure:"auth_required"`
}

// AuthConfig defines the structure for authentication configuration.
// AuthConfig defines the structure for authentication configuration.
type AuthConfig struct {
	SessionKey     string   `mapstructure:"session_key"`
	CSRFKey        string   `mapstructure:"csrf_key"`
	TrustedOrigins []string `mapstructure:"trusted_origins"`
}

// StaticConfig defines the structure for static file serving configuration.
type StaticConfig struct {
	URLPrefix string `mapstructure:"url_prefix"`
	Dir       string `mapstructure:"dir"`
}

// TranslatorConfig defines the structure for the translation service.
type TranslatorConfig struct {
	Type   string `mapstructure:"type"`
	APIKey string `mapstructure:"api_key"`
}

// I18nConfig defines the structure for i18n configuration.
type I18nConfig struct {
	DefaultLanguage string `mapstructure:"default_language"`
}

type Config struct {
	Server struct {
		Address string `mapstructure:"address"`
	} `mapstructure:"server"`
	Database struct {
		Type string `mapstructure:"type"`
		DSN  string `mapstructure:"dsn"`
	} `mapstructure:"database"`
	Auth       AuthConfig       `mapstructure:"auth"`
	Static     StaticConfig     `mapstructure:"static"`
	Translator TranslatorConfig `mapstructure:"translator"`
	I18n       I18nConfig       `mapstructure:"i18n"`
	Routes     []Route          `mapstructure:"routes"`
}

// LoadConfig reads configuration from file or environment variables.
func LoadConfig() (*Config, error) {
	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	viper.SetConfigType("yml")

	// Enable Viper to read environment variables
	viper.AutomaticEnv()
	// Replace . with _ in environment variable names (e.g., auth.session_key -> AUTH_SESSION_KEY)
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Explicitly bind environment variables to config keys
	viper.BindEnv("auth.session_key", "SESSION_KEY")
	viper.BindEnv("auth.csrf_key", "CSRF_KEY")
	viper.BindEnv("auth.username", "ADMIN_USERNAME") // Bind username/password too for consistency
	viper.BindEnv("auth.password", "ADMIN_PASSWORD")
	viper.BindEnv("translator.api_key", "TRANSLATOR_API_KEY")
	viper.BindEnv("translator.type", "TRANSLATOR_TYPE")

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err	}

	return &config, nil
}