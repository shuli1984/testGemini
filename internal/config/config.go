package config

import "github.com/spf13/viper"

// Route defines the structure for a single route in the config.
type Route struct {
	Path         string   `mapstructure:"path"`
	Handler      string   `mapstructure:"handler"`
	Methods      []string `mapstructure:"methods"`
	AuthRequired bool     `mapstructure:"auth_required"`
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
	Auth struct {
		SessionKey     string   `mapstructure:"session_key"`
		CSRFKey        string   `mapstructure:"csrf_key"`
		TrustedOrigins []string `mapstructure:"trusted_origins"`
	} `mapstructure:"auth"`
	Static struct {
		URLPrefix string `mapstructure:"url_prefix"`
		Dir       string `mapstructure:"dir"`
	} `mapstructure:"static"`
	Routes []Route `mapstructure:"routes"`
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
		return nil, err
	}

	return &config, nil
}
