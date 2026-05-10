package qweather

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"smog-detector/internal/model"
)

type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  strings.TrimSpace(apiKey),
		httpClient: &http.Client{
			Timeout: 12 * time.Second,
		},
	}
}

func (c *Client) LookupCityByCoordinates(ctx context.Context, lat, lon float64) (model.City, error) {
	query := url.Values{}
	query.Set("location", fmt.Sprintf("%.2f,%.2f", lon, lat))
	query.Set("lang", "zh")
	query.Set("number", "1")

	var resp geoLookupResponse
	if err := c.getJSON(ctx, "/geo/v2/city/lookup", query, &resp); err != nil {
		return model.City{}, err
	}
	return geoLocationToCity(resp, lat, lon)
}

func (c *Client) LookupCityByName(ctx context.Context, name string) (model.City, error) {
	query := url.Values{}
	query.Set("location", name)
	query.Set("lang", "zh")
	query.Set("number", "1")
	query.Set("range", "cn")

	var resp geoLookupResponse
	if err := c.getJSON(ctx, "/geo/v2/city/lookup", query, &resp); err != nil {
		return model.City{}, err
	}
	return geoLocationToCity(resp, 0, 0)
}

func (c *Client) GetWeatherNow(ctx context.Context, locationID string) (model.WeatherNow, error) {
	query := url.Values{}
	query.Set("location", locationID)
	query.Set("lang", "zh")

	var resp weatherNowResponse
	if err := c.getJSON(ctx, "/v7/weather/now", query, &resp); err != nil {
		return model.WeatherNow{}, err
	}

	return model.WeatherNow{
		ObservedAt: resp.Now.ObsTime,
		Text:       resp.Now.Text,
		Temp:       resp.Now.Temp,
		FeelsLike:  resp.Now.FeelsLike,
		Humidity:   resp.Now.Humidity,
		WindDir:    resp.Now.WindDir,
		WindScale:  resp.Now.WindScale,
		WindSpeed:  resp.Now.WindSpeed,
	}, nil
}

func (c *Client) GetHourlyWeather(ctx context.Context, locationID string) ([]model.HourlyPoint, string, error) {
	query := url.Values{}
	query.Set("location", locationID)
	query.Set("lang", "zh")

	var resp hourlyWeatherResponse
	if err := c.getJSON(ctx, "/v7/weather/24h", query, &resp); err != nil {
		return nil, "", err
	}

	points := make([]model.HourlyPoint, 0, len(resp.Hourly))
	for _, item := range resp.Hourly {
		points = append(points, model.HourlyPoint{
			Time:     trimHour(item.FxTime),
			Temp:     item.Temp,
			Humidity: item.Humidity,
			Text:     item.Text,
		})
	}
	return points, resp.UpdateTime, nil
}

func (c *Client) GetAirCurrent(ctx context.Context, lat, lon float64) (model.AirNow, error) {
	query := url.Values{}
	query.Set("lang", "zh")

	endpoint := fmt.Sprintf("/airquality/v1/current/%.2f/%.2f", lat, lon)
	var resp airCurrentResponse
	if err := c.getJSON(ctx, endpoint, query, &resp); err != nil {
		return model.AirNow{}, err
	}

	selected := pickBestIndex(resp.Indexes)
	station := ""
	if len(resp.Stations) > 0 {
		station = resp.Stations[0].Name
	}

	return model.AirNow{
		AQI:               selected.AQIDisplay,
		Category:          selected.Category,
		PrimaryPollutant:  selected.PrimaryPollutant.Name,
		HealthAdvice:      selected.Health.Advice.GeneralPopulation,
		SensitiveAdvice:   selected.Health.Advice.SensitivePopulation,
		MonitoringStation: station,
		IndexCode:         selected.Code,
	}, nil
}

func (c *Client) getJSON(ctx context.Context, endpoint string, query url.Values, out any) error {
	base, err := url.Parse(c.baseURL)
	if err != nil {
		return fmt.Errorf("parse base url: %w", err)
	}
	base.Path = path.Join(base.Path, endpoint)
	base.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base.String(), nil)
	if err != nil {
		return fmt.Errorf("new request: %w", err)
	}
	if c.apiKey != "" {
		req.Header.Set("X-QW-Api-Key", c.apiKey)
	} else {
		return errors.New("missing qweather credentials")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request qweather: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("qweather status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	if code, ok := extractCode(out); ok && code != "" && code != "200" {
		return fmt.Errorf("qweather api code %s", code)
	}
	return nil
}

func extractCode(v any) (string, bool) {
	type hasCode interface{ GetCode() string }
	if codeGetter, ok := v.(hasCode); ok {
		return codeGetter.GetCode(), true
	}
	return "", false
}

func geoLocationToCity(resp geoLookupResponse, fallbackLat, fallbackLon float64) (model.City, error) {
	if len(resp.Location) == 0 {
		return model.City{}, errors.New("no city found")
	}
	item := resp.Location[0]
	lat := fallbackLat
	lon := fallbackLon
	if parsed, err := strconv.ParseFloat(item.Lat, 64); err == nil {
		lat = parsed
	}
	if parsed, err := strconv.ParseFloat(item.Lon, 64); err == nil {
		lon = parsed
	}
	return model.City{
		Name:       item.Name,
		LocationID: item.ID,
		Lat:        lat,
		Lon:        lon,
		Adm1:       item.Adm1,
		Adm2:       item.Adm2,
	}, nil
}

func trimHour(value string) string {
	if len(value) >= 16 {
		return value[11:16]
	}
	return value
}

func pickBestIndex(indexes []airIndex) airIndex {
	if len(indexes) == 0 {
		return airIndex{}
	}
	for _, idx := range indexes {
		if idx.Code == "qaqi" || idx.Code == "cn-aqi" || idx.Code == "us-epa" {
			return idx
		}
	}
	return indexes[0]
}

type geoLookupResponse struct {
	Code     string        `json:"code"`
	Location []geoLocation `json:"location"`
}

func (r geoLookupResponse) GetCode() string { return r.Code }

type geoLocation struct {
	Name string `json:"name"`
	ID   string `json:"id"`
	Lat  string `json:"lat"`
	Lon  string `json:"lon"`
	Adm1 string `json:"adm1"`
	Adm2 string `json:"adm2"`
}

type weatherNowResponse struct {
	Code string `json:"code"`
	Now  struct {
		ObsTime   string `json:"obsTime"`
		Temp      string `json:"temp"`
		FeelsLike string `json:"feelsLike"`
		Text      string `json:"text"`
		Humidity  string `json:"humidity"`
		WindDir   string `json:"windDir"`
		WindScale string `json:"windScale"`
		WindSpeed string `json:"windSpeed"`
	} `json:"now"`
}

func (r weatherNowResponse) GetCode() string { return r.Code }

type hourlyWeatherResponse struct {
	Code       string `json:"code"`
	UpdateTime string `json:"updateTime"`
	Hourly     []struct {
		FxTime   string `json:"fxTime"`
		Temp     string `json:"temp"`
		Humidity string `json:"humidity"`
		Text     string `json:"text"`
	} `json:"hourly"`
}

func (r hourlyWeatherResponse) GetCode() string { return r.Code }

type airCurrentResponse struct {
	Indexes  []airIndex `json:"indexes"`
	Stations []struct {
		Name string `json:"name"`
	} `json:"stations"`
}

type airIndex struct {
	Code             string `json:"code"`
	AQIDisplay       string `json:"aqiDisplay"`
	Category         string `json:"category"`
	PrimaryPollutant struct {
		Name string `json:"name"`
	} `json:"primaryPollutant"`
	Health struct {
		Advice struct {
			GeneralPopulation   string `json:"generalPopulation"`
			SensitivePopulation string `json:"sensitivePopulation"`
		} `json:"advice"`
	} `json:"health"`
}
