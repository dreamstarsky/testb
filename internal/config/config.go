package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Addr            string
	QWeatherAPIKey  string
	QWeatherBaseURL string
	SQLitePath      string
	CacheDuration   time.Duration
	IpGeoAPIKey     string
	IpGeoBaseURL    string
}

func Load() (*Config, error) {
	if err := godotenv.Load(".env"); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("load .env: %w", err)
	}

	addr := value("ADDR", ":8080")
	apiKey := value("QWEATHER_API_KEY", "")
	baseURL := normalizeBaseURL(value("QWEATHER_BASE_URL", "https://api.qweather.com"))
	sqlitePath := value("SQLITE_PATH", filepath.Join("data", "weather.db"))
	cacheMinutes, err := strconv.Atoi(value("CACHE_MINUTES", "10"))
	ipGeoAPIKey := value("TENCENT_IPGEO_KEY", "")
	ipGeoBaseURL := normalizeBaseURL(value("TENCENT_IPGEO_BASE_URL", "https://apis.map.qq.com"))
	if err != nil || cacheMinutes <= 0 {
		cacheMinutes = 10
	}

	if apiKey == "" {
		return nil, errors.New("missing QWeather credentials: set QWEATHER_API_KEY")
	}

	return &Config{
		Addr:            addr,
		QWeatherAPIKey:  apiKey,
		QWeatherBaseURL: baseURL,
		SQLitePath:      sqlitePath,
		CacheDuration:   time.Duration(cacheMinutes) * time.Minute,
		IpGeoAPIKey:     ipGeoAPIKey,
		IpGeoBaseURL:    ipGeoBaseURL,
	}, nil
}

func value(key, fallback string) string {
	if envVal := strings.TrimSpace(os.Getenv(key)); envVal != "" {
		return envVal
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
