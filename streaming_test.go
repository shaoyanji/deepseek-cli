package main

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStreamChatCompletion(t *testing.T) {
	// Create a mock HTTP server that returns SSE stream
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request headers
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "text/event-stream", r.Header.Get("Accept"))
		
		// Set SSE headers
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		
		// Write SSE chunks
		fmt.Fprintln(w, "data: {\"choices\":[{\"delta\":{\"content\":\"Hello\"}}]}")
		fmt.Fprintln(w, "data: {\"choices\":[{\"delta\":{\"content\":\" world\"}}]}")
		fmt.Fprintln(w, "data: {\"choices\":[{\"finish_reason\":\"stop\"}]}")
		fmt.Fprintln(w, "data: [DONE]")
	}))
	defer server.Close()

	// Create client with mock server URL
	client := NewClient(server.URL, "test-key")
	
	// Create a chat request
	req := &ChatRequest{
		Model:    "deepseek-chat",
		Messages: []Message{{Role: "user", Content: "Say hello"}},
		Stream:   true,
	}

	// Capture stdout
	oldStdout := captureStdout(func() {
		err := client.streamChatCompletion(req)
		assert.NoError(t, err)
	})

	// Verify output contains expected content
	assert.Contains(t, oldStdout, "Hello")
	assert.Contains(t, oldStdout, "world")
	assert.Contains(t, oldStdout, "finish_reason")
}

func TestStreamChatCompletionWithError(t *testing.T) {
	// Create a mock HTTP server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": "invalid api key"}`))
	}))
	defer server.Close()

	// Create client with mock server URL
	client := NewClient(server.URL, "test-key")
	
	// Create a chat request
	req := &ChatRequest{
		Model:    "deepseek-chat",
		Messages: []Message{{Role: "user", Content: "Say hello"}},
		Stream:   true,
	}

	err := client.streamChatCompletion(req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP 401")
}

func TestStreamFIMCompletion(t *testing.T) {
	// Create a mock HTTP server that returns SSE stream
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request headers
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "text/event-stream", r.Header.Get("Accept"))
		
		// Set SSE headers
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		
		// Write SSE chunks for FIM
		fmt.Fprintln(w, "data: {\"choices\":[{\"text\":\"def \"}]}")
		fmt.Fprintln(w, "data: {\"choices\":[{\"text\":\"hello()\"}]}")
		fmt.Fprintln(w, "data: {\"choices\":[{\"finish_reason\":\"stop\"}]}")
		fmt.Fprintln(w, "data: [DONE]")
	}))
	defer server.Close()

	// Create client with mock server URL
	client := NewClient(server.URL, "test-key")
	
	// Create a FIM request
	req := &FIMRequest{
		Model:  "deepseek-coder",
		Prompt: "def ",
		Stream: true,
	}

	// Capture stdout
	oldStdout := captureStdout(func() {
		err := client.streamFIMCompletion(req)
		assert.NoError(t, err)
	})

	// Verify output contains expected content
	assert.Contains(t, oldStdout, "def ")
	assert.Contains(t, oldStdout, "hello()")
	assert.Contains(t, oldStdout, "finish_reason")
}

func TestStreamFIMCompletionWithError(t *testing.T) {
	// Create a mock HTTP server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": "invalid api key"}`))
	}))
	defer server.Close()

	// Create client with mock server URL
	client := NewClient(server.URL, "test-key")
	
	// Create a FIM request
	req := &FIMRequest{
		Model:  "deepseek-coder",
		Prompt: "def ",
		Stream: true,
	}

	err := client.streamFIMCompletion(req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP 401")
}

func TestParseSSEStreamChat(t *testing.T) {
	// Test parsing SSE stream for chat completion
	client := NewClient("https://api.deepseek.com", "test-key")
	
	// Create a mock SSE stream
	sseStream := `data: {"choices":[{"delta":{"content":"Hello"}}]}
data: {"choices":[{"delta":{"content":" world"}}]}
data: {"choices":[{"finish_reason":"stop"}]}
data: [DONE]
`
	
	reader := strings.NewReader(sseStream)
	
	// Capture stdout
	oldStdout := captureStdout(func() {
		err := client.parseSSEStream(reader, "chat")
		assert.NoError(t, err)
	})

	// Verify output
	assert.Contains(t, oldStdout, "Hello")
	assert.Contains(t, oldStdout, "world")
	assert.Contains(t, oldStdout, "finish_reason")
}

func TestParseSSEStreamFIM(t *testing.T) {
	// Test parsing SSE stream for FIM completion
	client := NewClient("https://api.deepseek.com", "test-key")
	
	// Create a mock SSE stream
	sseStream := `data: {"choices":[{"text":"def "}]}
data: {"choices":[{"text":"hello()"}]}
data: {"choices":[{"finish_reason":"stop"}]}
data: [DONE]
`
	
	reader := strings.NewReader(sseStream)
	
	// Capture stdout
	oldStdout := captureStdout(func() {
		err := client.parseSSEStream(reader, "fim")
		assert.NoError(t, err)
	})

	// Verify output
	assert.Contains(t, oldStdout, "def ")
	assert.Contains(t, oldStdout, "hello()")
	assert.Contains(t, oldStdout, "finish_reason")
}

func TestParseSSEStreamWithUsage(t *testing.T) {
	// Test parsing SSE stream with usage information
	client := NewClient("https://api.deepseek.com", "test-key")
	
	// Create a mock SSE stream with usage
	sseStream := `data: {"choices":[{"delta":{"content":"Hello"}}]}
data: {"choices":[{"finish_reason":"stop"}],"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}}
data: [DONE]
`
	
	reader := strings.NewReader(sseStream)
	
	// Capture stdout
	oldStdout := captureStdout(func() {
		err := client.parseSSEStream(reader, "chat")
		assert.NoError(t, err)
	})

	// Verify output contains usage information
	assert.Contains(t, oldStdout, "Hello")
	assert.Contains(t, oldStdout, "usage")
	assert.Contains(t, oldStdout, "prompt_tokens=10")
	assert.Contains(t, oldStdout, "completion_tokens=5")
	assert.Contains(t, oldStdout, "total_tokens=15")
}

func TestParseSSEStreamEmptyLines(t *testing.T) {
	// Test parsing SSE stream with empty lines
	client := NewClient("https://api.deepseek.com", "test-key")
	
	// Create a mock SSE stream with empty lines
	sseStream := `

data: {"choices":[{"delta":{"content":"Hello"}}]}

data: [DONE]

`
	
	reader := strings.NewReader(sseStream)
	
	// Capture stdout
	oldStdout := captureStdout(func() {
		err := client.parseSSEStream(reader, "chat")
		assert.NoError(t, err)
	})

	// Verify output
	assert.Contains(t, oldStdout, "Hello")
}

func TestParseSSEStreamInvalidData(t *testing.T) {
	// Test parsing SSE stream with invalid data (should skip)
	client := NewClient("https://api.deepseek.com", "test-key")
	
	// Create a mock SSE stream with invalid data
	sseStream := `data: invalid json
data: {"choices":[{"delta":{"content":"Hello"}}]}
data: [DONE]
`
	
	reader := strings.NewReader(sseStream)
	
	// Capture stdout
	oldStdout := captureStdout(func() {
		err := client.parseSSEStream(reader, "chat")
		assert.NoError(t, err)
	})

	// Verify output (invalid line should be skipped)
	assert.Contains(t, oldStdout, "Hello")
}

// Helper function to capture stdout
func captureStdout(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	
	f()
	
	w.Close()
	os.Stdout = old
	
	var buf bytes.Buffer
	buf.ReadFrom(r)
	return buf.String()
}