package main

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractContent(t *testing.T) {
	tests := []struct {
		name           string
		data           []byte
		completionType string
		expected       string
	}{
		{
			name:           "chat response with content",
			data:           []byte(`{"choices":[{"message":{"content":"Hello world"}}]}`),
			completionType: "chat",
			expected:       "Hello world",
		},
		{
			name:           "chat response without content",
			data:           []byte(`{"choices":[{"message":{}}]}`),
			completionType: "chat",
			expected:       `{"choices":[{"message":{}}]}`,
		},
		{
			name:           "fim response with text",
			data:           []byte(`{"choices":[{"text":"function test() {}"}]}`),
			completionType: "fim",
			expected:       "function test() {}",
		},
		{
			name:           "fim response without text",
			data:           []byte(`{"choices":[{}]}`),
			completionType: "fim",
			expected:       `{"choices":[{}]}`,
		},
		{
			name:           "invalid JSON",
			data:           []byte(`invalid json`),
			completionType: "chat",
			expected:       `invalid json`,
		},
		{
			name:           "empty choices",
			data:           []byte(`{"choices":[]}`),
			completionType: "chat",
			expected:       `{"choices":[]}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractContent(tt.data, tt.completionType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractUsage(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected *Usage
	}{
		{
			name: "valid usage",
			data: []byte(`{"usage":{"prompt_tokens":10,"completion_tokens":20,"total_tokens":30}}`),
			expected: &Usage{
				PromptTokens:     10,
				CompletionTokens: 20,
				TotalTokens:      30,
			},
		},
		{
			name:     "no usage field",
			data:     []byte(`{"choices":[]}`),
			expected: nil,
		},
		{
			name:     "invalid JSON",
			data:     []byte(`invalid`),
			expected: nil,
		},
		{
			name:     "partial usage",
			data:     []byte(`{"usage":{"prompt_tokens":10}}`),
			expected: &Usage{PromptTokens: 10},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractUsage(tt.data)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractFinishReason(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected string
	}{
		{
			name:     "valid finish reason",
			data:     []byte(`{"choices":[{"finish_reason":"stop"}]}`),
			expected: "stop",
		},
		{
			name:     "no finish reason",
			data:     []byte(`{"choices":[{}]}`),
			expected: "",
		},
		{
			name:     "invalid JSON",
			data:     []byte(`invalid`),
			expected: "",
		},
		{
			name:     "empty choices",
			data:     []byte(`{"choices":[]}`),
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractFinishReason(tt.data)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractReasoningContent(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected string
	}{
		{
			name:     "valid reasoning content",
			data:     []byte(`{"choices":[{"message":{"reasoning_content":"Thinking process"}}]}`),
			expected: "Thinking process",
		},
		{
			name:     "no reasoning content",
			data:     []byte(`{"choices":[{"message":{}}]}`),
			expected: "",
		},
		{
			name:     "invalid JSON",
			data:     []byte(`invalid`),
			expected: "",
		},
		{
			name:     "empty choices",
			data:     []byte(`{"choices":[]}`),
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractReasoningContent(tt.data)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractToolCalls(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected []string
	}{
		{
			name: "valid tool calls",
			data: []byte(`{"choices":[{"message":{"tool_calls":[{"id":"1","type":"function","function":{"name":"search","arguments":"{}"}}]}}]}`),
			expected: []string{"search({})"},
		},
		{
			name:     "no tool calls",
			data:     []byte(`{"choices":[{"message":{}}]}`),
			expected: nil,
		},
		{
			name:     "invalid JSON",
			data:     []byte(`invalid`),
			expected: nil,
		},
		{
			name:     "empty choices",
			data:     []byte(`{"choices":[]}`),
			expected: nil,
		},
		{
			name: "multiple tool calls",
			data: []byte(`{"choices":[{"message":{"tool_calls":[{"id":"1","type":"function","function":{"name":"search","arguments":"{}"}},{"id":"2","type":"function","function":{"name":"write","arguments":"{}"}}]}}]}`),
			expected: []string{"search({})", "write({})"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractToolCalls(tt.data)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsJSONModeResponse(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected bool
	}{
		{
			name:     "json_object mode",
			data:     []byte(`{"response_format":{"type":"json_object"}}`),
			expected: true,
		},
		{
			name:     "text mode",
			data:     []byte(`{"response_format":{"type":"text"}}`),
			expected: false,
		},
		{
			name:     "no response format",
			data:     []byte(`{"choices":[]}`),
			expected: false,
		},
		{
			name:     "invalid JSON",
			data:     []byte(`invalid`),
			expected: false,
		},
		{
			name:     "response format without type",
			data:     []byte(`{"response_format":{}}`),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isJSONModeResponse(tt.data)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatErrorResponse(t *testing.T) {
	tests := []struct {
		name        string
		data        []byte
		expectedErr string
	}{
		{
			name:        "valid error response",
			data:        []byte(`{"error":{"message":"Invalid API key","type":"invalid_request_error","code":"invalid_api_key"}}`),
			expectedErr: "API error: Invalid API key (type: invalid_request_error, code: invalid_api_key)",
		},
		{
			name:        "error with only message",
			data:        []byte(`{"error":{"message":"Error occurred"}}`),
			expectedErr: "API error: Error occurred (type: , code: )",
		},
		{
			name:        "invalid JSON",
			data:        []byte(`invalid`),
			expectedErr: "API error: invalid",
		},
		{
			name:        "no error field",
			data:        []byte(`{"choices":[]}`),
			expectedErr: "API error: {\"choices\":[]}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := formatErrorResponse(tt.data)
			assert.Error(t, err)
			assert.Equal(t, tt.expectedErr, err.Error())
		})
	}
}

func TestShouldFormatPretty(t *testing.T) {
	// This function currently always returns true
	result := shouldFormatPretty()
	assert.True(t, result)
}

func TestTrimWhitespace(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "leading and trailing whitespace",
			content:  "  hello world  ",
			expected: "hello world",
		},
		{
			name:     "only whitespace",
			content:  "   ",
			expected: "",
		},
		{
			name:     "no whitespace",
			content:  "hello",
			expected: "hello",
		},
		{
			name:     "newlines and tabs",
			content:  "\n\t  hello  \t\n",
			expected: "hello",
		},
		{
			name:     "empty string",
			content:  "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := trimWhitespace(tt.content)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatModelsResponse(t *testing.T) {
	tests := []struct {
		name        string
		data        []byte
		contains    []string
		expectError bool
	}{
		{
			name: "valid models response",
			data: []byte(`{"object":"list","data":[{"id":"deepseek-v4-pro","object":"model","owned_by":"deepseek","created":1234567890}]}`),
			contains: []string{
				"Object: list",
				"Total models: 1",
				"ID: deepseek-v4-pro",
				"Type: model",
				"Owned by: deepseek",
				"Created: 1234567890",
			},
			expectError: false,
		},
		{
			name:        "invalid JSON",
			data:        []byte(`invalid`),
			contains:    []string{"invalid"},
			expectError: false,
		},
		{
			name: "multiple models",
			data: []byte(`{"object":"list","data":[{"id":"model1","object":"model","owned_by":"deepseek"},{"id":"model2","object":"model","owned_by":"deepseek"}]}`),
			contains: []string{
				"Total models: 2",
				"ID: model1",
				"ID: model2",
			},
			expectError: false,
		},
		{
			name: "model without created timestamp",
			data: []byte(`{"object":"list","data":[{"id":"model1","object":"model","owned_by":"deepseek"}]}`),
			contains: []string{
				"ID: model1",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			err := formatModelsResponse(tt.data)

			// Restore stdout and read captured output
			w.Close()
			os.Stdout = oldStdout
			var buf bytes.Buffer
			io.Copy(&buf, r)
			output := buf.String()

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			for _, expected := range tt.contains {
				assert.Contains(t, output, expected)
			}
		})
	}
}

func TestFormatBalanceResponse(t *testing.T) {
	tests := []struct {
		name        string
		data        []byte
		contains    []string
		expectError bool
	}{
		{
			name: "valid balance response",
			data: []byte(`{"balance":100.0,"total_balance":150.0,"available_balance":120.0,"granted_balance":30.0}`),
			contains: []string{
				"Balance Information:",
				"Balance: 100.000000",
				"Total Balance: 150.000000",
				"Available Balance: 120.000000",
				"Granted Balance: 30.000000",
			},
			expectError: false,
		},
		{
			name:        "invalid JSON",
			data:        []byte(`invalid`),
			contains:    []string{"invalid"},
			expectError: false,
		},
		{
			name: "partial balance",
			data: []byte(`{"balance":50.0}`),
			contains: []string{
				"Balance Information:",
				"Balance: 50.000000",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			err := formatBalanceResponse(tt.data)

			// Restore stdout and read captured output
			w.Close()
			os.Stdout = oldStdout
			var buf bytes.Buffer
			io.Copy(&buf, r)
			output := buf.String()

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			for _, expected := range tt.contains {
				assert.Contains(t, output, expected)
			}
		})
	}
}

func TestParseSSEStream(t *testing.T) {
	client := &Client{
		APIKey: "test-key",
		Base:   "https://api.test.com",
		Client: &http.Client{},
	}

	tests := []struct {
		name           string
		input          string
		completionType string
		contains       []string
		expectError    bool
	}{
		{
			name:           "chat stream with delta content",
			input:          "data: {\"choices\":[{\"delta\":{\"content\":\"Hello\"}}]}\n\ndata: [DONE]\n",
			completionType: "chat",
			contains:       []string{"Hello"},
			expectError:    false,
		},
		{
			name:           "chat stream with finish reason",
			input:          "data: {\"choices\":[{\"finish_reason\":\"stop\"}]}\n\ndata: [DONE]\n",
			completionType: "chat",
			contains:       []string{"finish_reason: stop"},
			expectError:    false,
		},
		{
			name:           "chat stream with usage",
			input:          "data: {\"usage\":{\"prompt_tokens\":10,\"completion_tokens\":20,\"total_tokens\":30}}\n\ndata: [DONE]\n",
			completionType: "chat",
			contains:       []string{"usage: prompt_tokens=10"},
			expectError:    false,
		},
		{
			name:           "fim stream with text",
			input:          "data: {\"choices\":[{\"text\":\"function\"}]}\n\ndata: [DONE]\n",
			completionType: "fim",
			contains:       []string{"function"},
			expectError:    false,
		},
		{
			name:           "fim stream with finish reason",
			input:          "data: {\"choices\":[{\"finish_reason\":\"length\"}]}\n\ndata: [DONE]\n",
			completionType: "fim",
			contains:       []string{"finish_reason: length"},
			expectError:    false,
		},
		{
			name:           "empty lines",
			input:          "\n\n\ndata: [DONE]\n",
			completionType: "chat",
			contains:       []string{},
			expectError:    false,
		},
		{
			name:           "invalid data prefix",
			input:          "invalid: {\"choices\":[]}\n\ndata: [DONE]\n",
			completionType: "chat",
			contains:       []string{},
			expectError:    false,
		},
		{
			name:           "invalid JSON",
			input:          "data: {invalid}\n\ndata: [DONE]\n",
			completionType: "chat",
			contains:       []string{},
			expectError:    false,
		},
		{
			name:           "multiple chunks",
			input:          "data: {\"choices\":[{\"delta\":{\"content\":\"Hello\"}}]}\n\ndata: {\"choices\":[{\"delta\":{\"content\":\" world\"}}]}\n\ndata: [DONE]\n",
			completionType: "chat",
			contains:       []string{"Hello", "world"},
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			reader := strings.NewReader(tt.input)
			err := client.parseSSEStream(reader, tt.completionType)

			// Restore stdout and read captured output
			w.Close()
			os.Stdout = oldStdout
			var buf bytes.Buffer
			io.Copy(&buf, r)
			output := buf.String()

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			for _, expected := range tt.contains {
				assert.Contains(t, output, expected)
			}
		})
	}
}

func TestFormatChatResponse(t *testing.T) {
	tests := []struct {
		name        string
		data        []byte
		showCache   bool
		contains    []string
		expectError bool
	}{
		{
			name:      "valid chat response with content",
			data:      []byte(`{"choices":[{"message":{"content":"Hello world"}}]}`),
			showCache: false,
			contains:  []string{"Hello world"},
		},
		{
			name:      "chat response with usage",
			data:      []byte(`{"choices":[{"message":{"content":"Hello"}}],"usage":{"prompt_tokens":10,"completion_tokens":20,"total_tokens":30}}`),
			showCache: false,
			contains:  []string{"Hello", "usage: prompt_tokens=10"},
		},
		{
			name:      "chat response with cache info",
			data:      []byte(`{"choices":[{"message":{"content":"Hello"}}],"usage":{"prompt_tokens":10,"completion_tokens":20,"total_tokens":30,"prompt_cache_hit_tokens":5,"prompt_cache_miss_tokens":5}}`),
			showCache: true,
			contains:  []string{"cache_hit_tokens=5", "cache_miss_tokens=5"},
		},
		{
			name:      "chat response with reasoning content",
			data:      []byte(`{"choices":[{"message":{"content":"Answer","reasoning_content":"Thinking process"}}]}`),
			showCache: false,
			contains:  []string{"Answer", "Reasoning: Thinking process"},
		},
		{
			name:      "chat response with finish reason",
			data:      []byte(`{"choices":[{"message":{"content":"Hello"},"finish_reason":"stop"}]}`),
			showCache: false,
			contains:  []string{"finish_reason: stop"},
		},
		{
			name:      "invalid JSON",
			data:      []byte(`invalid`),
			showCache: false,
			contains:  []string{"invalid"},
		},
		{
			name:      "empty choices",
			data:      []byte(`{"choices":[]}`),
			showCache: false,
			contains:  []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			err := formatChatResponse(tt.data, tt.showCache)

			// Restore stdout and read captured output
			w.Close()
			os.Stdout = oldStdout
			var buf bytes.Buffer
			io.Copy(&buf, r)
			output := buf.String()

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			for _, expected := range tt.contains {
				assert.Contains(t, output, expected)
			}
		})
	}
}

func TestFormatFIMResponse(t *testing.T) {
	tests := []struct {
		name        string
		data        []byte
		contains    []string
		expectError bool
	}{
		{
			name:     "valid FIM response with text",
			data:     []byte(`{"choices":[{"text":"function test() {}"}]}`),
			contains: []string{"function test() {}"},
		},
		{
			name:     "FIM response with usage",
			data:     []byte(`{"choices":[{"text":"function"}],"usage":{"prompt_tokens":10,"completion_tokens":20,"total_tokens":30}}`),
			contains: []string{"function", "usage: prompt_tokens=10"},
		},
		{
			name:     "FIM response with finish reason",
			data:     []byte(`{"choices":[{"text":"function","finish_reason":"length"}]}`),
			contains: []string{"finish_reason: length"},
		},
		{
			name:     "invalid JSON",
			data:     []byte(`invalid`),
			contains: []string{"invalid"},
		},
		{
			name:     "empty choices",
			data:     []byte(`{"choices":[]}`),
			contains: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			err := formatFIMResponse(tt.data)

			// Restore stdout and read captured output
			w.Close()
			os.Stdout = oldStdout
			var buf bytes.Buffer
			io.Copy(&buf, r)
			output := buf.String()

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			for _, expected := range tt.contains {
				assert.Contains(t, output, expected)
			}
		})
	}
}

func TestFormatJSONModeResponse(t *testing.T) {
	tests := []struct {
		name        string
		data        []byte
		showCache   bool
		contains    []string
		expectError bool
	}{
		{
			name:      "valid JSON mode response",
			data:      []byte(`{"choices":[{"message":{"content":"{\"key\":\"value\"}"}}]}`),
			showCache: false,
			contains:  []string{"key", "value"},
		},
		{
			name:      "JSON mode with pretty printed output",
			data:      []byte(`{"choices":[{"message":{"content":"{\"name\":\"test\",\"value\":123}"}}]}`),
			showCache: false,
			contains:  []string{"name", "test", "value", "123"},
		},
		{
			name:      "JSON mode with usage",
			data:      []byte(`{"choices":[{"message":{"content":"{}"}}],"usage":{"prompt_tokens":10,"completion_tokens":20,"total_tokens":30}}`),
			showCache: false,
			contains:  []string{"usage: prompt_tokens=10"},
		},
		{
			name:      "invalid JSON in content",
			data:      []byte(`{"choices":[{"message":{"content":"not valid json"}}]}`),
			showCache: false,
			contains:  []string{"not valid json"},
		},
		{
			name:      "completely invalid response",
			data:      []byte(`invalid`),
			showCache: false,
			contains:  []string{"invalid"},
		},
		{
			name:      "raw map parsing fallback",
			data:      []byte(`{"choices":[{"message":{"content":"{\"test\":\"data\"}"}}],"response_format":{"type":"json_object"}}`),
			showCache: false,
			contains:  []string{"test", "data"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			err := formatJSONModeResponse(tt.data, tt.showCache)

			// Restore stdout and read captured output
			w.Close()
			os.Stdout = oldStdout
			var buf bytes.Buffer
			io.Copy(&buf, r)
			output := buf.String()

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			for _, expected := range tt.contains {
				assert.Contains(t, output, expected)
			}
		})
	}
}