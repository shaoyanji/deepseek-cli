package speculative

import (
	"fmt"
	"testing"
)

func TestNewAdaptiveSpeculativeDecoder(t *testing.T) {
	client := new(MockAPIClient)
	decoder := NewAdaptiveSpeculativeDecoder(client, "deepseek-v4-flash", "deepseek-v4-pro", 5)
	
	if decoder == nil {
		t.Fatal("NewAdaptiveSpeculativeDecoder() returned nil")
	}
	
	if decoder.FlashModel != "deepseek-v4-flash" {
		t.Errorf("FlashModel = %q, want %q", decoder.FlashModel, "deepseek-v4-flash")
	}
	
	if decoder.ProModel != "deepseek-v4-pro" {
		t.Errorf("ProModel = %q, want %q", decoder.ProModel, "deepseek-v4-pro")
	}
	
	if decoder.DifficultyLevel != 0 {
		t.Errorf("DifficultyLevel = %d, want 0", decoder.DifficultyLevel)
	}
	
	if decoder.FailureCount != 0 {
		t.Errorf("FailureCount = %d, want 0", decoder.FailureCount)
	}
}

func TestRecordFailure(t *testing.T) {
	client := new(MockAPIClient)
	decoder := NewAdaptiveSpeculativeDecoder(client, "flash", "pro", 5)
	
	// Initial state
	if decoder.GetDifficultyLevel() != 0 {
		t.Errorf("Initial difficulty = %d, want 0", decoder.GetDifficultyLevel())
	}
	
	// After 1 failure
	decoder.RecordFailure()
	if decoder.DifficultyLevel != 1 {
		t.Errorf("After 1 failure, difficulty = %d, want 1", decoder.DifficultyLevel)
	}
	
	// After 3 failures (total)
	decoder.RecordFailure()
	decoder.RecordFailure()
	if decoder.DifficultyLevel != 2 {
		t.Errorf("After 3 failures, difficulty = %d, want 2", decoder.DifficultyLevel)
	}
	
	// After 6 failures (total)
	decoder.RecordFailure()
	decoder.RecordFailure()
	decoder.RecordFailure()
	if decoder.DifficultyLevel != 3 {
		t.Errorf("After 6 failures, difficulty = %d, want 3", decoder.DifficultyLevel)
	}
}

func TestReset(t *testing.T) {
	client := new(MockAPIClient)
	decoder := NewAdaptiveSpeculativeDecoder(client, "flash", "pro", 5)
	
	// Set some failures
	decoder.RecordFailure()
	decoder.RecordFailure()
	decoder.RecordFailure()
	
	if decoder.FailureCount == 0 || decoder.DifficultyLevel == 0 {
		t.Error("Expected non-zero failure count and difficulty")
	}
	
	// Reset
	decoder.Reset()
	
	if decoder.FailureCount != 0 {
		t.Errorf("After reset, FailureCount = %d, want 0", decoder.FailureCount)
	}
	
	if decoder.DifficultyLevel != 0 {
		t.Errorf("After reset, DifficultyLevel = %d, want 0", decoder.DifficultyLevel)
	}
}

func TestShouldUsePro(t *testing.T) {
	client := new(MockAPIClient)
	decoder := NewAdaptiveSpeculativeDecoder(client, "flash", "pro", 5)
	
	// Should not use Pro initially
	if decoder.ShouldUsePro() {
		t.Error("ShouldUsePro() = true, want false (initial state)")
	}
	
	// After 2 failures (difficulty = 1)
	decoder.RecordFailure()
	decoder.RecordFailure()
	if decoder.ShouldUsePro() {
		t.Error("ShouldUsePro() = true, want false (difficulty 1)")
	}
	
	// After 3 failures (difficulty = 2)
	decoder.RecordFailure()
	if !decoder.ShouldUsePro() {
		t.Error("ShouldUsePro() = false, want true (difficulty 2)")
	}
	
	// Reset and test failure count threshold
	decoder.Reset()
	decoder.FailureCount = 3
	decoder.DifficultyLevel = 0 // Keep difficulty low
	
	if !decoder.ShouldUsePro() {
		t.Error("ShouldUsePro() = false, want true (failure count >= 3)")
	}
}

func TestSpawnVariadicCalls(t *testing.T) {
	client := new(MockAPIClient)
	
	// Setup mock to return different responses using ChatCompletionFunc
	callIndex := 0
	client.ChatCompletionFunc = func(req interface{}) (interface{}, error) {
		responses := []*ChatResponse{
			{Choices: []Choice{{Message: Message{Content: "response1"}}}},
			{Choices: []Choice{{Message: Message{Content: "response2"}}}},
			{Choices: []Choice{{Message: Message{Content: "response3"}}}},
		}
		
		if callIndex >= len(responses) {
			return nil, fmt.Errorf("no more responses")
		}
		resp := responses[callIndex]
		callIndex++
		return resp, nil
	}
	
	decoder := NewAdaptiveSpeculativeDecoder(client, "flash", "pro", 5)
	
	candidates, err := decoder.SpawnVariadicCalls("test prompt", 3)
	if err != nil {
		t.Fatalf("SpawnVariadicCalls() error = %v", err)
	}
	
	if len(candidates) != 3 {
		t.Errorf("SpawnVariadicCalls() returned %d candidates, want 3", len(candidates))
	}
	
	expectedCandidates := []string{"response1", "response2", "response3"}
	for i, cand := range candidates {
		if cand != expectedCandidates[i] {
			t.Errorf("Candidate %d = %q, want %q", i, cand, expectedCandidates[i])
		}
	}
}

func TestSpawnVariadicCallsWithFailures(t *testing.T) {
	client := new(MockAPIClient)
	
	// Setup mock with some failures - use ChatCompletionFunc for fine-grained control
	callIndex := 0
	client.ChatCompletionFunc = func(req interface{}) (interface{}, error) {
		responses := []interface{}{
			&ChatResponse{Choices: []Choice{{Message: Message{Content: "response1"}}}},
			nil, // This will cause an error
			&ChatResponse{Choices: []Choice{{Message: Message{Content: "response3"}}}},
		}
		
		if callIndex >= len(responses) {
			return nil, nil
		}
		resp := responses[callIndex]
		callIndex++
		if resp == nil {
			return nil, fmt.Errorf("mock error")
		}
		return resp, nil
	}
	
	decoder := NewAdaptiveSpeculativeDecoder(client, "flash", "pro", 5)
	
	candidates, err := decoder.SpawnVariadicCalls("test prompt", 3)
	if err != nil {
		t.Fatalf("SpawnVariadicCalls() error = %v", err)
	}
	
	// Should have at least 2 successful responses
	if len(candidates) < 2 {
		t.Errorf("SpawnVariadicCalls() returned %d candidates, want at least 2", len(candidates))
	}
}

func TestEvaluateAndSelect(t *testing.T) {
	client := new(MockAPIClient)
	
	// Mock evaluation response that selects candidate 1
	client.ChatCompletionFunc = func(req interface{}) (interface{}, error) {
		return &ChatResponse{
			Choices: []Choice{{Message: Message{Content: "1"}}},
		}, nil
	}
	
	decoder := NewAdaptiveSpeculativeDecoder(client, "flash", "pro", 5)
	
	candidates := []string{"bad", "good", "ok"}
	result, err := decoder.EvaluateAndSelect(candidates, "select the best code")
	if err != nil {
		t.Fatalf("EvaluateAndSelect() error = %v", err)
	}
	
	if result != "good" {
		t.Errorf("EvaluateAndSelect() = %q, want %q", result, "good")
	}
}

func TestEvaluateAndSelectEmptyCandidates(t *testing.T) {
	client := new(MockAPIClient)
	decoder := NewAdaptiveSpeculativeDecoder(client, "flash", "pro", 5)
	
	_, err := decoder.EvaluateAndSelect([]string{}, "prompt")
	if err == nil {
		t.Error("EvaluateAndSelect() should error with empty candidates")
	}
}

func TestEvaluateAndSelectSingleCandidate(t *testing.T) {
	client := new(MockAPIClient)
	decoder := NewAdaptiveSpeculativeDecoder(client, "flash", "pro", 5)
	
	candidates := []string{"only one"}
	result, err := decoder.EvaluateAndSelect(candidates, "prompt")
	if err != nil {
		t.Fatalf("EvaluateAndSelect() error = %v", err)
	}
	
	if result != "only one" {
		t.Errorf("EvaluateAndSelect() = %q, want %q", result, "only one")
	}
}

func TestAdaptiveDecodeLowDifficulty(t *testing.T) {
	client := new(MockAPIClient)
	
	// Setup for normal speculative decoding flow
	callCount := 0
	client.ChatCompletionFunc = func(req interface{}) (interface{}, error) {
		callCount++
		if callCount == 1 {
			// Draft model response
			return &ChatResponse{
				Choices: []Choice{{Message: Message{Content: "draft tokens"}}},
			}, nil
		}
		// Target model verification
		return &ChatResponse{
			Choices: []Choice{{
				Message:  Message{Content: "verified"},
				LogProbs: &LogProbs{Content: []TokenLogProb{{Token: "draft", LogProb: -0.1}}},
			}},
		}, nil
	}
	
	decoder := NewAdaptiveSpeculativeDecoder(client, "flash", "pro", 5)
	
	result, err := decoder.AdaptiveDecode("test prompt")
	if err != nil {
		t.Fatalf("AdaptiveDecode() error = %v", err)
	}
	
	if result == "" {
		t.Error("AdaptiveDecode() returned empty result")
	}
}

func TestAdaptiveDecodeHighDifficulty(t *testing.T) {
	client := new(MockAPIClient)
	
	// Set high difficulty - use ChatCompletionFunc for proper response type
	client.ChatCompletionFunc = func(req interface{}) (interface{}, error) {
		return &ChatResponse{
			Choices: []Choice{{Message: Message{Content: "pro response"}}},
		}, nil
	}
	
	decoder := NewAdaptiveSpeculativeDecoder(client, "flash", "pro", 5)
	decoder.DifficultyLevel = 3 // High difficulty
	
	result, err := decoder.AdaptiveDecode("test prompt")
	if err != nil {
		t.Fatalf("AdaptiveDecode() error = %v", err)
	}
	
	if result != "pro response" {
		t.Errorf("AdaptiveDecode() = %q, want %q", result, "pro response")
	}
}
