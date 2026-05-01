package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// APIClientIface defines the interface for API clients (for testability)
type APIClientIface interface {
	do(method, path string, body interface{}) ([]byte, error)
	streamChatCompletion(req *ChatRequest) error
	streamFIMCompletion(req *FIMRequest) error
}

type Client struct {
	Base   string
	APIKey string
	Client *http.Client
}

func NewClient(base, apiKey string) *Client {
	return &Client{
		Base:   strings.TrimSuffix(base, "/"),
		APIKey: apiKey,
		Client: &http.Client{Timeout: 120 * time.Second},
	}
}

func (c *Client) do(method, path string, body interface{}) ([]byte, error) {
	var buf io.Reader
	if body != nil {
		js, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		buf = bytes.NewReader(js)
	}
	url := c.Base + path
	req, err := http.NewRequest(method, url, buf)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	return b, nil
}

func loadConfig() (base, apiKey string) {
	// Try to load from config file first
	config, err := LoadConfig()
	if err == nil && config != nil {
		if config.APIKey != "" {
			apiKey = config.APIKey
		}
		if config.BaseURL != "" {
			base = config.BaseURL
		}
	}
	
	// Environment variables override config file
	if apiKey == "" {
		apiKey = getEnv("DEEPSEEK_API_KEY", "")
	}
	if base == "" {
		base = getEnv("DEEPSEEK_API_BASE", "")
	}
	
	// Default base URL
	if base == "" {
		base = "https://api.deepseek.com"
	}
	return base, apiKey
}

func loadBetaConfig() (base, apiKey string) {
	// Try to load from config file first
	config, err := LoadConfig()
	if err == nil && config != nil {
		if config.APIKey != "" {
			apiKey = config.APIKey
		}
		if config.BaseURL != "" {
			base = config.BaseURL
		}
	}
	
	// Environment variables override config file
	if apiKey == "" {
		apiKey = getEnv("DEEPSEEK_API_KEY", "")
	}
	if base == "" {
		base = getEnv("DEEPSEEK_API_BASE", "")
	}
	
	// Default to beta endpoint
	if base == "" {
		base = "https://api.deepseek.com/beta"
	}
	return base, apiKey
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
