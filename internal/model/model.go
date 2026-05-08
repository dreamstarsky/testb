package model

import "time"

type City struct {
	Name       string  `json:"name"`
	LocationID string  `json:"location_id"`
	Lat        float64 `json:"lat"`
	Lon        float64 `json:"lon"`
	Adm1       string  `json:"adm1,omitempty"`
	Adm2       string  `json:"adm2,omitempty"`
}

type WeatherNow struct {
	ObservedAt string `json:"observed_at"`
	Text       string `json:"text"`
	Temp       string `json:"temp"`
	FeelsLike  string `json:"feels_like"`
	Humidity   string `json:"humidity"`
	WindDir    string `json:"wind_dir"`
	WindScale  string `json:"wind_scale"`
	WindSpeed  string `json:"wind_speed"`
}

type AirNow struct {
	AQI               string `json:"aqi"`
	Category          string `json:"category"`
	PrimaryPollutant  string `json:"primary_pollutant"`
	HealthAdvice      string `json:"health_advice"`
	SensitiveAdvice   string `json:"sensitive_advice"`
	MonitoringStation string `json:"monitoring_station,omitempty"`
	IndexCode         string `json:"index_code,omitempty"`
}

type HourlyPoint struct {
	Time     string `json:"time"`
	Temp     string `json:"temp"`
	Humidity string `json:"humidity"`
	Text     string `json:"text"`
}

type Dashboard struct {
	ClientID  string        `json:"client_id"`
	City      City          `json:"city"`
	Weather   WeatherNow    `json:"weather_now"`
	Air       AirNow        `json:"air_now"`
	Hourly    []HourlyPoint `json:"hourly"`
	UpdatedAt string        `json:"updated_at"`
	FromCache bool          `json:"from_cache"`
}

type ClientLocation struct {
	ClientID  string
	City      City
	Source    string
	UpdatedAt time.Time
}

type CachedDashboard struct {
	Dashboard Dashboard
	ExpiresAt time.Time
	FetchedAt time.Time
}
