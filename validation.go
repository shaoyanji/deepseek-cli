package main

import (
	"fmt"
)

// ValidationError represents a validation error with context
type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// ValidateChatRequest validates a chat completion request
func ValidateChatRequest(req *ChatRequest) error {
	if req.Model == "" {
		return ValidationError{Field: "model", Message: "model is required"}
	}

	if len(req.Messages) == 0 {
		return ValidationError{Field: "messages", Message: "at least one message is required"}
	}

	// Validate messages
	for i, msg := range req.Messages {
		if err := validateMessage(&msg, i); err != nil {
			return err
		}
	}

	// Validate thinking config
	if req.Thinking != nil {
		if req.Thinking.Type != "enabled" && req.Thinking.Type != "disabled" {
			return ValidationError{Field: "thinking.type", Message: "must be 'enabled' or 'disabled'"}
		}
	}

	// Validate reasoning_effort
	if req.ReasoningEffort != "" {
		if req.ReasoningEffort != "high" && req.ReasoningEffort != "max" {
			return ValidationError{Field: "reasoning_effort", Message: "must be 'high' or 'max'"}
		}
	}

	// Validate temperature
	if req.Temperature != nil {
		if *req.Temperature < 0 || *req.Temperature > 2 {
			return ValidationError{Field: "temperature", Message: "must be between 0 and 2"}
		}
	}

	// Validate top_p
	if req.TopP != nil {
		if *req.TopP < 0 || *req.TopP > 1 {
			return ValidationError{Field: "top_p", Message: "must be between 0 and 1"}
		}
	}

	// Validate frequency_penalty
	if req.FrequencyPenalty != nil {
		if *req.FrequencyPenalty < -2 || *req.FrequencyPenalty > 2 {
			return ValidationError{Field: "frequency_penalty", Message: "must be between -2 and 2"}
		}
	}

	// Validate presence_penalty
	if req.PresencePenalty != nil {
		if *req.PresencePenalty < -2 || *req.PresencePenalty > 2 {
			return ValidationError{Field: "presence_penalty", Message: "must be between -2 and 2"}
		}
	}

	// Validate response_format
	if req.ResponseFormat != nil {
		if req.ResponseFormat.Type != "text" && req.ResponseFormat.Type != "json_object" {
			return ValidationError{Field: "response_format.type", Message: "must be 'text' or 'json_object'"}
		}
	}

	// Validate stop (string or array of up to 16 strings)
	if req.Stop != nil {
		if err := validateStop(req.Stop); err != nil {
			return ValidationError{Field: "stop", Message: err.Error()}
		}
	}

	// Validate top_logprobs
	if req.TopLogprobs != nil {
		if *req.TopLogprobs < 0 || *req.TopLogprobs > 20 {
			return ValidationError{Field: "top_logprobs", Message: "must be between 0 and 20"}
		}
	}

	// Validate tools
	for i, tool := range req.Tools {
		if tool.Type != "function" {
			return ValidationError{Field: fmt.Sprintf("tools[%d].type", i), Message: "must be 'function'"}
		}
		if tool.Function.Name == "" {
			return ValidationError{Field: fmt.Sprintf("tools[%d].function.name", i), Message: "is required"}
		}
	}

	// Validate tool_choice
	if req.ToolChoice != nil {
		if err := validateToolChoice(req.ToolChoice); err != nil {
			return ValidationError{Field: "tool_choice", Message: err.Error()}
		}
	}

	return nil
}

func validateMessage(msg *Message, index int) error {
	validRoles := map[string]bool{
		"system":    true,
		"user":      true,
		"assistant": true,
		"tool":      true,
	}

	if !validRoles[msg.Role] {
		return ValidationError{
			Field:   fmt.Sprintf("messages[%d].role", index),
			Message: fmt.Sprintf("invalid role '%s', must be one of: system, user, assistant, tool", msg.Role),
		}
	}

	// Validate content
	if msg.Content != nil {
		if contentStr, ok := msg.Content.(string); ok {
			if contentStr == "" && msg.Role != "assistant" {
				return ValidationError{
					Field:   fmt.Sprintf("messages[%d].content", index),
					Message: "cannot be empty",
				}
			}
		}
	} else if msg.Role != "assistant" {
		return ValidationError{
			Field:   fmt.Sprintf("messages[%d].content", index),
			Message: "is required",
		}
	}

	// Tool messages require tool_call_id
	if msg.Role == "tool" && (msg.ToolCallID == nil || *msg.ToolCallID == "") {
		return ValidationError{
			Field:   fmt.Sprintf("messages[%d].tool_call_id", index),
			Message: "is required for tool messages",
		}
	}

	// Assistant messages with prefix must have prefix=true
	if msg.Prefix != nil && *msg.Prefix && msg.Role != "assistant" {
		return ValidationError{
			Field:   fmt.Sprintf("messages[%d].prefix", index),
			Message: "can only be set on assistant messages",
		}
	}

	return nil
}

func validateStop(stop interface{}) error {
	switch s := stop.(type) {
	case string:
		if s == "" {
			return fmt.Errorf("stop string cannot be empty")
		}
	case []string:
		if len(s) == 0 {
			return fmt.Errorf("stop array cannot be empty")
		}
		if len(s) > 16 {
			return fmt.Errorf("stop array can have at most 16 strings")
		}
		for i, str := range s {
			if str == "" {
				return fmt.Errorf("stop array element %d cannot be empty", i)
			}
		}
	default:
		return fmt.Errorf("must be a string or array of strings")
	}
	return nil
}

func validateToolChoice(toolChoice interface{}) error {
	switch tc := toolChoice.(type) {
	case string:
		validChoices := map[string]bool{
			"none":     true,
			"auto":     true,
			"required": true,
		}
		if !validChoices[tc] {
			return fmt.Errorf("must be 'none', 'auto', or 'required'")
		}
	case map[string]interface{}:
		// ToolChoiceFunction format
		if tc["type"] != "function" {
			return fmt.Errorf("type must be 'function'")
		}
		funcMap, ok := tc["function"].(map[string]interface{})
		if !ok {
			return fmt.Errorf("function must be an object")
		}
		if funcMap["name"] == nil || funcMap["name"] == "" {
			return fmt.Errorf("function.name is required")
		}
	default:
		return fmt.Errorf("must be a string or object")
	}
	return nil
}

// ValidateFIMRequest validates a FIM completion request
func ValidateFIMRequest(req *FIMRequest) error {
	if req.Model == "" {
		return ValidationError{Field: "model", Message: "model is required"}
	}

	if req.Prompt == "" {
		return ValidationError{Field: "prompt", Message: "prompt is required"}
	}

	// Validate max_tokens (max 4K for FIM)
	if req.MaxTokens != nil && *req.MaxTokens > 4096 {
		return ValidationError{Field: "max_tokens", Message: "must be at most 4096 for FIM"}
	}

	// Validate temperature
	if req.Temperature != nil {
		if *req.Temperature < 0 || *req.Temperature > 2 {
			return ValidationError{Field: "temperature", Message: "must be between 0 and 2"}
		}
	}

	// Validate top_p
	if req.TopP != nil {
		if *req.TopP < 0 || *req.TopP > 1 {
			return ValidationError{Field: "top_p", Message: "must be between 0 and 1"}
		}
	}

	// Validate frequency_penalty
	if req.FrequencyPenalty != nil {
		if *req.FrequencyPenalty < -2 || *req.FrequencyPenalty > 2 {
			return ValidationError{Field: "frequency_penalty", Message: "must be between -2 and 2"}
		}
	}

	// Validate presence_penalty
	if req.PresencePenalty != nil {
		if *req.PresencePenalty < -2 || *req.PresencePenalty > 2 {
			return ValidationError{Field: "presence_penalty", Message: "must be between -2 and 2"}
		}
	}

	// Validate stop
	if req.Stop != nil {
		if err := validateStop(req.Stop); err != nil {
			return ValidationError{Field: "stop", Message: err.Error()}
		}
	}

	// Validate logprobs
	if req.Logprobs != nil {
		if *req.Logprobs < 0 || *req.Logprobs > 20 {
			return ValidationError{Field: "logprobs", Message: "must be between 0 and 20"}
		}
	}

	return nil
}
