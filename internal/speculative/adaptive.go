package speculative

import (
	"fmt"
)

// AdaptiveSpeculativeDecoder extends SpeculativeDecoder with adaptive behavior
// based on tool call failures and difficulty estimation
type AdaptiveSpeculativeDecoder struct {
	*SpeculativeDecoder
	FailureCount     int
	DifficultyLevel  int
	FlashModel       string
	ProModel         string
	MaxRetries       int
}

// NewAdaptiveSpeculativeDecoder creates a new adaptive speculative decoder
func NewAdaptiveSpeculativeDecoder(client APIClientIface, flashModel, proModel string, maxDraftTokens int) *AdaptiveSpeculativeDecoder {
	return &AdaptiveSpeculativeDecoder{
		SpeculativeDecoder: NewSpeculativeDecoder(client, flashModel, proModel, maxDraftTokens),
		FlashModel:         flashModel,
		ProModel:           proModel,
		MaxRetries:         3,
		DifficultyLevel:    0,
	}
}

// RecordFailure increments the failure count and adjusts difficulty
func (a *AdaptiveSpeculativeDecoder) RecordFailure() {
	a.FailureCount++
	// Increase difficulty level based on failure count
	if a.FailureCount > 5 {
		a.DifficultyLevel = 3 // High difficulty
	} else if a.FailureCount > 2 {
		a.DifficultyLevel = 2 // Medium difficulty
	} else {
		a.DifficultyLevel = 1 // Low difficulty
	}
}

// Reset resets the failure counter and difficulty
func (a *AdaptiveSpeculativeDecoder) Reset() {
	a.FailureCount = 0
	a.DifficultyLevel = 0
}

// GetDifficultyLevel returns the current difficulty level (0-3)
func (a *AdaptiveSpeculativeDecoder) GetDifficultyLevel() int {
	return a.DifficultyLevel
}

// ShouldUsePro determines if we should use Pro model based on difficulty
func (a *AdaptiveSpeculativeDecoder) ShouldUsePro() bool {
	// Use Pro model for high difficulty or after multiple failures
	return a.DifficultyLevel >= 2 || a.FailureCount >= 3
}

// AdaptiveDecode performs adaptive speculative decoding
// Uses Flash for draft, but escalates to Pro based on difficulty
func (a *AdaptiveSpeculativeDecoder) AdaptiveDecode(prompt string) (string, error) {
	if a.ShouldUsePro() {
		// For high difficulty, skip speculation and use Pro directly
		return a.decodeWithPro(prompt)
	}
	
	// Normal speculative decoding flow
	return a.Decode(prompt)
}

// decodeWithPro bypasses speculation and uses Pro model directly
func (a *AdaptiveSpeculativeDecoder) decodeWithPro(prompt string) (string, error) {
	req := map[string]interface{}{
		"model": a.ProModel,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	}
	
	resp, err := a.Client.ChatCompletion(req)
	if err != nil {
		return "", fmt.Errorf("pro model error: %w", err)
	}
	
	chatResp, ok := resp.(*ChatResponse)
	if !ok || len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("invalid pro response")
	}
	
	return chatResp.Choices[0].Message.Content, nil
}

// SpawnVariadicCalls spawns additional parallel calls to Flash for Best-N style evaluation
// Returns multiple candidate responses for difficult problems
func (a *AdaptiveSpeculativeDecoder) SpawnVariadicCalls(prompt string, n int) ([]string, error) {
	candidates := make([]string, 0, n)
	
	for i := 0; i < n; i++ {
		req := map[string]interface{}{
			"model": a.FlashModel,
			"messages": []map[string]string{
				{"role": "user", "content": prompt},
			},
			"temperature": 0.7 + float64(i)*0.1, // Vary temperature for diversity
		}
		
		resp, err := a.Client.ChatCompletion(req)
		if err != nil {
			// Continue with remaining calls even if one fails
			continue
		}
		
		chatResp, ok := resp.(*ChatResponse)
		if !ok || len(chatResp.Choices) == 0 {
			continue
		}
		
		candidates = append(candidates, chatResp.Choices[0].Message.Content)
	}
	
	if len(candidates) == 0 {
		return nil, fmt.Errorf("all variadic calls failed")
	}
	
	return candidates, nil
}

// EvaluateAndSelect evaluates candidates using Pro as judge and selects best
func (a *AdaptiveSpeculativeDecoder) EvaluateAndSelect(candidates []string, evalPrompt string) (string, error) {
	if len(candidates) == 0 {
		return "", fmt.Errorf("no candidates to evaluate")
	}
	
	if len(candidates) == 1 {
		return candidates[0], nil
	}
	
	// Build evaluation prompt
	evalMsg := fmt.Sprintf("%s\n\nCandidates:\n", evalPrompt)
	for i, cand := range candidates {
		evalMsg += fmt.Sprintf("[%d] %s\n\n", i, cand)
	}
	evalMsg += "Respond with only the index number of the best candidate (0-based)."
	
	req := map[string]interface{}{
		"model": a.ProModel,
		"messages": []map[string]string{
			{"role": "user", "content": evalMsg},
		},
		"temperature": 0.0, // Deterministic evaluation
	}
	
	resp, err := a.Client.ChatCompletion(req)
	if err != nil {
		return "", fmt.Errorf("evaluation error: %w", err)
	}
	
	chatResp, ok := resp.(*ChatResponse)
	if !ok || len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("invalid evaluation response")
	}
	
	// Parse the winner index from response
	// In production, use structured output or regex
	content := chatResp.Choices[0].Message.Content
	// Simple parsing - extract first digit
	for _, c := range content {
		if c >= '0' && c <= '9' {
			idx := int(c - '0')
			if idx >= 0 && idx < len(candidates) {
				return candidates[idx], nil
			}
		}
	}
	
	// Default to first candidate if parsing fails
	return candidates[0], nil
}
