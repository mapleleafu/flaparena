package config

import (
    "os"
    "log"
)

type Config struct {
    DBHost     string
    DBPort     string
    DBUser     string
    DBPassword string
    DBName     string
    JWTSecret  string
}

func LoadConfig() *Config {
    return &Config{
        DBHost:     getEnv("DB_HOST", "localhost"),
        DBPort:     getEnv("DB_PORT", "5432"),
        DBUser:     getEnv("DB_USER", "user"),
        DBPassword: getEnv("DB_PASSWORD", "password"),
        DBName:     getEnv("DB_NAME", "dbname"),
        JWTSecret:  getEnv("JWT_SECRET", "secret"),
    }
}

// getEnv reads an environment variable and returns its value or a default value
func getEnv(key, defaultValue string) string {
    value, exists := os.LookupEnv(key)
    if !exists {
        value = defaultValue
        log.Printf("Environment variable %s not set, using default value: %s", key, defaultValue)
    }
    return value
}
