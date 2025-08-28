package config

import "github.com/spf13/viper"

// Route defines the structure for a single route in the config.
type Route struct {
	Path         string   `mapstructure:"path"`
	Handler      string   `mapstructure:"handler"`
	Methods      []string `mapstructure:"methods"`
	AuthRequired bool     `mapstructure:"auth_required"`
}

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

// Config holds all configuration for the application.
type Config struct {
	Server struct {
		Address string `mapstructure:"address"`
	} `mapstructure:"server"`
	Database struct {
		Type string `mapstructure:"type"`
		DSN  string `mapstructure:"dsn"`
	} `mapstructure:"database"`
	Auth   AuthConfig   `mapstructure:"auth"`
	Static StaticConfig `mapstructure:"static"`
	Routes []Route      `mapstructure:"routes"`
}

// LoadConfig reads configuration from file or environment variables.
func LoadConfig() (*Config, error) {
	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	viper.SetConfigType("yml")

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err	}

	return &config, nil
}
