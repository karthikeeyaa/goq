package config

import (
	"os"
	"path/filepath"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	AppName string
	Port    string
	LogFile string
	Version string
	Mode    string

	// HTTP
	HTTPTimeoutSeconds int
	IntegrationKey     string

	// Control plane
	DBDSN string

	// Message plane
	DataDir     string
	FixtureFile string

	// Log engine
	LogSegmentBytes       int
	LogIndexIntervalBytes int
	MaxMessageBytes       int
}

var AppConfig *Config

func LoadConfig() *Config {
	_ = godotenv.Load()

	mode := getEnv("MODE", "production")
	logsDir := getEnv("LOGS_DIRECTORY", "logs")

	AppConfig = &Config{
		AppName: getEnv("APP_NAME", "goq"),
		Port:    getEnv("PORT", "8080"),
		LogFile: filepath.Join(logsDir, mode+".log"),
		Version: getEnv("VERSION", "v1"),
		Mode:    mode,

		HTTPTimeoutSeconds: getEnv("HTTP_TIMEOUT", 30),
		IntegrationKey:     getEnv("INTEGRATION_KEY", ""),

		DBDSN:       getEnv("DB_DSN", "data/goq.db"),
		DataDir:     getEnv("DATA_DIR", "data/logs"),
		FixtureFile: getEnv("FIXTURES", "build/fixture.json"),

		LogSegmentBytes:       getEnv("LOG_SEGMENT_BYTES", 1<<30),
		LogIndexIntervalBytes: getEnv("LOG_INDEX_INTERVAL_BYTES", 4096),
		MaxMessageBytes:       getEnv("MAX_MESSAGE_BYTES", 1<<20),
	}

	return AppConfig
}

func getEnv[T string | int](key string, defaultVal T) T {
	if val, ok := os.LookupEnv(key); ok {
		switch any(defaultVal).(type) {
		case string:
			return any(val).(T)
		case int:
			if i, err := strconv.Atoi(val); err == nil {
				return any(i).(T)
			}
		}
	}
	return defaultVal
}
