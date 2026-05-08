package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"smog-detector/internal/ipgeo"
	"smog-detector/internal/model"
	"smog-detector/internal/qweather"
	"smog-detector/internal/repository"
)

var ErrLocationRequired = errors.New("location required")

type repositoryAPI interface {
	SaveClient(ctx context.Context, clientID string) error
	SaveLocation(ctx context.Context, record model.ClientLocation) error
	GetLocation(ctx context.Context, clientID string) (model.ClientLocation, error)
	SaveDashboard(ctx context.Context, clientID string, dashboard model.Dashboard, fetchedAt, expiresAt time.Time) error
	GetDashboard(ctx context.Context, clientID string) (model.CachedDashboard, error)
}

type WeatherService struct {
	repo          repositoryAPI
	qweather      *qweather.Client
	ipgeo         *ipgeo.Client
	cacheDuration time.Duration
}

func NewWeatherService(repo repositoryAPI, client *qweather.Client, ipgeoClient *ipgeo.Client, cacheDuration time.Duration) *WeatherService {
	return &WeatherService{repo: repo, qweather: client, ipgeo: ipgeoClient, cacheDuration: cacheDuration}
}

func (s *WeatherService) UpdateLocation(ctx context.Context, clientID string, lat, lon float64) (model.Dashboard, error) {
	return s.updateLocationByCoords(ctx, clientID, lat, lon, "gps")
}

func (s *WeatherService) UpdateLocationByIP(ctx context.Context, clientID, ip string) (model.Dashboard, error) {
	if s.ipgeo == nil {
		return model.Dashboard{}, errors.New("ip geolocation client not configured")
	}
	location, err := s.ipgeo.Lookup(ctx, ip)
	if err != nil {
		return model.Dashboard{}, fmt.Errorf("lookup client ip location: %w", err)
	}
	return s.updateLocationByCoords(ctx, clientID, location.Latitude, location.Longitude, "ip")
}

func (s *WeatherService) updateLocationByCoords(ctx context.Context, clientID string, lat, lon float64, source string) (model.Dashboard, error) {
	clientID = strings.TrimSpace(clientID)
	if clientID == "" {
		return model.Dashboard{}, errors.New("missing client_id")
	}
	if err := s.repo.SaveClient(ctx, clientID); err != nil {
		return model.Dashboard{}, err
	}

	city, err := s.qweather.LookupCityByCoordinates(ctx, lat, lon)
	if err != nil {
		return model.Dashboard{}, fmt.Errorf("lookup city by coordinates: %w", err)
	}
	city.Lat = lat
	city.Lon = lon

	record := model.ClientLocation{
		ClientID:  clientID,
		City:      city,
		Source:    source,
		UpdatedAt: time.Now(),
	}
	if err := s.repo.SaveLocation(ctx, record); err != nil {
		return model.Dashboard{}, err
	}

	return s.refreshDashboard(ctx, clientID, city)
}

func (s *WeatherService) SelectCity(ctx context.Context, clientID, cityName string) (model.Dashboard, error) {
	clientID = strings.TrimSpace(clientID)
	cityName = strings.TrimSpace(cityName)
	if clientID == "" {
		return model.Dashboard{}, errors.New("missing client_id")
	}
	if cityName == "" {
		return model.Dashboard{}, errors.New("missing city_name")
	}
	if err := s.repo.SaveClient(ctx, clientID); err != nil {
		return model.Dashboard{}, err
	}

	city, err := s.qweather.LookupCityByName(ctx, cityName)
	if err != nil {
		return model.Dashboard{}, fmt.Errorf("lookup city by name: %w", err)
	}
	if err := s.repo.SaveLocation(ctx, model.ClientLocation{
		ClientID:  clientID,
		City:      city,
		Source:    "manual",
		UpdatedAt: time.Now(),
	}); err != nil {
		return model.Dashboard{}, err
	}

	return s.refreshDashboard(ctx, clientID, city)
}

func (s *WeatherService) GetDashboard(ctx context.Context, clientID string) (model.Dashboard, error) {
	clientID = strings.TrimSpace(clientID)
	if clientID == "" {
		return model.Dashboard{}, errors.New("missing client_id")
	}

	var stale *model.Dashboard
	if cached, err := s.repo.GetDashboard(ctx, clientID); err == nil {
		if time.Now().Before(cached.ExpiresAt) {
			cached.Dashboard.FromCache = true
			return cached.Dashboard, nil
		}
		stale = &cached.Dashboard
	} else if !errors.Is(err, repository.ErrNotFound) {
		return model.Dashboard{}, err
	}

	location, err := s.repo.GetLocation(ctx, clientID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return model.Dashboard{}, ErrLocationRequired
		}
		return model.Dashboard{}, err
	}

	dashboard, err := s.refreshDashboard(ctx, clientID, location.City)
	if err != nil && stale != nil {
		stale.FromCache = true
		return *stale, nil
	}
	return dashboard, err
}

func (s *WeatherService) refreshDashboard(ctx context.Context, clientID string, city model.City) (model.Dashboard, error) {
	weather, err := s.qweather.GetWeatherNow(ctx, city.LocationID)
	if err != nil {
		return model.Dashboard{}, fmt.Errorf("get weather now: %w", err)
	}
	hourly, updateTime, err := s.qweather.GetHourlyWeather(ctx, city.LocationID)
	if err != nil {
		return model.Dashboard{}, fmt.Errorf("get hourly weather: %w", err)
	}
	air, err := s.qweather.GetAirCurrent(ctx, city.Lat, city.Lon)
	if err != nil {
		return model.Dashboard{}, fmt.Errorf("get air quality: %w", err)
	}

	dashboard := model.Dashboard{
		ClientID:  clientID,
		City:      city,
		Weather:   weather,
		Air:       air,
		Hourly:    hourly,
		UpdatedAt: firstNonEmpty(updateTime, weather.ObservedAt, time.Now().Format(time.RFC3339)),
		FromCache: false,
	}

	now := time.Now()
	if err := s.repo.SaveDashboard(ctx, clientID, dashboard, now, now.Add(s.cacheDuration)); err != nil {
		return model.Dashboard{}, err
	}
	return dashboard, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
