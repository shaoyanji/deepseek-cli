package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewClient(t *testing.T) {
	client := NewClient("https://api.deepseek.com", "test-key")
	assert.NotNil(t, client)
	assert.Equal(t, "test-key", client.APIKey)
	assert.Equal(t, "https://api.deepseek.com", client.Base)
}

func TestNewClientWithBaseURL(t *testing.T) {
	client := NewClient("https://custom-api.example.com", "test-key")
	assert.NotNil(t, client)
	assert.Equal(t, "https://custom-api.example.com", client.Base)
}

func TestNewClientDefaultBaseURL(t *testing.T) {
	client := NewClient("https://api.deepseek.com", "test-key")
	assert.NotNil(t, client)
	assert.Equal(t, "https://api.deepseek.com", client.Base)
}

func TestClientDo(t *testing.T) {
	// Create a mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request headers
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		
		// Return success response
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"result": "success"}`))
	}))
	defer server.Close()

	// Create client with mock server URL
	client := NewClient(server.URL, "test-key")
	
	// Test POST request with body
	resp, err := client.do("POST", "/test", map[string]string{"key": "value"})
	assert.NoError(t, err)
	assert.Equal(t, `{"result": "success"}`, string(resp))
}

func TestClientDoGET(t *testing.T) {
	// Create a mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))
		
		// Return success response
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"models": ["model1", "model2"]}`))
	}))
	defer server.Close()

	// Create client with mock server URL
	client := NewClient(server.URL, "test-key")
	
	// Test GET request with nil body
	resp, err := client.do("GET", "/models", nil)
	assert.NoError(t, err)
	assert.Equal(t, `{"models": ["model1", "model2"]}`, string(resp))
}

func TestClientDoHTTPError(t *testing.T) {
	// Create a mock HTTP server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error": "invalid api key"}`))
	}))
	defer server.Close()

	// Create client with mock server URL
	client := NewClient(server.URL, "test-key")
	
	// Test error handling
	resp, err := client.do("POST", "/test", map[string]string{"key": "value"})
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "HTTP 401")
}

func TestClientDoNetworkError(t *testing.T) {
	// Create client with invalid URL
	client := NewClient("http://invalid-url-that-does-not-exist-12345.com", "test-key")
	
	// Test network error handling
	resp, err := client.do("POST", "/test", map[string]string{"key": "value"})
	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestClientDoWithNilBody(t *testing.T) {
	// Create a mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))
		
		// Verify Content-Type is not set for nil body
		assert.Equal(t, "", r.Header.Get("Content-Type"))
		
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"result": "success"}`))
	}))
	defer server.Close()

	// Create client with mock server URL
	client := NewClient(server.URL, "test-key")
	
	// Test GET request with nil body
	resp, err := client.do("GET", "/test", nil)
	assert.NoError(t, err)
	assert.Equal(t, `{"result": "success"}`, string(resp))
}

func TestClientDoBaseURLTrailingSlash(t *testing.T) {
	// Create a mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/test", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"result": "success"}`))
	}))
	defer server.Close()

	// Create client with trailing slash in base URL
	client := NewClient(server.URL+"/", "test-key")
	
	// Test that trailing slash is trimmed
	resp, err := client.do("GET", "/test", nil)
	assert.NoError(t, err)
	assert.Equal(t, `{"result": "success"}`, string(resp))
}

func TestLoadConfig(t *testing.T) {
	// Save original env vars
	originalAPIKey := os.Getenv("DEEPSEEK_API_KEY")
	originalBase := os.Getenv("DEEPSEEK_API_BASE")
	defer func() {
		if originalAPIKey != "" {
			_ = os.Setenv("DEEPSEEK_API_KEY", originalAPIKey)
		} else {
			_ = os.Unsetenv("DEEPSEEK_API_KEY")
		}
		if originalBase != "" {
			_ = os.Setenv("DEEPSEEK_API_BASE", originalBase)
		} else {
			_ = os.Unsetenv("DEEPSEEK_API_BASE")
		}
	}()

	// Test with env vars set
	_ = os.Setenv("DEEPSEEK_API_KEY", "env-key")
	_ = os.Setenv("DEEPSEEK_API_BASE", "https://env-api.example.com")

	base, apiKey := loadConfig()
	assert.Equal(t, "env-key", apiKey)
	assert.Equal(t, "https://env-api.example.com", base)

	// Test with only API key env var
	_ = os.Unsetenv("DEEPSEEK_API_BASE")
	base, apiKey = loadConfig()
	assert.Equal(t, "env-key", apiKey)
	assert.Equal(t, "https://api.deepseek.com", base) // default

	// Test with no env vars (should return defaults)
	_ = os.Unsetenv("DEEPSEEK_API_KEY")
	base, apiKey = loadConfig()
	assert.Equal(t, "", apiKey)
	assert.Equal(t, "https://api.deepseek.com", base)
}

func TestLoadBetaConfig(t *testing.T) {
	// Save original env vars
	originalAPIKey := os.Getenv("DEEPSEEK_API_KEY")
	originalBase := os.Getenv("DEEPSEEK_API_BASE")
	defer func() {
		if originalAPIKey != "" {
			_ = os.Setenv("DEEPSEEK_API_KEY", originalAPIKey)
		} else {
			_ = os.Unsetenv("DEEPSEEK_API_KEY")
		}
		if originalBase != "" {
			_ = os.Setenv("DEEPSEEK_API_BASE", originalBase)
		} else {
			_ = os.Unsetenv("DEEPSEEK_API_BASE")
		}
	}()

	// Test with env vars set
	_ = os.Setenv("DEEPSEEK_API_KEY", "env-key")
	_ = os.Setenv("DEEPSEEK_API_BASE", "https://env-api.example.com")

	base, apiKey := loadBetaConfig()
	assert.Equal(t, "env-key", apiKey)
	assert.Equal(t, "https://env-api.example.com", base)

	// Test with only API key env var (should use beta default)
	_ = os.Unsetenv("DEEPSEEK_API_BASE")
	base, apiKey = loadBetaConfig()
	assert.Equal(t, "env-key", apiKey)
	assert.Equal(t, "https://api.deepseek.com/beta", base) // beta default

	// Test with no env vars (should return beta defaults)
	_ = os.Unsetenv("DEEPSEEK_API_KEY")
	base, apiKey = loadBetaConfig()
	assert.Equal(t, "", apiKey)
	assert.Equal(t, "https://api.deepseek.com/beta", base)
}

func TestGetEnv(t *testing.T) {
	// Test getEnv helper
	_ = os.Setenv("TEST_VAR_123", "test_value")
	defer func() { _ = os.Unsetenv("TEST_VAR_123") }()
	
	val := getEnv("TEST_VAR_123", "")
	assert.Equal(t, "test_value", val)
}

func TestGetEnvEmpty(t *testing.T) {
	val := getEnv("NONEXISTENT_VAR_123456", "")
	assert.Equal(t, "", val)
}
