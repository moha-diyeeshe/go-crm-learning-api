package config // Defines application configuration loading logic.

import (
	"fmt" // Formats readable error messages.
	"os"  // Reads environment variables from the OS process.

	"github.com/joho/godotenv" // Loads .env file values into process environment.
)

type Config struct {
	AppEnv        string // Stores environment name like development/staging/production.
	HTTPPort      string // Stores the port where HTTP server will listen.
	DBURL         string // Stores full PostgreSQL connection string.
	JWTSecret     string // Stores signing secret used to create and verify JWT tokens.
	JWTTTLMinutes int    // Stores JWT expiration time in minutes.
}

func Load() (Config, error) {
	_ = godotenv.Load() // Try loading .env from current directory (ignore if file doesn't exist).

	cfg := Config{ // Build config with defaults first, then validate required values.
		AppEnv:        getEnv("APP_ENV", "development"),   // Default app environment.
		HTTPPort:      getEnv("HTTP_PORT", "8080"),        // Default HTTP port.
		DBURL:         os.Getenv("DB_URL"),                // No default for DB URL because it is required.
		JWTSecret:     os.Getenv("JWT_SECRET"),            // No default for JWT secret because it is required.
		JWTTTLMinutes: getEnvAsInt("JWT_TTL_MINUTES", 60), // Default token TTL to 60 minutes.
	}

	if cfg.DBURL == "" { // Ensure required DB setting exists.
		return Config{}, fmt.Errorf("DB_URL is required") // Return explicit startup error.
	}
	if cfg.JWTSecret == "" { // Ensure required JWT secret exists.
		return Config{}, fmt.Errorf("JWT_SECRET is required") // Return explicit startup error.
	}
	if cfg.JWTTTLMinutes <= 0 { // Ensure TTL is a positive integer.
		return Config{}, fmt.Errorf("JWT_TTL_MINUTES must be greater than zero") // Return explicit validation error.
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

func getEnvAsInt(key string, fallback int) int {
	value := os.Getenv(key) // Read integer-like environment variable from process.
	if value == "" {        // If key is missing...
		return fallback // ...use fallback integer value.
	}

	var parsed int                             // Holds parsed integer value.
	_, err := fmt.Sscanf(value, "%d", &parsed) // Parses decimal integer from string.
	if err != nil {                            // If parsing fails...
		return fallback // ...fall back to default instead of crashing loader.
	}
	return parsed // Return parsed integer when valid.
}
