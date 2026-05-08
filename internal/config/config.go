package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Addr            string
	QWeatherAPIKey  string
	QWeatherToken   string
	QWeatherBaseURL string
	SQLitePath      string
	CacheDuration   time.Duration
}

func Load() (*Config, error) {
	fileEnv, _ := parseEnvFile(".env")

	addr := value(fileEnv, "ADDR", ":8080")
	apiKey := value(fileEnv, "QWEATHER_API_KEY", "")
	token := value(fileEnv, "QWEATHER_TOKEN", "")
	baseURL := normalizeBaseURL(value(fileEnv, "QWEATHER_BASE_URL", "https://api.qweather.com"))
	sqlitePath := value(fileEnv, "SQLITE_PATH", filepath.Join("data", "weather.db"))
	cacheMinutes, err := strconv.Atoi(value(fileEnv, "CACHE_MINUTES", "10"))
	if err != nil || cacheMinutes <= 0 {
		cacheMinutes = 10
	}

	if apiKey == "" && token == "" {
		return nil, errors.New("missing QWeather credentials: set QWEATHER_API_KEY or QWEATHER_TOKEN")
	}

	return &Config{
		Addr:            addr,
		QWeatherAPIKey:  apiKey,
		QWeatherToken:   token,
		QWeatherBaseURL: baseURL,
		SQLitePath:      sqlitePath,
		CacheDuration:   time.Duration(cacheMinutes) * time.Minute,
	}, nil
}

func parseEnvFile(path string) (map[string]string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]string{}, nil
		}
		return nil, fmt.Errorf("read env file: %w", err)
	}

	values := map[string]string{}
	for _, line := range strings.Split(string(content), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		values[strings.TrimSpace(key)] = strings.Trim(strings.TrimSpace(val), `"`)
	}
	return values, nil
}

func value(fileEnv map[string]string, key, fallback string) string {
	if envVal := strings.TrimSpace(os.Getenv(key)); envVal != "" {
		return envVal
	}
	if fileVal := strings.TrimSpace(fileEnv[key]); fileVal != "" {
		return fileVal
	}
	return fallback
}

func normalizeBaseURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "https://api.qweather.com"
	}
	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
		return strings.TrimRight(raw, "/")
	}
	return "https://" + strings.TrimRight(raw, "/")
}
