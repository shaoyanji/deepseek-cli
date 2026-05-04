package engine

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"deepseek-cli/internal/execpolicy"
)

// MockLLMClient is a mock implementation of LLMClient
type MockLLMClient struct {
	Response *LLMResponse
	Error    error
}

func (m *MockLLMClient) Chat(ctx context.Context, messages []Message, tools []ToolDefinition) (*LLMResponse, error) {
	return m.Response, m.Error
}

func (m *MockLLMClient) StreamChat(ctx context.Context, messages []Message, tools []ToolDefinition, callback func(chunk string)) (*LLMResponse, error) {
	return m.Response, m.Error
}

// MockToolExecutor is a mock implementation of ToolExecutor
type MockToolExecutor struct{}

func (m *MockToolExecutor) Execute(ctx context.Context, name string, args map[string]interface{}) (string, error) {
	return "mock result", nil
}

func TestUpdateSessionUsage_NilGuard(t *testing.T) {
	session, err := NewSession("test-session", "/tmp/test", execpolicy.ModeAgent)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	engine := NewEngine(session, &MockToolExecutor{}, &MockLLMClient{})

	// Test that nil usage doesn't panic
	engine.updateSessionUsage(nil)

	// Verify total usage is still initialized but zero
	if session.TotalUsage == nil {
		t.Error("TotalUsage should be initialized")
	}
	if session.TotalUsage.PromptTokens != 0 {
		t.Errorf("Expected PromptTokens to be 0, got %d", session.TotalUsage.PromptTokens)
	}
	if session.TotalUsage.CompletionTokens != 0 {
		t.Errorf("Expected CompletionTokens to be 0, got %d", session.TotalUsage.CompletionTokens)
	}
	if session.TotalUsage.TotalTokens != 0 {
		t.Errorf("Expected TotalTokens to be 0, got %d", session.TotalUsage.TotalTokens)
	}
	if session.TotalUsage.CostUSD != 0 {
		t.Errorf("Expected CostUSD to be 0, got %f", session.TotalUsage.CostUSD)
	}
}

func TestUpdateSessionUsage_ValidUsage(t *testing.T) {
	session, err := NewSession("test-session", "/tmp/test", execpolicy.ModeAgent)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	engine := NewEngine(session, &MockToolExecutor{}, &MockLLMClient{})

	usage := &TokenUsage{
		PromptTokens:     100,
		CompletionTokens: 50,
		TotalTokens:      150,
		CostUSD:          0.001,
	}

	engine.updateSessionUsage(usage)

	if session.TotalUsage.PromptTokens != 100 {
		t.Errorf("Expected PromptTokens to be 100, got %d", session.TotalUsage.PromptTokens)
	}
	if session.TotalUsage.CompletionTokens != 50 {
		t.Errorf("Expected CompletionTokens to be 50, got %d", session.TotalUsage.CompletionTokens)
	}
	if session.TotalUsage.TotalTokens != 150 {
		t.Errorf("Expected TotalTokens to be 150, got %d", session.TotalUsage.TotalTokens)
	}
	if session.TotalUsage.CostUSD != 0.001 {
		t.Errorf("Expected CostUSD to be 0.001, got %f", session.TotalUsage.CostUSD)
	}
}

func TestExecuteTurn_NilUsageDoesNotPanic(t *testing.T) {
	session, err := NewSession("test-session", "/tmp/test", execpolicy.ModeAgent)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	mockLLM := &MockLLMClient{
		Response: &LLMResponse{
			Content:   "Test response",
			ToolCalls: []ToolCall{},
			Thinking:  "",
			Usage:     nil, // Nil usage to test the guard
		},
	}

	engine := NewEngine(session, &MockToolExecutor{}, mockLLM)

	// This should not panic
	_, err = engine.RunTurn(context.Background(), "test input")
	if err != nil {
		t.Fatalf("RunTurn failed: %v", err)
	}

	// Verify the turn was created successfully
	if len(session.Turns) != 1 {
		t.Errorf("Expected 1 turn, got %d", len(session.Turns))
	}
}

func TestSessionSerialization_ToolErrorString(t *testing.T) {
	// Create a tool result with an error
	toolResult := ToolResult{
		ToolCallID: "call-123",
		Result:     "",
		Error:      errors.New("tool execution failed: permission denied"),
	}

	// Serialize to JSON
	data, err := json.MarshalIndent(toolResult, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal tool result: %v", err)
	}

	// Verify the error is serialized as a string (not an object)
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Failed to unmarshal to raw: %v", err)
	}

	errorField, ok := raw["Error"].(string)
	if !ok {
		t.Errorf("Expected Error field to be a string, got %T", raw["Error"])
	}
	if errorField != "tool execution failed: permission denied" {
		t.Errorf("Expected error string 'tool execution failed: permission denied', got '%s'", errorField)
	}

	// Deserialize back
	var loadedResult ToolResult
	if err := json.Unmarshal(data, &loadedResult); err != nil {
		t.Fatalf("Failed to unmarshal tool result: %v", err)
	}

	// Verify the error was correctly deserialized
	if loadedResult.Error == nil || loadedResult.Error.Error() != "tool execution failed: permission denied" {
		t.Errorf("Expected error string 'tool execution failed: permission denied', got '%v'", loadedResult.Error)
	}
}

func TestSessionSerialization_NoToolError(t *testing.T) {
	// Create a tool result without an error
	toolResult := ToolResult{
		ToolCallID: "call-123",
		Result:     "success",
		Error:      nil,
	}

	// Serialize to JSON
	data, err := json.MarshalIndent(toolResult, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal tool result: %v", err)
	}

	// Deserialize back
	var loadedResult ToolResult
	if err := json.Unmarshal(data, &loadedResult); err != nil {
		t.Fatalf("Failed to unmarshal tool result: %v", err)
	}

	// Verify the tool result was correctly deserialized
	if loadedResult.Error != nil {
		t.Errorf("Expected nil error, got '%v'", loadedResult.Error)
	}
	if loadedResult.Result != "success" {
		t.Errorf("Expected result 'success', got '%s'", loadedResult.Result)
	}
}

func TestBuildMessages_NoDuplicateUserInput(t *testing.T) {
	session, err := NewSession("test-session", "/tmp/test", execpolicy.ModeAgent)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Add a previous turn
	previousTurn := &Turn{
		ID:            1,
		UserInput:     "previous input",
		ModelResponse: "previous response",
		Timestamp:     session.CreatedAt,
		Status:        TurnComplete,
	}
	session.Turns = append(session.Turns, previousTurn)

	engine := NewEngine(session, &MockToolExecutor{}, &MockLLMClient{})

	// Build messages - the current input should only be added once at the end
	messages := engine.buildMessages("current input")

	// Count how many times "current input" appears
	currentInputCount := 0
	for _, msg := range messages {
		if msg.Content == "current input" {
			currentInputCount++
		}
	}

	// Should appear exactly once
	if currentInputCount != 1 {
		t.Errorf("Expected 'current input' to appear exactly once, but it appeared %d times", currentInputCount)
	}

	// Verify the message order is correct
	expectedRoles := []string{"system", "user", "assistant", "user"}
	if len(messages) != len(expectedRoles) {
		t.Errorf("Expected %d messages, got %d", len(expectedRoles), len(messages))
	}

	for i, expectedRole := range expectedRoles {
		if i < len(messages) && messages[i].Role != expectedRole {
			t.Errorf("Expected message %d to have role '%s', got '%s'", i, expectedRole, messages[i].Role)
		}
	}
}

func TestSessionSerialization_PolicySkipped(t *testing.T) {
	session, err := NewSession("test-session", "/tmp/test", execpolicy.ModeYOLO)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Verify policy is set initially
	if session.Policy == nil {
		t.Error("Policy should be set after NewSession")
	}

	engine := NewEngine(session, &MockToolExecutor{}, &MockLLMClient{})

	// Serialize session
	data, err := engine.SaveSession()
	if err != nil {
		t.Fatalf("Failed to save session: %v", err)
	}

	// Verify Policy field is not in the JSON
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Failed to unmarshal to raw: %v", err)
	}

	if _, exists := raw["Policy"]; exists {
		t.Error("Policy field should not be present in JSON (should be skipped)")
	}

	// Deserialize session
	loadedSession, err := LoadSession(data)
	if err != nil {
		t.Fatalf("Failed to load session: %v", err)
	}

	// Verify policy was recreated
	if loadedSession.Policy == nil {
		t.Error("Policy should be recreated after LoadSession")
	}

	// Verify mode was preserved
	if loadedSession.Mode != execpolicy.ModeYOLO {
		t.Errorf("Expected mode %v, got %v", execpolicy.ModeYOLO, loadedSession.Mode)
	}

	// Verify other fields were preserved
	if loadedSession.ID != session.ID {
		t.Errorf("Expected ID %s, got %s", session.ID, loadedSession.ID)
	}
}

func TestSessionSerialization_WithToolErrors(t *testing.T) {
	session, err := NewSession("test-session", "/tmp/test", execpolicy.ModeAgent)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Add a turn with tool errors
	turn := &Turn{
		ID:        1,
		UserInput: "test input",
		Timestamp: session.CreatedAt,
		Status:    TurnComplete,
		ToolResults: []ToolResult{
			{
				ToolCallID: "call-1",
				Result:     "",
				Error:      errors.New("tool failed: permission denied"),
			},
			{
				ToolCallID: "call-2",
				Result:     "success",
				Error:      nil,
			},
		},
	}
	session.Turns = append(session.Turns, turn)

	engine := NewEngine(session, &MockToolExecutor{}, &MockLLMClient{})

	// Serialize session
	data, err := engine.SaveSession()
	if err != nil {
		t.Fatalf("Failed to save session: %v", err)
	}

	// Deserialize session
	loadedSession, err := LoadSession(data)
	if err != nil {
		t.Fatalf("Failed to load session: %v", err)
	}

	// Verify tool errors were preserved
	if len(loadedSession.Turns) != 1 {
		t.Errorf("Expected 1 turn, got %d", len(loadedSession.Turns))
	}
	if len(loadedSession.Turns[0].ToolResults) != 2 {
		t.Errorf("Expected 2 tool results, got %d", len(loadedSession.Turns[0].ToolResults))
	}
	if loadedSession.Turns[0].ToolResults[0].Error == nil || loadedSession.Turns[0].ToolResults[0].Error.Error() != "tool failed: permission denied" {
		t.Errorf("Expected error 'tool failed: permission denied', got '%v'", loadedSession.Turns[0].ToolResults[0].Error)
	}
	if loadedSession.Turns[0].ToolResults[1].Error != nil {
		t.Errorf("Expected nil error, got '%v'", loadedSession.Turns[0].ToolResults[1].Error)
	}
}

func TestGetToolDefinitions_ByMode(t *testing.T) {
	// Test Acme mode (read-only)
	acmeSession, err := NewSession("test-acme", "/tmp/test", execpolicy.ModeAcme)
	if err != nil {
		t.Fatalf("Failed to create acme session: %v", err)
	}
	acmeEngine := NewEngine(acmeSession, &MockToolExecutor{}, &MockLLMClient{})
	acmeTools := acmeEngine.getToolDefinitions()

	// Verify only read-only tools are present
	acmeToolNames := make(map[string]bool)
	for _, tool := range acmeTools {
		acmeToolNames[tool.Name] = true
	}

	expectedReadOnly := []string{"view", "ls", "grep", "fetch", "web_search", "lsp"}
	for _, name := range expectedReadOnly {
		if !acmeToolNames[name] {
			t.Errorf("Acme mode should include read-only tool '%s'", name)
		}
	}

	// Verify write tools are NOT present
	writeTools := []string{"edit", "bash"}
	for _, name := range writeTools {
		if acmeToolNames[name] {
			t.Errorf("Acme mode should NOT include write tool '%s'", name)
		}
	}

	// Test Agent mode (read-only + write)
	agentSession, err := NewSession("test-agent", "/tmp/test", execpolicy.ModeAgent)
	if err != nil {
		t.Fatalf("Failed to create agent session: %v", err)
	}
	agentEngine := NewEngine(agentSession, &MockToolExecutor{}, &MockLLMClient{})
	agentTools := agentEngine.getToolDefinitions()

	// Verify all tools are present
	agentToolNames := make(map[string]bool)
	for _, tool := range agentTools {
		agentToolNames[tool.Name] = true
	}

	allExpectedTools := append(expectedReadOnly, writeTools...)
	for _, name := range allExpectedTools {
		if !agentToolNames[name] {
			t.Errorf("Agent mode should include tool '%s'", name)
		}
	}

	// Test YOLO mode (read-only + write)
	yoloSession, err := NewSession("test-yolo", "/tmp/test", execpolicy.ModeYOLO)
	if err != nil {
		t.Fatalf("Failed to create yolo session: %v", err)
	}
	yoloEngine := NewEngine(yoloSession, &MockToolExecutor{}, &MockLLMClient{})
	yoloTools := yoloEngine.getToolDefinitions()

	// Verify all tools are present
	yoloToolNames := make(map[string]bool)
	for _, tool := range yoloTools {
		yoloToolNames[tool.Name] = true
	}

	for _, name := range allExpectedTools {
		if !yoloToolNames[name] {
			t.Errorf("YOLO mode should include tool '%s'", name)
		}
	}
}