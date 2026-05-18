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

	HMACSecret string
	PGPSecret  string

	CBRURL          string
	CBRRateMargin   float64
	CBRLookbackDays int

	SMTPEnabled bool
	SMTPHost    string
	SMTPPort    int
	SMTPUser    string
	SMTPPass    string
	SMTPFrom    string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	jwtTTLHours, err := getEnvAsInt("JWT_TTL_HOURS")
	if err != nil {
		return nil, err
	}

	cbrRateMargin, err := getEnvAsFloat("CBR_RATE_MARGIN")
	if err != nil {
		return nil, err
	}

	cbrLookbackDays, err := getEnvAsInt("CBR_LOOKBACK_DAYS")
	if err != nil {
		return nil, err
	}

	smtpEnabled, err := getEnvAsBool("SMTP_ENABLED")
	if err != nil {
		return nil, err
	}

	smtpPort, err := getEnvAsInt("SMTP_PORT")
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

		HMACSecret: getEnv("HMAC_SECRET"),
		PGPSecret:  getEnv("PGP_SECRET"),

		CBRURL:          getEnv("CBR_URL"),
		CBRRateMargin:   cbrRateMargin,
		CBRLookbackDays: cbrLookbackDays,

		SMTPEnabled: smtpEnabled,
		SMTPHost:    getEnv("SMTP_HOST"),
		SMTPPort:    smtpPort,
		SMTPUser:    getEnv("SMTP_USER"),
		SMTPPass:    getEnv("SMTP_PASS"),
		SMTPFrom:    getEnv("SMTP_FROM"),
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

func getEnvAsFloat(key string) (float64, error) {
	value := os.Getenv(key)

	floatValue, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid value for %s: %w", key, err)
	}

	return floatValue, nil
}

func getEnvAsBool(key string) (bool, error) {
	value := os.Getenv(key)

	boolValue, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("invalid value for %s: %w", key, err)
	}

	return boolValue, nil
}
