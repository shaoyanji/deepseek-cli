package main

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateChatRequest(t *testing.T) {
	// Valid request
	req := &ChatRequest{
		Model:    "deepseek-v4-pro",
		Messages: []Message{
			{Role: "user", Content: "hello"},
		},
	}
	
	err := ValidateChatRequest(req)
	assert.NoError(t, err)
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
	// Valid FIM request
	req := &FIMRequest{
		Model:  "deepseek-v4-pro",
		Prompt: "func main() {",
	}
	
	err := ValidateFIMRequest(req)
	assert.NoError(t, err)
}

func TestValidateFIMRequestInvalidMaxTokens(t *testing.T) {
	// Max tokens > 4096 for FIM
	maxTokens := 5000
	req := &FIMRequest{
		Model:     "deepseek-v4-pro",
		Prompt:    "func main() {",
		MaxTokens: &maxTokens,
	}
	
	err := ValidateFIMRequest(req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "max_tokens")
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
	// Valid stop sequences
	stop := []string{"\n", "END"}
	err := validateStop(stop)
	assert.NoError(t, err)
}

func TestValidateStopTooMany(t *testing.T) {
	// More than 16 stop sequences
	stop := make([]string, 20)
	for i := range stop {
		stop[i] = fmt.Sprintf("stop%d", i)
	}
	err := validateStop(stop)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "stop")
}

func TestValidateToolChoice(t *testing.T) {
	// Valid string values
	validChoices := []string{"none", "auto", "required"}
	for _, choice := range validChoices {
		err := validateToolChoice(choice)
		assert.NoError(t, err)
	}
	
	// Valid JSON object
	jsonObj := map[string]interface{}{
		"type": "function",
		"function": map[string]interface{}{"name": "test"},
	}
	err := validateToolChoice(jsonObj)
	assert.NoError(t, err)
	
	// Invalid
	err = validateToolChoice(123) // not string or map
	assert.Error(t, err)
}
