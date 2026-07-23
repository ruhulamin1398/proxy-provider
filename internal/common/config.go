package common

import (
	"log"

	"github.com/spf13/viper"
)

// Config holds application configuration.
type Config struct {
	Port     string `mapstructure:"PORT"`
	LogLevel string `mapstructure:"LOG_LEVEL"`
}

// LoadConfig reads configuration from .env and environment variables.
func LoadConfig() Config {
	v := viper.New()
	v.SetConfigFile(".env")
	v.SetConfigType("env")
	v.AutomaticEnv()

	// Explicitly bind env vars
	for _, key := range []string{
		"PORT", "LOG_LEVEL",
	} {
		v.BindEnv(key)
	}

	v.SetDefault("PORT", "8080")
	v.SetDefault("LOG_LEVEL", "info")

	if err := v.ReadInConfig(); err != nil {
		log.Println("no .env file found, using environment variables")
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		log.Fatal("failed to parse config", err)
	}
	return cfg
}