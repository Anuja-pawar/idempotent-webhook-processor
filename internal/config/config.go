package config

import "os"

type Config struct {
	Port     string
	RedisURL string
}

// Load configurations from environment variables with fallback defaults.
func Load() *Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "localhost:6379"
	}

	return &Config{
		Port:     port,
		RedisURL: redisURL,
	}
}
