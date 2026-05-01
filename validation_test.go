package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateChatRequest(t *testing.T) {
	tests := []struct {
		name    string
		req     *ChatRequest
		wantErr bool
	}{
		{
			name: "valid request",
			req: &ChatRequest{
				Model:    "deepseek-v4-pro",
				Messages: []Message{{Role: "user", Content: "hello"}},
			},
			wantErr: false,
		},
		{
			name: "missing model",
			req: &ChatRequest{
				Messages: []Message{{Role: "user", Content: "hello"}},
			},
			wantErr: true,
		},
		{
			name: "missing messages",
			req: &ChatRequest{
				Model: "deepseek-v4-pro",
			},
			wantErr: true,
		},
		{
			name: "invalid thinking type",
			req: &ChatRequest{
				Model:    "deepseek-v4-pro",
				Messages: []Message{{Role: "user", Content: "hello"}},
				Thinking:  &ThinkingConfig{Type: "invalid"},
			},
			wantErr: true,
		},
		{
			name: "valid thinking enabled",
			req: &ChatRequest{
				Model:    "deepseek-v4-pro",
				Messages: []Message{{Role: "user", Content: "hello"}},
				Thinking:  &ThinkingConfig{Type: "enabled"},
			},
			wantErr: false,
		},
		{
			name: "invalid reasoning effort",
			req: &ChatRequest{
				Model:          "deepseek-v4-pro",
				Messages:        []Message{{Role: "user", Content: "hello"}},
				ReasoningEffort: "medium",
			},
			wantErr: true,
		},
		{
			name: "valid reasoning effort max",
			req: &ChatRequest{
				Model:          "deepseek-v4-pro",
				Messages:        []Message{{Role: "user", Content: "hello"}},
				ReasoningEffort: "max",
			},
			wantErr: false,
		},
		{
			name: "temperature too high",
			req: &ChatRequest{
				Model:      "deepseek-v4-pro",
				Messages:    []Message{{Role: "user", Content: "hello"}},
				Temperature: float64Ptr(2.5),
			},
			wantErr: true,
		},
		{
			name: "valid temperature",
			req: &ChatRequest{
				Model:      "deepseek-v4-pro",
				Messages:    []Message{{Role: "user", Content: "hello"}},
				Temperature: float64Ptr(0.5),
			},
			wantErr: false,
		},
		{
			name: "top_p too high",
			req: &ChatRequest{
				Model:   "deepseek-v4-pro",
				Messages: []Message{{Role: "user", Content: "hello"}},
				TopP:     float64Ptr(1.5),
			},
			wantErr: true,
		},
		{
			name: "valid top_p",
			req: &ChatRequest{
				Model:   "deepseek-v4-pro",
				Messages: []Message{{Role: "user", Content: "hello"}},
				TopP:     float64Ptr(0.8),
			},
			wantErr: false,
		},
		{
			name: "frequency_penalty too high",
			req: &ChatRequest{
				Model:           "deepseek-v4-pro",
				Messages:         []Message{{Role: "user", Content: "hello"}},
				FrequencyPenalty: float64Ptr(2.5),
			},
			wantErr: true,
		},
		{
			name: "valid frequency_penalty",
			req: &ChatRequest{
				Model:           "deepseek-v4-pro",
				Messages:         []Message{{Role: "user", Content: "hello"}},
				FrequencyPenalty: float64Ptr(1.0),
			},
			wantErr: false,
		},
		{
			name: "presence_penalty too low",
			req: &ChatRequest{
				Model:          "deepseek-v4-pro",
				Messages:        []Message{{Role: "user", Content: "hello"}},
				PresencePenalty: float64Ptr(-2.5),
			},
			wantErr: true,
		},
		{
			name: "valid presence_penalty",
			req: &ChatRequest{
				Model:          "deepseek-v4-pro",
				Messages:        []Message{{Role: "user", Content: "hello"}},
				PresencePenalty: float64Ptr(-1.0),
			},
			wantErr: false,
		},
		{
			name: "invalid response_format type",
			req: &ChatRequest{
				Model:    "deepseek-v4-pro",
				Messages:  []Message{{Role: "user", Content: "hello"}},
				ResponseFormat: &ResponseFormat{Type: "invalid"},
			},
			wantErr: true,
		},
		{
			name: "valid response_format json_object",
			req: &ChatRequest{
				Model:    "deepseek-v4-pro",
				Messages:  []Message{{Role: "user", Content: "hello"}},
				ResponseFormat: &ResponseFormat{Type: "json_object"},
			},
			wantErr: false,
		},
		{
			name: "top_logprobs too high",
			req: &ChatRequest{
				Model:      "deepseek-v4-pro",
				Messages:    []Message{{Role: "user", Content: "hello"}},
				TopLogprobs: intPtr(25),
			},
			wantErr: true,
		},
		{
			name: "valid top_logprobs",
			req: &ChatRequest{
				Model:      "deepseek-v4-pro",
				Messages:    []Message{{Role: "user", Content: "hello"}},
				TopLogprobs: intPtr(10),
			},
			wantErr: false,
		},
		{
			name: "invalid tool type",
			req: &ChatRequest{
				Model:    "deepseek-v4-pro",
				Messages:  []Message{{Role: "user", Content: "hello"}},
				Tools: []Tool{{Type: "invalid", Function: ToolFunction{Name: "test"}}},
			},
			wantErr: true,
		},
		{
			name: "missing tool function name",
			req: &ChatRequest{
				Model:    "deepseek-v4-pro",
				Messages:  []Message{{Role: "user", Content: "hello"}},
				Tools: []Tool{{Type: "function", Function: ToolFunction{Name: ""}}},
			},
			wantErr: true,
		},
		{
			name: "valid tools",
			req: &ChatRequest{
				Model:    "deepseek-v4-pro",
				Messages:  []Message{{Role: "user", Content: "hello"}},
				Tools: []Tool{{Type: "function", Function: ToolFunction{Name: "test"}}},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateChatRequest(tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateChatRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateChatRequestInvalidTemperature(t *testing.T) {
	// Temperature out of range
	temp := 3.0 // > 2.0
	req := &ChatRequest{
		Model:       "deepseek-v4-pro",
		Messages:    []Message{{Role: "user", Content: "hello"}},
		Temperature: &temp,
	}
	
	err := ValidateChatRequest(req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "temperature")
}

func TestValidateChatRequestInvalidTopP(t *testing.T) {
	// TopP out of range
	topP := 1.5 // > 1.0
	req := &ChatRequest{
		Model:     "deepseek-v4-pro",
		Messages:  []Message{{Role: "user", Content: "hello"}},
		TopP:      &topP,
	}
	
	err := ValidateChatRequest(req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "top_p")
}

func TestValidateChatRequestInvalidThinking(t *testing.T) {
	// Invalid thinking value
	req := &ChatRequest{
		Model:    "deepseek-v4-pro",
		Messages: []Message{{Role: "user", Content: "hello"}},
		Thinking:  &ThinkingConfig{Type: "invalid"},
	}
	
	err := ValidateChatRequest(req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "thinking")
}

func TestValidateChatRequestInvalidReasoningEffort(t *testing.T) {
	// Invalid reasoning effort
	req := &ChatRequest{
		Model:           "deepseek-v4-pro",
		Messages:        []Message{{Role: "user", Content: "hello"}},
		ReasoningEffort: "invalid",
	}
	
	err := ValidateChatRequest(req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "reasoning_effort")
}

func TestValidateChatRequestInvalidToolChoice(t *testing.T) {
	// Invalid tool choice
	req := &ChatRequest{
		Model:      "deepseek-v4-pro",
		Messages:   []Message{{Role: "user", Content: "hello"}},
		ToolChoice: "invalid",
	}
	
	err := ValidateChatRequest(req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "tool_choice")
}

func TestValidateChatRequestValidToolChoiceFunction(t *testing.T) {
	// Valid tool choice as JSON
	req := &ChatRequest{
		Model:      "deepseek-v4-pro",
		Messages:   []Message{{Role: "user", Content: "hello"}},
		ToolChoice: map[string]interface{}{"type": "function", "function": map[string]interface{}{"name": "test"}},
	}
	
	err := ValidateChatRequest(req)
	assert.NoError(t, err)
}

func TestValidateFIMRequest(t *testing.T) {
	tests := []struct {
		name    string
		req     *FIMRequest
		wantErr bool
	}{
		{
			name: "valid request",
			req: &FIMRequest{
				Model:  "deepseek-v4-pro",
				Prompt: "test",
			},
			wantErr: false,
		},
		{
			name: "missing model",
			req: &FIMRequest{
				Prompt: "test",
			},
			wantErr: true,
		},
		{
			name: "missing prompt",
			req: &FIMRequest{
				Model: "deepseek-v4-pro",
			},
			wantErr: true,
		},
		{
			name: "valid with max tokens",
			req: &FIMRequest{
				Model:    "deepseek-v4-pro",
				Prompt:   "test",
				MaxTokens: intPtr(100),
			},
			wantErr: false,
		},
		{
			name: "max tokens too high",
			req: &FIMRequest{
				Model:    "deepseek-v4-pro",
				Prompt:   "test",
				MaxTokens: intPtr(5000),
			},
			wantErr: true,
		},
		{
			name: "valid with temperature",
			req: &FIMRequest{
				Model:      "deepseek-v4-pro",
				Prompt:     "test",
				Temperature: float64Ptr(0.5),
			},
			wantErr: false,
		},
		{
			name: "temperature too high",
			req: &FIMRequest{
				Model:      "deepseek-v4-pro",
				Prompt:     "test",
				Temperature: float64Ptr(2.5),
			},
			wantErr: true,
		},
		{
			name: "valid with top_p",
			req: &FIMRequest{
				Model: "deepseek-v4-pro",
				Prompt: "test",
				TopP:   float64Ptr(0.8),
			},
			wantErr: false,
		},
		{
			name: "top_p too high",
			req: &FIMRequest{
				Model: "deepseek-v4-pro",
				Prompt: "test",
				TopP:   float64Ptr(1.5),
			},
			wantErr: true,
		},
		{
			name: "valid with frequency_penalty",
			req: &FIMRequest{
				Model:           "deepseek-v4-pro",
				Prompt:          "test",
				FrequencyPenalty: float64Ptr(1.0),
			},
			wantErr: false,
		},
		{
			name: "frequency_penalty too high",
			req: &FIMRequest{
				Model:           "deepseek-v4-pro",
				Prompt:          "test",
				FrequencyPenalty: float64Ptr(2.5),
			},
			wantErr: true,
		},
		{
			name: "valid with presence_penalty",
			req: &FIMRequest{
				Model:          "deepseek-v4-pro",
				Prompt:         "test",
				PresencePenalty: float64Ptr(-1.0),
			},
			wantErr: false,
		},
		{
			name: "presence_penalty too low",
			req: &FIMRequest{
				Model:          "deepseek-v4-pro",
				Prompt:         "test",
				PresencePenalty: float64Ptr(-2.5),
			},
			wantErr: true,
		},
		{
			name: "valid with logprobs",
			req: &FIMRequest{
				Model:   "deepseek-v4-pro",
				Prompt:  "test",
				Logprobs: intPtr(5),
			},
			wantErr: false,
		},
		{
			name: "logprobs too high",
			req: &FIMRequest{
				Model:   "deepseek-v4-pro",
				Prompt:  "test",
				Logprobs: intPtr(25),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFIMRequest(tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateFIMRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func intPtr(i int) *int {
	return &i
}

func float64Ptr(f float64) *float64 {
	return &f
}

func TestValidateMessage(t *testing.T) {
	// Valid message
	msg := Message{
		Role:    "user",
		Content: "hello",
	}
	err := validateMessage(&msg, 0)
	assert.NoError(t, err)
}

func TestValidateMessageInvalidRole(t *testing.T) {
	// Invalid role
	msg := Message{
		Role:    "invalid",
		Content: "hello",
	}
	err := validateMessage(&msg, 0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "role")
}

func TestValidateMessageMissingContent(t *testing.T) {
	// Missing content (nil content)
	msg := Message{
		Role: "user",
	}
	err := validateMessage(&msg, 0)
	assert.Error(t, err)
}


func TestValidateStop(t *testing.T) {
	tests := []struct {
		name    string
		stop    interface{}
		wantErr bool
	}{
		{
			name:    "valid string",
			stop:    "STOP",
			wantErr: false,
		},
		{
			name:    "empty string",
			stop:    "",
			wantErr: true,
		},
		{
			name:    "valid string array",
			stop:    []string{"STOP1", "STOP2"},
			wantErr: false,
		},
		{
			name:    "empty string array",
			stop:    []string{},
			wantErr: true,
		},
		{
			name:    "too many strings",
			stop:    []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11", "12", "13", "14", "15", "16", "17"},
			wantErr: true,
		},
		{
			name:    "array with empty string",
			stop:    []string{"STOP1", ""},
			wantErr: true,
		},
		{
			name:    "invalid type int",
			stop:    123,
			wantErr: true,
		},
		{
			name:    "invalid type bool",
			stop:    true,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateStop(tt.stop)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateStop() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateToolChoice(t *testing.T) {
	tests := []struct {
		name    string
		tc      interface{}
		wantErr bool
	}{
		{
			name:    "valid string none",
			tc:      "none",
			wantErr: false,
		},
		{
			name:    "valid string auto",
			tc:      "auto",
			wantErr: false,
		},
		{
			name:    "valid string required",
			tc:      "required",
			wantErr: false,
		},
		{
			name:    "invalid string",
			tc:      "invalid",
			wantErr: true,
		},
		{
			name:    "valid function object",
			tc:      map[string]interface{}{"type": "function", "function": map[string]interface{}{"name": "test"}},
			wantErr: false,
		},
		{
			name:    "function object missing type",
			tc:      map[string]interface{}{"function": map[string]interface{}{"name": "test"}},
			wantErr: true,
		},
		{
			name:    "function object missing name",
			tc:      map[string]interface{}{"type": "function", "function": map[string]interface{}{}},
			wantErr: true,
		},
		{
			name:    "invalid type int",
			tc:      123,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateToolChoice(tt.tc)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateToolChoice() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
