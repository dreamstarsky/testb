package ipgeo

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	httpClient *http.Client
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

func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 8 * time.Second},
		baseURL:    "https://ipwho.is",
	}
}

func (c *Client) Lookup(ctx context.Context, ip string) (Location, error) {
	ip = strings.TrimSpace(ip)
	if ip == "" {
		return Location{}, fmt.Errorf("empty client ip")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/"+ip, nil)
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
		Success   bool    `json:"success"`
		Message   string  `json:"message"`
		IP        string  `json:"ip"`
		City      string  `json:"city"`
		Region    string  `json:"region"`
		Country   string  `json:"country"`
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return Location{}, fmt.Errorf("decode ipgeo response: %w", err)
	}
	if !payload.Success {
		return Location{}, fmt.Errorf("ipgeo failed: %s", payload.Message)
	}

	return Location{
		IP:        payload.IP,
		City:      payload.City,
		Region:    payload.Region,
		Country:   payload.Country,
		Latitude:  payload.Latitude,
		Longitude: payload.Longitude,
	}, nil
}
