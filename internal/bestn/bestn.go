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

// BestN implements the Best N evaluation logic
type BestN struct {
	Evaluator EvaluatorIface
	N         int
}

// NewBestN creates a new BestN instance
func NewBestN(evaluator EvaluatorIface, n int) *BestN {
	return &BestN{
		Evaluator: evaluator,
		N:         n,
	}
}

// GenerateCandidates generates N candidate responses
func (b *BestN) GenerateCandidates(prompt string) ([]string, error) {
	// Implementation will use draft model or multiple API calls
	return nil, fmt.Errorf("not implemented")
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
