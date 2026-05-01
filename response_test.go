package main

import (
	"strings"
	"testing"
)

func TestFormatChatResponse(t *testing.T) {
	tests := []struct {
		name      string
		data      string
		showCache bool
		wantErr   bool
	}{
		{
			name: "valid chat response with content",
			data: `{"choices":[{"message":{"content":"Hello!"}}],"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}}`,
			showCache: false,
			wantErr:   false,
		},
		{
			name: "valid chat response with reasoning",
			data: `{"choices":[{"message":{"content":"Answer","reasoning_content":"Thinking..."}}],"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}}`,
			showCache: false,
			wantErr:   false,
		},
		{
			name: "chat response with tool calls",
			data: `{"choices":[{"message":{"role":"assistant","content":"","tool_calls":[{"id":"call1","type":"function","function":{"name":"get_weather","arguments":"{\"location\":\"NYC\"}"}}]}}]}`,
			showCache: false,
			wantErr:   false,
		},
		{
			name: "invalid JSON falls back to raw output",
			data: `not valid json`,
			showCache: false,
			wantErr:   false,
		},
		{
			name: "show cache metrics",
			data: `{"choices":[{"message":{"content":"Hi"}}],"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15,"prompt_cache_hit_tokens":3,"prompt_cache_miss_tokens":7}}`,
			showCache: true,
			wantErr:   false,
		},
		{
			name: "with finish reason",
			data: `{"choices":[{"message":{"content":"Done"},"finish_reason":"stop"}]}`,
			showCache: false,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := formatChatResponse([]byte(tt.data), tt.showCache)
			if (err != nil) != tt.wantErr {
				t.Errorf("formatChatResponse() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFormatFIMResponse(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		wantErr bool
	}{
		{
			name:    "valid FIM response",
			data:    `{"choices":[{"text":"func main() {}"}],"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}}`,
			wantErr: false,
		},
		{
			name:    "invalid JSON falls back to raw output",
			data:    `not valid json`,
			wantErr: false,
		},
		{
			name:    "with finish reason",
			data:    `{"choices":[{"text":"code","finish_reason":"stop"}]}`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := formatFIMResponse([]byte(tt.data))
			if (err != nil) != tt.wantErr {
				t.Errorf("formatFIMResponse() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFormatJSONModeResponse(t *testing.T) {
	tests := []struct {
		name      string
		data      string
		showCache bool
		wantErr   bool
	}{
		{
			name:      "valid JSON mode response",
			data:      `{"choices":[{"message":{"content":"{\"key\":\"value\"}"}}]}`,
			showCache: false,
			wantErr:   false,
		},
		{
			name:      "pretty prints JSON content",
			data:      `{"choices":[{"message":{"content":"{\"name\":\"test\",\"value\":123}"}}]}`,
			showCache: false,
			wantErr:   false,
		},
		{
			name:      "invalid JSON falls back gracefully",
			data:      `not valid json`,
			showCache: false,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := formatJSONModeResponse([]byte(tt.data), tt.showCache)
			if (err != nil) != tt.wantErr {
				t.Errorf("formatJSONModeResponse() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestExtractContent(t *testing.T) {
	tests := []struct {
		name           string
		data           string
		completionType string
		want           string
	}{
		{
			name:           "extract chat content",
			data:           `{"choices":[{"message":{"content":"Hello world"}}]}`,
			completionType: "chat",
			want:           "Hello world",
		},
		{
			name:           "extract FIM text",
			data:           `{"choices":[{"text":"func main() {}"}]}`,
			completionType: "fim",
			want:           "func main() {}",
		},
		{
			name:           "invalid JSON returns raw",
			data:           `not json`,
			completionType: "chat",
			want:           "not json",
		},
		{
			name:           "empty choices",
			data:           `{"choices":[]}`,
			completionType: "chat",
			want:           `{"choices":[]}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractContent([]byte(tt.data), tt.completionType)
			if got != tt.want {
				t.Errorf("extractContent() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractUsage(t *testing.T) {
	tests := []struct {
		name string
		data string
		want *Usage
	}{
		{
			name: "valid usage",
			data: `{"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}}`,
			want: &Usage{PromptTokens: 10, CompletionTokens: 5, TotalTokens: 15},
		},
		{
			name: "no usage in response",
			data: `{"choices":[]}`,
			want: nil,
		},
		{
			name: "invalid JSON",
			data: `not json`,
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractUsage([]byte(tt.data))
			if tt.want == nil {
				if got != nil {
					t.Errorf("extractUsage() = %v, want nil", got)
				}
			} else if got == nil || got.PromptTokens != tt.want.PromptTokens || got.CompletionTokens != tt.want.CompletionTokens || got.TotalTokens != tt.want.TotalTokens {
				t.Errorf("extractUsage() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractFinishReason(t *testing.T) {
	tests := []struct {
		name string
		data string
		want string
	}{
		{
			name: "extract finish reason",
			data: `{"choices":[{"finish_reason":"stop"}]}`,
			want: "stop",
		},
		{
			name: "no finish reason",
			data: `{"choices":[{"message":{"content":"hi"}}]}`,
			want: "",
		},
		{
			name: "invalid JSON",
			data: `not json`,
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractFinishReason([]byte(tt.data))
			if got != tt.want {
				t.Errorf("extractFinishReason() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractReasoningContent(t *testing.T) {
	tests := []struct {
		name string
		data string
		want string
	}{
		{
			name: "extract reasoning content",
			data: `{"choices":[{"message":{"reasoning_content":"I'm thinking..."}}]}`,
			want: "I'm thinking...",
		},
		{
			name: "no reasoning content",
			data: `{"choices":[{"message":{"content":"hi"}}]}`,
			want: "",
		},
		{
			name: "invalid JSON",
			data: `not json`,
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractReasoningContent([]byte(tt.data))
			if got != tt.want {
				t.Errorf("extractReasoningContent() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractToolCalls(t *testing.T) {
	tests := []struct {
		name string
		data string
		want []string
	}{
		{
			name: "extract tool calls",
			data: `{"choices":[{"message":{"tool_calls":[{"function":{"name":"get_weather","arguments":"{\"loc\":\"NYC\"}"}}]}}]}`,
			want: []string{"get_weather({\"loc\":\"NYC\"})"},
		},
		{
			name: "no tool calls",
			data: `{"choices":[{"message":{"content":"hi"}}]}`,
			want: nil,
		},
		{
			name: "invalid JSON",
			data: `not json`,
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractToolCalls([]byte(tt.data))
			if len(got) != len(tt.want) {
				t.Errorf("extractToolCalls() = %v, want %v", got, tt.want)
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("extractToolCalls()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestIsJSONModeResponse(t *testing.T) {
	tests := []struct {
		name string
		data string
		want bool
	}{
		{
			name: "JSON mode response",
			data: `{"response_format":{"type":"json_object"}}`,
			want: true,
		},
		{
			name: "not JSON mode",
			data: `{"model":"deepseek-v4-pro"}`,
			want: false,
		},
		{
			name: "invalid JSON",
			data: `not json`,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isJSONModeResponse([]byte(tt.data))
			if got != tt.want {
				t.Errorf("isJSONModeResponse() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFormatErrorResponse(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		wantErr string
	}{
		{
			name:    "API error with message",
			data:    `{"error":{"message":"Invalid API key","type":"authentication_error","code":"invalid_api_key"}}`,
			wantErr: "API error: Invalid API key (type: authentication_error, code: invalid_api_key)",
		},
		{
			name:    "API error without code",
			data:    `{"error":{"message":"Rate limited"}}`,
			wantErr: "API error: Rate limited (type: , code: )",
		},
		{
			name:    "raw error message",
			data:    `some error occurred`,
			wantErr: "API error: some error occurred",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := formatErrorResponse([]byte(tt.data))
			if err == nil || err.Error() != tt.wantErr {
				t.Errorf("formatErrorResponse() error = %v, want %q", err, tt.wantErr)
			}
		})
	}
}

func TestShouldFormatPretty(t *testing.T) {
	result := shouldFormatPretty()
	if !result {
		t.Errorf("shouldFormatPretty() = false, want true")
	}
}

func TestTrimWhitespace(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "trim spaces",
			input: "  hello  ",
			want:  "hello",
		},
		{
			name:  "trim newlines",
			input: "\nworld\n",
			want:  "world",
		},
		{
			name:  "no whitespace",
			input: "test",
			want:  "test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := trimWhitespace(tt.input)
			if got != tt.want {
				t.Errorf("trimWhitespace() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatModelsResponse(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		wantErr bool
	}{
		{
			name:    "valid models response",
			data:    `{"object":"list","data":[{"id":"deepseek-v4-pro","object":"model","owned_by":"deepseek"}]}`,
			wantErr: false,
		},
		{
			name:    "invalid JSON falls back to raw",
			data:    `not json`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := formatModelsResponse([]byte(tt.data))
			if (err != nil) != tt.wantErr {
				t.Errorf("formatModelsResponse() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFormatBalanceResponse(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		wantErr bool
	}{
		{
			name:    "valid balance response",
			data:    `{"balance":100.50,"total_balance":150.75,"available_balance":100.50,"granted_balance":50.25}`,
			wantErr: false,
		},
		{
			name:    "invalid JSON falls back to raw",
			data:    `not json`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := formatBalanceResponse([]byte(tt.data))
			if (err != nil) != tt.wantErr {
				t.Errorf("formatBalanceResponse() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFormatChatResponse_EmptyContent(t *testing.T) {
	data := `{"choices":[{"message":{"content":""}}]}`
	err := formatChatResponse([]byte(data), false)
	if err != nil {
		t.Errorf("formatChatResponse() with empty content error = %v", err)
	}
}

func TestFormatChatResponse_NilContent(t *testing.T) {
	data := `{"choices":[{"message":{"content":null}}]}`
	err := formatChatResponse([]byte(data), false)
	if err != nil {
		t.Errorf("formatChatResponse() with nil content error = %v", err)
	}
}

func TestExtractContent_EmptyData(t *testing.T) {
	result := extractContent([]byte(""), "chat")
	if result != "" {
		t.Errorf("extractContent() with empty data = %q, want empty string", result)
	}
}

func TestFormatJSONModeResponse_EmptyContent(t *testing.T) {
	data := `{"choices":[{"message":{"content":""}}]}`
	err := formatJSONModeResponse([]byte(data), false)
	if err != nil {
		t.Errorf("formatJSONModeResponse() with empty content error = %v", err)
	}
}

func TestFormatJSONModeResponse_InvalidJSONInContent(t *testing.T) {
	// Content is invalid JSON, should fall back to printing content
	data := `{"choices":[{"message":{"content":"not valid json"}}]}`
	err := formatJSONModeResponse([]byte(data), false)
	if err != nil {
		t.Errorf("formatJSONModeResponse() with invalid JSON in content error = %v", err)
	}
}

func TestFormatJSONModeResponse_NoChoices(t *testing.T) {
	data := `{"choices":[]}`
	err := formatJSONModeResponse([]byte(data), false)
	if err != nil {
		t.Errorf("formatJSONModeResponse() with no choices error = %v", err)
	}
}

func TestFormatJSONModeResponse_NoMessage(t *testing.T) {
	data := `{"choices":[{}]}`
	err := formatJSONModeResponse([]byte(data), false)
	if err != nil {
		t.Errorf("formatJSONModeResponse() with no message error = %v", err)
	}
}

func TestFormatJSONModeResponse_WithCacheAndUsage(t *testing.T) {
	data := `{"choices":[{"message":{"content":"{\"a\":1}"}}],"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15,"prompt_cache_hit_tokens":3,"prompt_cache_miss_tokens":7}}`
	err := formatJSONModeResponse([]byte(data), true)
	if err != nil {
		t.Errorf("formatJSONModeResponse() with cache error = %v", err)
	}
}

func TestLongContent(t *testing.T) {
	longContent := strings.Repeat("a", 10000)
	data := `{"choices":[{"message":{"content":"` + longContent + `"}}]}`
	err := formatChatResponse([]byte(data), false)
	if err != nil {
		t.Errorf("formatChatResponse() with long content error = %v", err)
	}
}

func TestFormatJSONModeResponse_ValidJSONContent(t *testing.T) {
	// Content is valid JSON that can be pretty-printed
	data := `{"choices":[{"message":{"content":"{\"name\":\"test\",\"value\":123}"}}]}`
	err := formatJSONModeResponse([]byte(data), false)
	if err != nil {
		t.Errorf("formatJSONModeResponse() with valid JSON content error = %v", err)
	}
}

func TestFormatJSONModeResponse_InvalidJSONContent(t *testing.T) {
	// Content is NOT valid JSON - should print as-is
	data := `{"choices":[{"message":{"content":"not valid json"}}]}`
	err := formatJSONModeResponse([]byte(data), false)
	if err != nil {
		t.Errorf("formatJSONModeResponse() with invalid JSON content error = %v", err)
	}
}

func TestFormatJSONModeResponse_WithUsage(t *testing.T) {
	data := `{"choices":[{"message":{"content":"{\"a\":1}"}}],"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}}`
	err := formatJSONModeResponse([]byte(data), false)
	if err != nil {
		t.Errorf("formatJSONModeResponse() with usage error = %v", err)
	}
}

func TestFormatJSONModeResponse_WithCacheUsage(t *testing.T) {
	data := `{"choices":[{"message":{"content":"{\"a\":1}"}}],"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15,"prompt_cache_hit_tokens":3,"prompt_cache_miss_tokens":7}}`
	err := formatJSONModeResponse([]byte(data), true)
	if err != nil {
		t.Errorf("formatJSONModeResponse() with cache usage error = %v", err)
	}
}

func TestFormatJSONModeResponse_NilMessage(t *testing.T) {
	// Test when json.Unmarshal(data, &resp) succeeds but message content handling
	data := `{"choices":[{"message":{"content":null}}]}`
	err := formatJSONModeResponse([]byte(data), false)
	if err != nil {
		t.Errorf("formatJSONModeResponse() with null content error = %v", err)
	}
}
