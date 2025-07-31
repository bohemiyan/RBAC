package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config holds application configuration loaded from environment variables.
type Config struct {
	PostgresHost     string
	PostgresPort     int
	PostgresUser     string
	PostgresPassword string
	PostgresDB       string
	RedisHost        string
	RedisPort        int
	RedisPassword    string
	AppPort          int
	CacheTTL		int
	CachePrefix		string 
	AutoMigrate		bool 
	AuditLog		bool 
}

// LoadConfig loads environment variables into a Config struct.
func LoadConfig() (*Config, error) {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		return nil, fmt.Errorf("failed to load .env file: %w", err)
	}

	// Parse Postgres port
	postgresPort, err := strconv.Atoi(os.Getenv("POSTGRES_PORT"))
	if err != nil {
		return nil, fmt.Errorf("invalid POSTGRES_PORT: %w", err)
	}

	// Parse Redis port
	redisPort, err := strconv.Atoi(os.Getenv("REDIS_PORT"))
	if err != nil {
		return nil, fmt.Errorf("invalid REDIS_PORT: %w", err)
	}

	// Parse App port
	appPort, err := strconv.Atoi(os.Getenv("APP_PORT"))
	if err != nil {
		return nil, fmt.Errorf("invalid APP_PORT: %w", err)
	}

	return &Config{
		PostgresHost:     os.Getenv("POSTGRES_HOST"),
		PostgresPort:     postgresPort,
		PostgresUser:     os.Getenv("POSTGRES_USER"),
		PostgresPassword: os.Getenv("POSTGRES_PASSWORD"),
		PostgresDB:       os.Getenv("POSTGRES_DB"),
		RedisHost:        os.Getenv("REDIS_HOST"),
		RedisPort:        redisPort,
		RedisPassword:    os.Getenv("REDIS_PASSWORD"),
		AppPort:          appPort,
	}, nil
}