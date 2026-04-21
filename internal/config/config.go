package config // Defines application configuration loading logic.

import (
	"fmt" // Formats readable error messages.
	"os"  // Reads environment variables from the OS process.

	"github.com/joho/godotenv" // Loads .env file values into process environment.
)

type Config struct {
	AppEnv   string // Stores environment name like development/staging/production.
	HTTPPort string // Stores the port where HTTP server will listen.
	DBURL    string // Stores full PostgreSQL connection string.
}

func Load() (Config, error) {
	_ = godotenv.Load() // Try loading .env from current directory (ignore if file doesn't exist).

	cfg := Config{ // Build config with defaults first, then validate required values.
		AppEnv:   getEnv("APP_ENV", "development"), // Default app environment.
		HTTPPort: getEnv("HTTP_PORT", "8080"),      // Default HTTP port.
		DBURL:    os.Getenv("DB_URL"),              // No default for DB URL because it is required.
	}

	if cfg.DBURL == "" { // Ensure required DB setting exists.
		return Config{}, fmt.Errorf("DB_URL is required") // Return explicit startup error.
	}

	return cfg, nil // Return validated configuration.
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key) // Read key from environment.
	if value == "" {        // If not provided...
		return fallback // ...return fallback default.
	}
	return value // Otherwise return real configured value.
}
