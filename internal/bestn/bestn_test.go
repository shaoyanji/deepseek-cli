package bestn

import (
	"errors"
	"testing"
)

// MockEvaluator mocks the evaluator for testing
type MockEvaluator struct {
	Result *EvalResult
	Err    error
}

func (m *MockEvaluator) Evaluate(candidates []string, evalPrompt string) (*EvalResult, error) {
	return m.Result, m.Err
}

// MockAPIClientWithCallCount wraps MockAPIClient to track calls for testing
type MockAPIClientWithCallCount struct {
	Responses []map[string]interface{}
	Err       error
	CallCount int
}

func (m *MockAPIClientWithCallCount) ChatCompletion(req interface{}) (interface{}, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	if m.CallCount >= len(m.Responses) {
		return nil, errors.New("no more mock responses")
	}
	resp := m.Responses[m.CallCount]
	m.CallCount++
	return resp, nil
}

func TestNewBestN(t *testing.T) {
	evaluator := &MockEvaluator{}
	apiClient := &MockAPIClientWithCallCount{}
	bestn := NewBestN(evaluator, apiClient, 5)

	if bestn == nil {
		t.Fatal("NewBestN() returned nil")
	}
	if bestn.N != 5 {
		t.Errorf("NewBestN() N = %d, want 5", bestn.N)
	}
	if bestn.Evaluator != evaluator {
		t.Error("NewBestN() Evaluator not set correctly")
	}
	if bestn.APIClient != apiClient {
		t.Error("NewBestN() APIClient not set correctly")
	}
}

func TestGenerateCandidates(t *testing.T) {
	tests := []struct {
		name      string
		n         int
		apiClient APIClientIface
		wantErr   bool
		wantCount int
	}{
		{
			name: "generate 3 candidates",
			n:    3,
			apiClient: &MockAPIClientWithCallCount{
				Responses: []map[string]interface{}{
					{"choices": []interface{}{map[string]interface{}{"message": map[string]interface{}{"content": "response1"}}}},
					{"choices": []interface{}{map[string]interface{}{"message": map[string]interface{}{"content": "response2"}}}},
					{"choices": []interface{}{map[string]interface{}{"message": map[string]interface{}{"content": "response3"}}}},
				},
			},
			wantErr:   false,
			wantCount: 3,
		},
		{
			name: "API error",
			n:    2,
			apiClient: &MockAPIClientWithCallCount{
				Err: errors.New("API error"),
			},
			wantErr:   true,
			wantCount: 0,
		},
		{
			name:      "nil API client",
			n:         2,
			apiClient: nil,
			wantErr:   true,
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluator := &MockEvaluator{}
			bestn := NewBestN(evaluator, tt.apiClient, tt.n)

			candidates, err := bestn.GenerateCandidates("test prompt")

			if tt.wantErr && err == nil {
				t.Error("GenerateCandidates() should have returned error")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("GenerateCandidates() unexpected error: %v", err)
			}
			if len(candidates) != tt.wantCount {
				t.Errorf("GenerateCandidates() returned %d candidates, want %d", len(candidates), tt.wantCount)
			}
		})
	}
}

func TestEvaluateCandidates(t *testing.T) {
	evaluator := &MockEvaluator{
		Result: &EvalResult{
			WinnerID:        0,
			Recommendations: []string{"First candidate is best"},
			Merged:         "merged result",
		},
	}
	apiClient := &MockAPIClient{}
	bestn := NewBestN(evaluator, apiClient, 3)

	candidates := []string{"cand1", "cand2", "cand3"}
	result, err := bestn.EvaluateCandidates(candidates, "which is best?")

	if err != nil {
		t.Errorf("EvaluateCandidates() unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("EvaluateCandidates() returned nil result")
	}
	if result.WinnerID != 0 {
		t.Errorf("EvaluateCandidates() WinnerID = %d, want 0", result.WinnerID)
	}
}

func TestSelectWinner(t *testing.T) {
	evaluator := &MockEvaluator{
		Result: &EvalResult{WinnerID: 1},
	}
	apiClient := &MockAPIClient{}
	bestn := NewBestN(evaluator, apiClient, 3)

	candidates := []string{"bad", "good", "ok"}
	winner, err := bestn.SelectWinner(candidates, "select best")

	if err != nil {
		t.Errorf("SelectWinner() unexpected error: %v", err)
	}
	if winner != "good" {
		t.Errorf("SelectWinner() = %q, want %q", winner, "good")
	}
}

func TestEvalResultValidation(t *testing.T) {
	// Valid result
	result := &EvalResult{
		WinnerID:        0,
		Recommendations: []string{"rec1"},
		Merged:         "merged",
	}
	if !result.IsValid(3) {
		t.Error("IsValid() should return true for valid result")
	}

	// Invalid: WinnerID out of range
	result.WinnerID = 5
	if result.IsValid(3) {
		t.Error("IsValid() should return false for WinnerID out of range")
	}

	// Invalid: negative WinnerID
	result.WinnerID = -1
	if result.IsValid(3) {
		t.Error("IsValid() should return false for negative WinnerID")
	}
}

func TestEvaluatorSchema(t *testing.T) {
	schema := GetEvaluatorSchema()

	if schema == nil {
		t.Error("GetEvaluatorSchema() returned nil")
	}
	if schema["type"] != "object" {
		t.Errorf("GetEvaluatorSchema() type = %v, want 'object'", schema["type"])
	}

	properties, ok := schema["properties"].(map[string]interface{})
	if !ok {
		t.Error("GetEvaluatorSchema() properties not found")
	}

	if _, ok := properties["winner"]; !ok {
		t.Error("GetEvaluatorSchema() missing 'winner' property")
	}
	if _, ok := properties["recommendations"]; !ok {
		t.Error("GetEvaluatorSchema() missing 'recommendations' property")
	}
	if _, ok := properties["merged"]; !ok {
		t.Error("GetEvaluatorSchema() missing 'merged' property")
	}

	required, ok := schema["required"].([]string)
	if !ok {
		t.Error("GetEvaluatorSchema() required not found")
	}

	found := false
	for _, r := range required {
		if r == "winner" {
			found = true
			break
		}
	}
	if !found {
		t.Error("GetEvaluatorSchema() missing 'winner' in required")
	}
}

func TestEmptyCandidates(t *testing.T) {
	evaluator := &MockEvaluator{}
	apiClient := &MockAPIClient{}
	bestn := NewBestN(evaluator, apiClient, 3)

	winner, err := bestn.SelectWinner([]string{}, "prompt")
	if err == nil {
		t.Error("SelectWinner() should error with empty candidates")
	}
	if winner != "" {
		t.Errorf("SelectWinner() with empty candidates = %q, want empty", winner)
	}
}

func TestSingleCandidate(t *testing.T) {
	evaluator := &MockEvaluator{
		Result: &EvalResult{WinnerID: 0},
	}
	apiClient := &MockAPIClient{}
	bestn := NewBestN(evaluator, apiClient, 1)

	candidates := []string{"only one"}
	winner, err := bestn.SelectWinner(candidates, "prompt")

	if err != nil {
		t.Errorf("SelectWinner() unexpected error: %v", err)
	}
	if winner != "only one" {
		t.Errorf("SelectWinner() = %q, want 'only one'", winner)
	}
}

func TestMergeCandidates(t *testing.T) {
	evaluator := &MockEvaluator{
		Result: &EvalResult{
			WinnerID: 0,
			Merged:   "merged code",
		},
	}
	apiClient := &MockAPIClient{}
	bestn := NewBestN(evaluator, apiClient, 2)

	candidates := []string{"code1", "code2"}
	merged, err := bestn.MergeCandidates(candidates, "merge these")

	if err != nil {
		t.Errorf("MergeCandidates() unexpected error: %v", err)
	}
	if merged != "merged code" {
		t.Errorf("MergeCandidates() = %q, want 'merged code'", merged)
	}
}

func TestBestNWithGitDiff(t *testing.T) {
	evaluator := &MockEvaluator{
		Result: &EvalResult{
			WinnerID:        1,
			Recommendations: []string{"Second diff is cleaner"},
		},
	}
	apiClient := &MockAPIClient{}
	bestn := NewBestN(evaluator, apiClient, 3)

	diffs := []string{
		"diff --git a/file1.go...",
		"diff --git a/file2.go...",
		"diff --git a/file3.go...",
	}
	result, err := bestn.EvaluateCandidates(diffs, "which diff is best?")

	if err != nil {
		t.Errorf("EvaluateCandidates() unexpected error: %v", err)
	}
	if result.WinnerID != 1 {
		t.Errorf("EvaluateCandidates() WinnerID = %d, want 1", result.WinnerID)
	}
}
