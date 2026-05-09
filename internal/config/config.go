package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	ServerPort string
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string

	JWTSecret   string
	JWTTTLHours int
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	jwtTTLHours, err := getEnvAsInt("JWT_TTL_HOURS")
	if err != nil {
		return nil, err
	}

	cfg := &Config{
		ServerPort: getEnv("SERVER_PORT"),
		DBHost:     getEnv("DB_HOST"),
		DBPort:     getEnv("DB_PORT"),
		DBUser:     getEnv("DB_USER"),
		DBPassword: getEnv("DB_PASSWORD"),
		DBName:     getEnv("DB_NAME"),
		DBSSLMode:  getEnv("DB_SSLMODE"),

		JWTSecret:   getEnv("JWT_SECRET"),
		JWTTTLHours: jwtTTLHours,
	}

	return cfg, nil
}

func getEnv(key string) string {
	value := os.Getenv(key)
	return value
}

func getEnvAsInt(key string) (int, error) {
	value := os.Getenv(key)

	intValue, err := strconv.Atoi(value)

	if err != nil {
		return 0, fmt.Errorf("invalid value for %s: %w", key, err)
	}

	return intValue, nil
}
