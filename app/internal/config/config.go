package config

import (
	"os"
)

type Config struct {
	PostgresHost     string
	PostgresPort     string
	PostgresUser     string
	PostgresPassword string
	PostgresDB       string

	RedisHost string
	RedisPort string

	AppPort string
	AppEnv  string
}

func Load() *Config {
	return &Config{
		PostgresHost: getEnv("POSTGRES_HOST", "database"),
		PostgresPort: getEnv("POSTGRES_PORT", "5432"),
		PostgresUser:     getEnv("POSTGRES_USER", ""),
		PostgresPassword: getEnv("POSTGRES_PASSWORD", ""),
		PostgresDB:       getEnv("POSTGRES_DB", "iot_db"),

		RedisHost: getEnv("REDIS_HOST", "cache"),
		RedisPort: getEnv("REDIS_PORT", "6379"),

		AppPort: getEnv("APP_PORT", "8080"),
		AppEnv:  getEnv("APP_ENV", "development"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
