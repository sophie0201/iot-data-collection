package config

import (
	"errors"
	"os"
	"strings"
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

func Load() (*Config, error) {
	cfg := &Config{
		PostgresHost:     getEnv("POSTGRES_HOST", "database"),
		PostgresPort:     getEnv("POSTGRES_PORT", "5432"),
		PostgresUser:     getEnv("POSTGRES_USER", ""),
		PostgresPassword: getEnv("POSTGRES_PASSWORD", ""),
		PostgresDB:       getEnv("POSTGRES_DB", "iot_db"),

		RedisHost: getEnv("REDIS_HOST", "cache"),
		RedisPort: getEnv("REDIS_PORT", "6379"),

		AppPort: getEnv("APP_PORT", "8080"),
		AppEnv:  getEnv("APP_ENV", "development"),
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) Validate() error {
	var missing []string
	if strings.TrimSpace(c.PostgresUser) == "" {
		missing = append(missing, "POSTGRES_USER")
	}
	if strings.TrimSpace(c.PostgresPassword) == "" {
		missing = append(missing, "POSTGRES_PASSWORD")
	}
	if len(missing) > 0 {
		return errors.New("環境變數必填欄位缺失: " + strings.Join(missing, ", "))
	}
	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
