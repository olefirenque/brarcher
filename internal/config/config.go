package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	PostgresDSN string
	RedisAddr   string
	HostName    string
	HTTPPort    int
}

// New returns a new Config struct.
func New() *Config {
	hostname, _ := os.Hostname()
	fmt.Printf("os.Hostname()=%s\n", hostname)

	return &Config{
		// default values are not insecure and used only for local run
		PostgresDSN: getEnv("POSTGRES_DSN", "postgresql://postgres:password@localhost:5431/db"),
		RedisAddr:   getEnv("REDIS_ADDR", "localhost:6381"),
		HostName:    getEnv("HOSTNAME", hostname),
		HTTPPort:    getEnvAsInt("HTTP_PORT", 7999),
	}
}

// getEnv reads an environment or return a default value.
func getEnv(key string, defaultVal string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}

	return defaultVal
}

// getEnvAsInt reads an environment variable into integer or return a default value.
func getEnvAsInt(name string, defaultVal int) int {
	valueStr := getEnv(name, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}

	return defaultVal
}
