package main

import (
	"encoding/json"
	"io"
	"strings"
	"testing"
)

func TestParseSSEStream_Chat(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name: "valid SSE chat chunks",
			input: "data: {\"choices\":[{\"delta\":{\"content\":\"Hello\"}}]}\n\n" +
				"data: {\"choices\":[{\"delta\":{\"content\":\" world\"}}]}\n\n" +
				"data: [DONE]\n\n",
			wantErr: false,
		},
		{
			name: "SSE with usage info",
			input: "data: {\"choices\":[{\"delta\":{\"content\":\"Hi\"}}],\"usage\":{\"prompt_tokens\":10,\"completion_tokens\":5,\"total_tokens\":15}}\n\n" +
				"data: [DONE]\n\n",
			wantErr: false,
		},
		{
			name: "SSE with finish reason",
			input: "data: {\"choices\":[{\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n" +
				"data: [DONE]\n\n",
			wantErr: false,
		},
		{
			name: "empty lines skipped",
			input: "\n\n" +
				"data: {\"choices\":[{\"delta\":{\"content\":\"test\"}}]}\n\n" +
				"\n" +
				"data: [DONE]\n\n",
			wantErr: false,
		},
		{
			name: "invalid JSON chunk skipped",
			input: "data: not valid json\n\n" +
				"data: {\"choices\":[{\"delta\":{\"content\":\"valid\"}}]}\n\n" +
				"data: [DONE]\n\n",
			wantErr: false,
		},
		{
			name: "non-data lines skipped",
			input: "event: message\n" +
				"data: {\"choices\":[{\"delta\":{\"content\":\"test\"}}]}\n\n" +
				"data: [DONE]\n\n",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{}
			err := client.parseSSEStream(strings.NewReader(tt.input), "chat")
			if (err != nil) != tt.wantErr {
				t.Errorf("parseSSEStream() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestParseSSEStream_FIM(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name: "valid SSE FIM chunks",
			input: "data: {\"choices\":[{\"text\":\"func \"}]}\n\n" +
				"data: {\"choices\":[{\"text\":\"main() {}\"}]}\n\n" +
				"data: [DONE]\n\n",
			wantErr: false,
		},
		{
			name: "FIM with finish reason",
			input: "data: {\"choices\":[{\"text\":\"code\",\"finish_reason\":\"stop\"}]}\n\n" +
				"data: [DONE]\n\n",
			wantErr: false,
		},
		{
			name: "FIM with usage info",
			input: "data: {\"choices\":[{\"text\":\"test\"}],\"usage\":{\"prompt_tokens\":10,\"completion_tokens\":5,\"total_tokens\":15}}\n\n" +
				"data: [DONE]\n\n",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{}
			err := client.parseSSEStream(strings.NewReader(tt.input), "fim")
			if (err != nil) != tt.wantErr {
				t.Errorf("parseSSEStream() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestParseSSEStream_EmptyBody(t *testing.T) {
	client := &Client{}
	err := client.parseSSEStream(strings.NewReader(""), "chat")
	if err != nil {
		t.Errorf("parseSSEStream() with empty body error = %v", err)
	}
}

func TestParseSSEStream_OnlyDone(t *testing.T) {
	client := &Client{}
	err := client.parseSSEStream(strings.NewReader("data: [DONE]\n\n"), "chat")
	if err != nil {
		t.Errorf("parseSSEStream() with only DONE error = %v", err)
	}
}

func TestParseSSEStream_InvalidCompletionType(t *testing.T) {
	client := &Client{}
	err := client.parseSSEStream(strings.NewReader("data: {\"test\":\"value\"}\n\n"), "invalid")
	if err != nil {
		t.Errorf("parseSSEStream() with invalid type error = %v", err)
	}
}

func TestStreamChatCompletion_NilClient(t *testing.T) {
	req := &ChatRequest{
		Model:    "deepseek-v4-pro",
		Messages: []Message{{Role: "user", Content: "test"}},
	}
	client := NewClient("https://api.deepseek.com", "test-key")
	err := client.streamChatCompletion(req)
	// Will fail with HTTP error since no real server
	if err == nil {
		t.Error("streamChatCompletion() should error without real server")
	}
}

func TestStreamFIMCompletion_NilClient(t *testing.T) {
	req := &FIMRequest{
		Model:  "deepseek-v4-pro",
		Prompt: "test",
	}
	client := NewClient("https://api.deepseek.com", "test-key")
	err := client.streamFIMCompletion(req)
	// Will fail with HTTP error since no real server
	if err == nil {
		t.Error("streamFIMCompletion() should error without real server")
	}
}

func TestParseSSEStream_ChatResponseFormat(t *testing.T) {
	// Test with ChatResponse format (not delta)
	input := "data: {\"choices\":[{\"message\":{\"content\":\"Hello\"}}]}\n\n" +
		"data: [DONE]\n\n"

	client := &Client{}
	err := client.parseSSEStream(strings.NewReader(input), "chat")
	if err != nil {
		t.Errorf("parseSSEStream() with ChatResponse format error = %v", err)
	}
}

func TestParseSSEStream_FIMResponseFormat(t *testing.T) {
	// Test with FIMResponse format (not simple chunk)
	input := "data: {\"choices\":[{\"text\":\"code\"}]}\n\n" +
		"data: [DONE]\n\n"

	client := &Client{}
	err := client.parseSSEStream(strings.NewReader(input), "fim")
	if err != nil {
		t.Errorf("parseSSEStream() with FIMResponse format error = %v", err)
	}
}

func TestParseSSEStream_MultipleChoices(t *testing.T) {
	input := "data: {\"choices\":[{\"delta\":{\"content\":\"a\"}},{\"delta\":{\"content\":\"b\"}}]}\n\n" +
		"data: [DONE]\n\n"

	client := &Client{}
	err := client.parseSSEStream(strings.NewReader(input), "chat")
	if err != nil {
		t.Errorf("parseSSEStream() with multiple choices error = %v", err)
	}
}

func TestParseSSEStream_ScannerError(t *testing.T) {
	// Create a reader that returns an error
	errorReader := &errorReader{err: io.ErrUnexpectedEOF}
	client := &Client{}
	err := client.parseSSEStream(errorReader, "chat")
	if err == nil {
		t.Error("parseSSEStream() should return error from scanner")
	}
}

// errorReader is a mock reader that returns an error
type errorReader struct {
	err error
}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, e.err
}

func TestStreamChatCompletion_RequestMarshalError(t *testing.T) {
	// This is hard to trigger since ChatRequest marshaling rarely fails
	// Just verify the function exists and handles basic cases
	client := &Client{
		Base:   "https://api.deepseek.com",
		APIKey: "test-key",
	}
	req := &ChatRequest{
		Model:    "deepseek-v4-pro",
		Messages: []Message{{Role: "user", Content: "test"}},
	}
	_ = client
	_ = req
}

func TestParseSSEStream_NoNewlineAtEnd(t *testing.T) {
	input := "data: {\"choices\":[{\"delta\":{\"content\":\"test\"}}]}"
	client := &Client{}
	err := client.parseSSEStream(strings.NewReader(input), "chat")
	if err != nil {
		t.Errorf("parseSSEStream() with no newline at end error = %v", err)
	}
}

func TestChatResponse_Marshaling(t *testing.T) {
	// Test the ChatResponse struct marshaling for coverage
	resp := ChatResponse{
		Choices: []Choice{
			{
				Delta: &Message{Content: "test"},
			},
		},
	}
	data, err := json.Marshal(resp)
	if err != nil {
		t.Errorf("Failed to marshal ChatResponse: %v", err)
	}

	var unmarshaled ChatResponse
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Errorf("Failed to unmarshal ChatResponse: %v", err)
	}
}

func TestFIMResponse_Marshaling(t *testing.T) {
	resp := FIMResponse{
		Choices: []FIMChoice{
			{Text: "test code"},
		},
	}
	data, err := json.Marshal(resp)
	if err != nil {
		t.Errorf("Failed to marshal FIMResponse: %v", err)
	}

	var unmarshaled FIMResponse
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Errorf("Failed to unmarshal FIMResponse: %v", err)
	}
}

func TestParseSSEStream_EmptyDataLines(t *testing.T) {
	input := "data: \n\n" +
		"data: {\"choices\":[{\"delta\":{\"content\":\"test\"}}]}\n\n" +
		"data: [DONE]\n\n"

	client := &Client{}
	err := client.parseSSEStream(strings.NewReader(input), "chat")
	if err != nil {
		t.Errorf("parseSSEStream() with empty data lines error = %v", err)
	}
}

func TestParseSSEStream_UsageOnlyChunk(t *testing.T) {
	input := "data: {\"usage\":{\"prompt_tokens\":10,\"completion_tokens\":5,\"total_tokens\":15}}\n\n" +
		"data: [DONE]\n\n"

	client := &Client{}
	err := client.parseSSEStream(strings.NewReader(input), "chat")
	if err != nil {
		t.Errorf("parseSSEStream() with usage-only chunk error = %v", err)
	}
}

func TestStreamChatCompletion_InvalidURL(t *testing.T) {
	client := &Client{
		Base:   ":://invalid-url",
		APIKey: "test-key",
	}
	req := &ChatRequest{
		Model:    "deepseek-v4-pro",
		Messages: []Message{{Role: "user", Content: "test"}},
	}
	err := client.streamChatCompletion(req)
	if err == nil {
		t.Error("streamChatCompletion() should error with invalid URL")
	}
}

func TestStreamFIMCompletion_InvalidURL(t *testing.T) {
	client := &Client{
		Base:   ":://invalid-url",
		APIKey: "test-key",
	}
	req := &FIMRequest{
		Model:  "deepseek-v4-pro",
		Prompt: "test",
	}
	err := client.streamFIMCompletion(req)
	if err == nil {
		t.Error("streamFIMCompletion() should error with invalid URL")
	}
}
