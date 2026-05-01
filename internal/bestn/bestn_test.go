package bestn

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockEvaluator mocks the evaluator for testing
type MockEvaluator struct {
	mock.Mock
}

func (m *MockEvaluator) Evaluate(candidates []string, evalPrompt string) (*EvalResult, error) {
	args := m.Called(candidates, evalPrompt)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*EvalResult), args.Error(1)
}

func TestNewBestN(t *testing.T) {
	evaluator := new(MockEvaluator)
	bestn := NewBestN(evaluator, 5)
	
	assert.NotNil(t, bestn)
	assert.Equal(t, 5, bestn.N)
	assert.Equal(t, evaluator, bestn.Evaluator)
}

func TestGenerateCandidates(t *testing.T) {
	evaluator := new(MockEvaluator)
	bestn := NewBestN(evaluator, 3)
	
	// Implementation will generate candidates via API
	result, err := bestn.GenerateCandidates("write a function to add two numbers")
	assert.Error(t, err) // Not implemented yet
	assert.Nil(t, result)
}

func TestEvaluateCandidates(t *testing.T) {
	evaluator := new(MockEvaluator)
	evaluator.On("Evaluate", mock.Anything, mock.Anything).Return(&EvalResult{
		WinnerID:        0,
		Recommendations: []string{"First candidate is best"},
		Merged:         "merged result",
	}, nil)
	
	bestn := NewBestN(evaluator, 3)
	candidates := []string{"cand1", "cand2", "cand3"}
	
	result, err := bestn.EvaluateCandidates(candidates, "which is best?")
	assert.NoError(t, err)
	assert.Equal(t, 0, result.WinnerID)
	assert.Len(t, result.Recommendations, 1)
	assert.Equal(t, "merged result", result.Merged)
	evaluator.AssertExpectations(t)
}

func TestSelectWinner(t *testing.T) {
	evaluator := new(MockEvaluator)
	evaluator.On("Evaluate", mock.Anything, mock.Anything).Return(&EvalResult{
		WinnerID: 1,
	}, nil)
	
	bestn := NewBestN(evaluator, 3)
	candidates := []string{"bad", "good", "ok"}
	
	winner, err := bestn.SelectWinner(candidates, "select best")
	assert.NoError(t, err)
	assert.Equal(t, "good", winner)
}

func TestEvalResultValidation(t *testing.T) {
	// Valid result
	result := &EvalResult{
		WinnerID:        0,
		Recommendations: []string{"rec1"},
		Merged:         "merged",
	}
	assert.True(t, result.IsValid(3)) // 3 candidates
	
	// Invalid: WinnerID out of range
	result.WinnerID = 5
	assert.False(t, result.IsValid(3))
	
	// Invalid: negative WinnerID
	result.WinnerID = -1
	assert.False(t, result.IsValid(3))
}

func TestEvaluatorSchema(t *testing.T) {
	// Test that the JSON schema for evaluator is correct
	schema := GetEvaluatorSchema()
	
	assert.NotNil(t, schema)
	assert.Equal(t, "object", schema["type"])
	
	properties, ok := schema["properties"].(map[string]interface{})
	assert.True(t, ok)
	
	// Check required fields exist
	assert.Contains(t, properties, "winner")
	assert.Contains(t, properties, "recommendations")
	assert.Contains(t, properties, "merged")
	
	// Check required array
	required, ok := schema["required"].([]string)
	assert.True(t, ok)
	assert.Contains(t, required, "winner")
}

func TestEmptyCandidates(t *testing.T) {
	evaluator := new(MockEvaluator)
	bestn := NewBestN(evaluator, 3)
	
	winner, err := bestn.SelectWinner([]string{}, "prompt")
	assert.Error(t, err)
	assert.Empty(t, winner)
}

func TestSingleCandidate(t *testing.T) {
	evaluator := new(MockEvaluator)
	evaluator.On("Evaluate", mock.Anything, mock.Anything).Return(&EvalResult{
		WinnerID: 0,
	}, nil)
	
	bestn := NewBestN(evaluator, 1)
	candidates := []string{"only one"}
	
	winner, err := bestn.SelectWinner(candidates, "prompt")
	assert.NoError(t, err)
	assert.Equal(t, "only one", winner)
}

func TestMergeCandidates(t *testing.T) {
	evaluator := new(MockEvaluator)
	evaluator.On("Evaluate", mock.Anything, mock.Anything).Return(&EvalResult{
		WinnerID: 0,
		Merged:   "merged code",
	}, nil)
	
	bestn := NewBestN(evaluator, 2)
	candidates := []string{"code1", "code2"}
	
	merged, err := bestn.MergeCandidates(candidates, "merge these")
	assert.NoError(t, err)
	assert.Equal(t, "merged code", merged)
}

func TestBestNWithGitDiff(t *testing.T) {
	evaluator := new(MockEvaluator)
	evaluator.On("Evaluate", mock.Anything, mock.Anything).Return(&EvalResult{
		WinnerID: 1,
		Recommendations: []string{"Second diff is cleaner"},
	}, nil)
	
	bestn := NewBestN(evaluator, 3)
	diffs := []string{
		"diff --git a/file1.go...",
		"diff --git a/file2.go...",
		"diff --git a/file3.go...",
	}
	
	result, err := bestn.EvaluateCandidates(diffs, "which diff is best?")
	assert.NoError(t, err)
	assert.Equal(t, 1, result.WinnerID)
}

func TestBestNWithToolCalls(t *testing.T) {
	evaluator := new(MockEvaluator)
	evaluator.On("Evaluate", mock.Anything, mock.Anything).Return(&EvalResult{
		WinnerID: 0,
	}, nil)
	
	bestn := NewBestN(evaluator, 2)
	toolCalls := []string{
		`{"tool": "bash", "args": {"command": "ls"}}`,
		`{"tool": "bash", "args": {"command": "pwd"}}`,
	}
	
	winner, err := bestn.SelectWinner(toolCalls, "which tool call is better?")
	assert.NoError(t, err)
	assert.Equal(t, toolCalls[0], winner)
}

// Test concurrent evaluation
func TestConcurrentEvaluation(t *testing.T) {
	evaluator := new(MockEvaluator)
	evaluator.On("Evaluate", mock.Anything, mock.Anything).Return(&EvalResult{
		WinnerID: 0,
	}, nil)
	
	bestn := NewBestN(evaluator, 3)
	candidates := []string{"a", "b", "c"}
	
	done := make(chan bool)
	for i := 0; i < 5; i++ {
		go func() {
			_, err := bestn.EvaluateCandidates(candidates, "test")
			assert.NoError(t, err)
			done <- true
		}()
	}
	
	for i := 0; i < 5; i++ {
		<-done
	}
}
