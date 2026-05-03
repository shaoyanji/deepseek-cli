package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

// formatChatResponse formats a chat completion response for display
func formatChatResponse(data []byte, showCache bool) error {
	var resp ChatResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		// If parsing fails, just output raw JSON
		fmt.Println(string(data))
		return nil
	}

	// Output main content
	for _, choice := range resp.Choices {
		if choice.Message.Content != nil {
			if contentStr, ok := choice.Message.Content.(string); ok && contentStr != "" {
				fmt.Println(contentStr)
			}
		}
		
		// Handle reasoning_content (thinking mode)
		if choice.Message.ReasoningContent != nil && *choice.Message.ReasoningContent != "" {
			fmt.Printf("\n[Reasoning: %s]\n", *choice.Message.ReasoningContent)
		}
		
		// Handle tool_calls
		if choice.Message.ToolCallID == nil || *choice.Message.ToolCallID == "" {
			// Try to extract tool calls if they exist
			if toolCallsData, err := json.Marshal(choice.Message); err == nil {
				var toolMsg struct {
					ToolCalls []struct {
						ID       string `json:"id"`
						Type     string `json:"type"`
						Function struct {
							Name      string `json:"name"`
							Arguments string `json:"arguments"`
						} `json:"function"`
					} `json:"tool_calls"`
				}
				if err := json.Unmarshal(toolCallsData, &toolMsg); err == nil && len(toolMsg.ToolCalls) > 0 {
					fmt.Println("\n[Tool Calls]")
					for _, tc := range toolMsg.ToolCalls {
						fmt.Printf("  - %s(%s)\n", tc.Function.Name, tc.Function.Arguments)
					}
				}
			}
		}
		
		if choice.FinishReason != nil {
			fmt.Printf("\n[finish_reason: %s]\n", *choice.FinishReason)
		}
	}

	// Output usage statistics
	if resp.Usage != nil {
		if showCache {
			fmt.Printf("\n[usage: prompt_tokens=%d, completion_tokens=%d, total_tokens=%d, cache_hit_tokens=%d, cache_miss_tokens=%d]\n",
				resp.Usage.PromptTokens,
				resp.Usage.CompletionTokens,
				resp.Usage.TotalTokens,
				resp.Usage.PromptCacheHitTokens,
				resp.Usage.PromptCacheMissTokens)
		} else {
			fmt.Printf("\n[usage: prompt_tokens=%d, completion_tokens=%d, total_tokens=%d]\n",
				resp.Usage.PromptTokens,
				resp.Usage.CompletionTokens,
				resp.Usage.TotalTokens)
		}
	}

	return nil
}

// formatFIMResponse formats a FIM completion response for display
func formatFIMResponse(data []byte) error {
	var resp FIMResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		// If parsing fails, just output raw JSON
		fmt.Println(string(data))
		return nil
	}

	// Output main content
	for _, choice := range resp.Choices {
		if choice.Text != "" {
			fmt.Println(choice.Text)
		}
		if choice.FinishReason != nil {
			fmt.Printf("\n[finish_reason: %s]\n", *choice.FinishReason)
		}
	}

	// Output usage statistics
	if resp.Usage != nil {
		fmt.Printf("\n[usage: prompt_tokens=%d, completion_tokens=%d, total_tokens=%d]\n",
			resp.Usage.PromptTokens,
			resp.Usage.CompletionTokens,
			resp.Usage.TotalTokens)
	}

	return nil
}

// formatJSONModeResponse formats a JSON mode response
func formatJSONModeResponse(data []byte, showCache bool) error {
	var resp ChatResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		// If parsing fails, try to extract just the content
		var raw map[string]interface{}
		if err := json.Unmarshal(data, &raw); err != nil {
			fmt.Println(string(data))
			return nil
		}
		
		// Try to get choices[0].message.content
		if choices, ok := raw["choices"].([]interface{}); ok && len(choices) > 0 {
			if choice, ok := choices[0].(map[string]interface{}); ok {
				if message, ok := choice["message"].(map[string]interface{}); ok {
					if content, ok := message["content"].(string); ok {
						// Pretty print the JSON content
						var prettyJSON interface{}
						if err := json.Unmarshal([]byte(content), &prettyJSON); err == nil {
							formatted, _ := json.MarshalIndent(prettyJSON, "", "  ")
							fmt.Println(string(formatted))
							return nil
						}
						fmt.Println(content)
						return nil
					}
				}
			}
		}
		fmt.Println(string(data))
		return nil
	}

	// Output formatted JSON content
	for _, choice := range resp.Choices {
		if choice.Message.Content != nil {
			if contentStr, ok := choice.Message.Content.(string); ok && contentStr != "" {
				// Try to parse and pretty-print the JSON content
				var prettyJSON interface{}
				if err := json.Unmarshal([]byte(contentStr), &prettyJSON); err == nil {
					formatted, _ := json.MarshalIndent(prettyJSON, "", "  ")
					fmt.Println(string(formatted))
				} else {
					fmt.Println(contentStr)
				}
			}
		}
	}

	// Output usage statistics
	if resp.Usage != nil {
		if showCache {
			fmt.Printf("\n[usage: prompt_tokens=%d, completion_tokens=%d, total_tokens=%d, cache_hit_tokens=%d, cache_miss_tokens=%d]\n",
				resp.Usage.PromptTokens,
				resp.Usage.CompletionTokens,
				resp.Usage.TotalTokens,
				resp.Usage.PromptCacheHitTokens,
				resp.Usage.PromptCacheMissTokens)
		} else {
			fmt.Printf("\n[usage: prompt_tokens=%d, completion_tokens=%d, total_tokens=%d]\n",
				resp.Usage.PromptTokens,
				resp.Usage.CompletionTokens,
				resp.Usage.TotalTokens)
		}
	}

	return nil
}

// extractContent extracts the main content from a response
func extractContent(data []byte, completionType string) string {
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return string(data)
	}

	switch completionType {
	case "chat":
		if choices, ok := raw["choices"].([]interface{}); ok && len(choices) > 0 {
			if choice, ok := choices[0].(map[string]interface{}); ok {
				if message, ok := choice["message"].(map[string]interface{}); ok {
					if content, ok := message["content"].(string); ok {
						return content
					}
				}
			}
		}
	case "fim":
		if choices, ok := raw["choices"].([]interface{}); ok && len(choices) > 0 {
			if choice, ok := choices[0].(map[string]interface{}); ok {
				if text, ok := choice["text"].(string); ok {
					return text
				}
			}
		}
	}

	return string(data)
}

// extractUsage extracts usage statistics from a response
func extractUsage(data []byte) *Usage {
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil
	}

	if usageData, ok := raw["usage"].(map[string]interface{}); ok {
		usage := &Usage{}
		if promptTokens, ok := usageData["prompt_tokens"].(float64); ok {
			usage.PromptTokens = int(promptTokens)
		}
		if completionTokens, ok := usageData["completion_tokens"].(float64); ok {
			usage.CompletionTokens = int(completionTokens)
		}
		if totalTokens, ok := usageData["total_tokens"].(float64); ok {
			usage.TotalTokens = int(totalTokens)
		}
		return usage
	}

	return nil
}

// extractFinishReason extracts the finish reason from a response
func extractFinishReason(data []byte) string {
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return ""
	}

	if choices, ok := raw["choices"].([]interface{}); ok && len(choices) > 0 {
		if choice, ok := choices[0].(map[string]interface{}); ok {
			if finishReason, ok := choice["finish_reason"].(string); ok {
				return finishReason
			}
		}
	}

	return ""
}

// extractReasoningContent extracts reasoning content from a thinking mode response
func extractReasoningContent(data []byte) string {
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return ""
	}

	if choices, ok := raw["choices"].([]interface{}); ok && len(choices) > 0 {
		if choice, ok := choices[0].(map[string]interface{}); ok {
			if message, ok := choice["message"].(map[string]interface{}); ok {
				if reasoningContent, ok := message["reasoning_content"].(string); ok {
					return reasoningContent
				}
			}
		}
	}

	return ""
}

// extractToolCalls extracts tool calls from a response
func extractToolCalls(data []byte) []string {
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil
	}

	if choices, ok := raw["choices"].([]interface{}); ok && len(choices) > 0 {
		if choice, ok := choices[0].(map[string]interface{}); ok {
			if message, ok := choice["message"].(map[string]interface{}); ok {
				if toolCalls, ok := message["tool_calls"].([]interface{}); ok {
					var calls []string
					for _, tc := range toolCalls {
						if tcMap, ok := tc.(map[string]interface{}); ok {
							if function, ok := tcMap["function"].(map[string]interface{}); ok {
								name, _ := function["name"].(string)
								args, _ := function["arguments"].(string)
								calls = append(calls, fmt.Sprintf("%s(%s)", name, args))
							}
						}
					}
					return calls
				}
			}
		}
	}

	return nil
}

// isJSONModeResponse checks if the response is from JSON mode
func isJSONModeResponse(data []byte) bool {
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return false
	}

	if responseFormat, ok := raw["response_format"].(map[string]interface{}); ok {
		if formatType, ok := responseFormat["type"].(string); ok {
			return formatType == "json_object"
		}
	}

	return false
}

// formatErrorResponse formats an error response for display
func formatErrorResponse(data []byte) error {
	var errResp struct {
		Error struct {
			Message string `json:"message"`
			Type    string `json:"type"`
			Code    string `json:"code"`
		} `json:"error"`
	}
	
	if err := json.Unmarshal(data, &errResp); err == nil && errResp.Error.Message != "" {
		return fmt.Errorf("API error: %s (type: %s, code: %s)", errResp.Error.Message, errResp.Error.Type, errResp.Error.Code)
	}
	
	return fmt.Errorf("API error: %s", string(data))
}

// shouldFormatPretty determines if we should use pretty formatting
func shouldFormatPretty() bool {
	// This could be controlled by a flag in the future
	// For now, we'll default to pretty formatting for non-JSON mode
	return true
}

// trimWhitespace trims leading/trailing whitespace from content
func trimWhitespace(content string) string {
	return strings.TrimSpace(content)
}

// formatModelsResponse formats a models response for display
func formatModelsResponse(data []byte) error {
	var resp ModelsResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		// If parsing fails, just output raw JSON
		fmt.Println(string(data))
		return nil
	}

	fmt.Printf("Object: %s\n", resp.Object)
	fmt.Printf("Total models: %d\n\n", len(resp.Data))
	
	for _, model := range resp.Data {
		fmt.Printf("  ID: %s\n", model.ID)
		fmt.Printf("  Type: %s\n", model.Object)
		fmt.Printf("  Owned by: %s\n", model.OwnedBy)
		if model.Created > 0 {
			fmt.Printf("  Created: %d\n", model.Created)
		}
		fmt.Println()
	}

	return nil
}

// formatBalanceResponse formats a balance response for display
func formatBalanceResponse(data []byte) error {
	var resp BalanceResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		// If parsing fails, just output raw JSON
		fmt.Println(string(data))
		return nil
	}

	fmt.Printf("Balance Information:\n")
	fmt.Printf("  Balance: %.6f\n", resp.Balance)
	fmt.Printf("  Total Balance: %.6f\n", resp.TotalBalance)
	fmt.Printf("  Available Balance: %.6f\n", resp.AvailableBalance)
	fmt.Printf("  Granted Balance: %.6f\n", resp.GrantedBalance)

	return nil
}
