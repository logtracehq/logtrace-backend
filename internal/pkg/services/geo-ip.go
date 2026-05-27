package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const baseURL = "https://api.country.is"

var httpClient = &http.Client{
	Timeout: 10 * time.Second,
}

type Location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type Response struct {
	IP      string `json:"ip"`
	Country string `json:"country"`
	City    string `json:"city,omitempty"`
}

func GeoIPLookup(ctx context.Context, ip string) (*Response, error) {
	if ip == "" {
		return nil, fmt.Errorf("ip must not be empty")
	}
	return get(ctx, ip)
}

func get(ctx context.Context, ip string) (*Response, error) {
	url := baseURL + "/"

	if ip != "" {
		url += ip
	}

	url += "?fields=city"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var result Response
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
