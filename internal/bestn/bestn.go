package bestn

import "fmt"

// EvalResult represents the evaluation result from DeepSeek
type EvalResult struct {
	WinnerID        int      `json:"winner"`
	Recommendations []string `json:"recommendations,omitempty"`
	Merged          string   `json:"merged,omitempty"`
}

// EvaluatorIface represents the evaluator interface
type EvaluatorIface interface {
	Evaluate(candidates []string, evalPrompt string) (*EvalResult, error)
}

// APIClientIface represents the API client interface for generating candidates
type APIClientIface interface {
	ChatCompletion(req interface{}) (interface{}, error)
}

// BestN implements the Best N evaluation logic
type BestN struct {
	Evaluator EvaluatorIface
	APIClient APIClientIface
	N         int
}

// NewBestN creates a new BestN instance
func NewBestN(evaluator EvaluatorIface, apiClient APIClientIface, n int) *BestN {
	return &BestN{
		Evaluator: evaluator,
		APIClient: apiClient,
		N:         n,
	}
}

// GenerateCandidates generates N candidate responses
func (b *BestN) GenerateCandidates(prompt string) ([]string, error) {
	// Check if APIClient is nil or holds a nil pointer
	if b.APIClient == nil {
		return nil, fmt.Errorf("no API client configured")
	}

	// Type assert to check if the underlying value is nil
	if mc, ok := b.APIClient.(*MockAPIClient); ok && mc == nil {
		return nil, fmt.Errorf("no API client configured")
	}

	if b.N <= 0 {
		return nil, fmt.Errorf("N must be positive")
	}

	candidates := make([]string, 0, b.N)
	for i := 0; i < b.N; i++ {
		req := map[string]interface{}{
			"model": "deepseek-v4-pro",
			"messages": []map[string]string{
				{"role": "user", "content": prompt},
			},
		}
		resp, err := b.APIClient.ChatCompletion(req)
		if err != nil {
			return nil, fmt.Errorf("API call %d failed: %w", i, err)
		}

		// Extract content from response
		respMap, ok := resp.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid response format")
		}
		choices, ok := respMap["choices"].([]interface{})
		if !ok || len(choices) == 0 {
			return nil, fmt.Errorf("no choices in response")
		}
		choice, ok := choices[0].(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid choice format")
		}
		message, ok := choice["message"].(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("no message in choice")
		}
		content, ok := message["content"].(string)
		if !ok {
			return nil, fmt.Errorf("no content in message")
		}
		candidates = append(candidates, content)
	}
	return candidates, nil
}

// EvaluateCandidates evaluates candidates and returns the full result
func (b *BestN) EvaluateCandidates(candidates []string, evalPrompt string) (*EvalResult, error) {
	if len(candidates) == 0 {
		return nil, fmt.Errorf("no candidates to evaluate")
	}
	
	if b.Evaluator == nil {
		return nil, fmt.Errorf("no evaluator configured")
	}
	
	return b.Evaluator.Evaluate(candidates, evalPrompt)
}

// SelectWinner evaluates candidates and returns the winning candidate
func (b *BestN) SelectWinner(candidates []string, evalPrompt string) (string, error) {
	result, err := b.EvaluateCandidates(candidates, evalPrompt)
	if err != nil {
		return "", err
	}
	
	if result.WinnerID < 0 || result.WinnerID >= len(candidates) {
		return "", fmt.Errorf("invalid winner ID: %d", result.WinnerID)
	}
	
	return candidates[result.WinnerID], nil
}

// MergeCandidates evaluates and returns merged result
func (b *BestN) MergeCandidates(candidates []string, evalPrompt string) (string, error) {
	result, err := b.EvaluateCandidates(candidates, evalPrompt)
	if err != nil {
		return "", err
	}
	
	return result.Merged, nil
}

// IsValid checks if an EvalResult is valid
func (r *EvalResult) IsValid(numCandidates int) bool {
	if r.WinnerID < 0 || r.WinnerID >= numCandidates {
		return false
	}
	return true
}

// GetEvaluatorSchema returns the JSON schema for evaluator responses
func GetEvaluatorSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"winner": map[string]interface{}{
				"type": "integer",
			},
			"recommendations": map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "string",
				},
			},
			"merged": map[string]interface{}{
				"type": "string",
			},
		},
		"required": []string{"winner"},
	}
}
