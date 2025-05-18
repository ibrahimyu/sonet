package config

import (
	"fmt"

	"github.com/spf13/viper"
)

// LoadConfig loads the configuration from .env file
func LoadConfig() error {
	viper.SetConfigName(".env")
	viper.SetConfigType("env")
	viper.AddConfigPath(".")

	// Set defaults
	viper.SetDefault("PORT", "8080")
	viper.SetDefault("ENV", "development")
	viper.SetDefault("DB_ADAPTER", "sqlite")
	viper.SetDefault("DB_CONNECTION_STRING", "sonet.db")
	viper.SetDefault("RATE_LIMIT_ENABLED", false)
	viper.SetDefault("RATE_LIMIT_REQUESTS", 100)
	viper.SetDefault("RATE_LIMIT_DURATION", 60)
	viper.SetDefault("HOOKS_ENABLED", true)

	// Read the .env file
	err := viper.ReadInConfig()
	if err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			fmt.Println("No .env file found, using defaults and environment variables")
		} else {
			return fmt.Errorf("error reading config file: %s", err)
		}
	}

	// Also read from environment variables
	viper.AutomaticEnv()

	return nil
}
