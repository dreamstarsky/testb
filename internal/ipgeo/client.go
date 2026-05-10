package ipgeo

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	httpClient *http.Client
	apiKey     string
	baseURL    string
}

type Location struct {
	IP        string
	City      string
	Region    string
	Country   string
	Latitude  float64
	Longitude float64
}

func NewClient(apiKey, apiBaseURL string) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 8 * time.Second},
		apiKey:     apiKey,
		baseURL:    apiBaseURL,
	}
}

func (c *Client) Lookup(ctx context.Context, ip string) (Location, error) {
	ip = strings.TrimSpace(ip)
	if ip == "" {
		return Location{}, fmt.Errorf("empty client ip")
	}

	params := url.Values{}
	params.Set("ip", ip)
	params.Set("key", c.apiKey)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/ws/location/v1/ip"+"?"+params.Encode(), nil)
	if err != nil {
		return Location{}, fmt.Errorf("new ipgeo request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return Location{}, fmt.Errorf("request ipgeo: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Location{}, fmt.Errorf("read ipgeo response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Location{}, fmt.Errorf("ipgeo status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var payload struct {
		Status  int    `json:"status"`
		Message string `json:"message"`
		Result  struct {
			IP       string `json:"ip"`
			Location struct {
				Latitude  float64 `json:"lat"`
				Longitude float64 `json:"lng"`
			} `json:"location"`
			AdInfo struct {
				Nation     string `json:"nation"`
				NationCode int    `json:"nation_code"`
				Province   string `json:"province"`
				City       string `json:"city"`
				District   string `json:"district"`
				Adcode     int    `json:"adcode"`
			} `json:"ad_info"`
		} `json:"result"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return Location{}, fmt.Errorf("decode ipgeo response: %w", err)
	}
	if payload.Status != 0 {
		return Location{}, fmt.Errorf("ipgeo failed: %s", payload.Message)
	}

	return Location{
		IP:        payload.Result.IP,
		City:      payload.Result.AdInfo.City,
		Region:    payload.Result.AdInfo.District,
		Country:   payload.Result.AdInfo.Nation,
		Latitude:  payload.Result.Location.Latitude,
		Longitude: payload.Result.Location.Longitude,
	}, nil
}
