package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	AppName            string
	Port               string
	DBDriver           string
	DBDSN              string
	WorkerCount        int
	PollIntervalMs     int
	HTTPTimeoutSeconds int
	APIKey             string
	FixtureFile string
	LogFile            string
}

var AppConfig *Config

func LoadConfig() *Config {
	_ = godotenv.Load()

	AppConfig = &Config{
		AppName:            getEnv("APP_NAME", "goq"),
		Port:               getEnv("PORT", "8080"),
		DBDriver:           getEnv("DB_DRIVER", "sqlite3"),
		DBDSN:              getEnv("DB_DSN", "data/goq.db"),
		WorkerCount:        getEnvInt("WORKER_COUNT", 10),
		PollIntervalMs:     getEnvInt("POLL_INTERVAL", 500),
		HTTPTimeoutSeconds: getEnvInt("HTTP_TIMEOUT", 30),
		APIKey:             getEnv("API_KEY", ""),
		FixtureFile: getEnv("FIXTURE_FILE", "build/fixture.json"),
		LogFile:            getEnv("LOG_FILE", "logs/development.log"),
	}

	return AppConfig
}

func getEnv(key, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if val, ok := os.LookupEnv(key); ok {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return defaultVal
}