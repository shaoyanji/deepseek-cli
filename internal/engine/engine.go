// Package engine provides the core agent loop and orchestration logic.
package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"deepseek-cli/internal/execpolicy"
)

// Turn represents a single interaction turn in the agent session
type Turn struct {
	ID            int
	UserInput     string
	ModelResponse string
	ToolCalls     []ToolCall
	ToolResults   []ToolResult
	TokenUsage    *TokenUsage
	Thinking      string
	Timestamp     time.Time
	Status        TurnStatus
}

// TurnStatus represents the status of a turn
type TurnStatus string

const (
	TurnPending   TurnStatus = "pending"
	TurnRunning   TurnStatus = "running"
	TurnComplete  TurnStatus = "complete"
	TurnFailed    TurnStatus = "failed"
	TurnCancelled TurnStatus = "cancelled"
)

// ToolCall represents a tool invocation request from the model
type ToolCall struct {
	ID        string
	Name      string
	Arguments map[string]interface{}
}

// ToolResult represents the result of a tool execution
type ToolResult struct {
	ToolCallID string
	Result     string
	Error      string
}

// TokenUsage tracks token consumption
type TokenUsage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
	CostUSD          float64
}

// Session represents an agent session state
type Session struct {
	ID           string
	Turns        []*Turn
	CurrentTurn  int
	Model        string
	Mode         execpolicy.ExecutionMode
	Policy       execpolicy.Policy `json:"-"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	TotalUsage   *TokenUsage
	WorkspacePath string
}

// ToolExecutor defines the interface for executing tools
type ToolExecutor interface {
	Execute(ctx context.Context, name string, args map[string]interface{}) (string, error)
}

// LLMClient defines the interface for LLM interactions
type LLMClient interface {
	Chat(ctx context.Context, messages []Message, tools []ToolDefinition) (*LLMResponse, error)
	StreamChat(ctx context.Context, messages []Message, tools []ToolDefinition, callback func(chunk string)) (*LLMResponse, error)
}

// Message represents a chat message
type Message struct {
	Role    string
	Content string
}

// ToolDefinition defines a tool's schema
type ToolDefinition struct {
	Name        string
	Description string
	Parameters  map[string]interface{}
}

// LLMResponse represents the response from an LLM call
type LLMResponse struct {
	Content   string
	ToolCalls []ToolCall
	Thinking  string
	Usage     *TokenUsage
}

// Engine is the core agent loop engine
type Engine struct {
	session      *Session
	toolExecutor ToolExecutor
	llmClient    LLMClient
	mu           sync.RWMutex
	callbacks    EngineCallbacks
}

// EngineCallbacks holds callback functions for various engine events
type EngineCallbacks struct {
	OnTurnStart   func(turn *Turn)
	OnTurnEnd     func(turn *Turn)
	OnToolCall    func(call *ToolCall)
	OnToolResult  func(result *ToolResult)
	OnThinking    func(thinking string)
	OnTokenUsage  func(usage *TokenUsage)
}

// NewEngine creates a new agent engine
func NewEngine(session *Session, toolExecutor ToolExecutor, llmClient LLMClient) *Engine {
	return &Engine{
		session:      session,
		toolExecutor: toolExecutor,
		llmClient:    llmClient,
		callbacks:    EngineCallbacks{},
	}
}

// SetCallback sets a callback function
func (e *Engine) SetCallback(callback EngineCallbacks) {
	e.callbacks = callback
}

// GetSession returns the current session
func (e *Engine) GetSession() *Session {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.session
}

// RunTurn executes a single turn of the agent loop
func (e *Engine) RunTurn(ctx context.Context, userInput string) (*Turn, error) {
	e.mu.Lock()
	
	// Create new turn
	turn := &Turn{
		ID:        len(e.session.Turns) + 1,
		UserInput: userInput,
		Timestamp: time.Now(),
		Status:    TurnPending,
	}
	
	e.session.Turns = append(e.session.Turns, turn)
	e.session.CurrentTurn = turn.ID
	e.session.UpdatedAt = time.Now()
	
	e.mu.Unlock()
	
	// Notify turn start
	if e.callbacks.OnTurnStart != nil {
		e.callbacks.OnTurnStart(turn)
	}
	
	// Execute the turn
	err := e.executeTurn(ctx, turn)
	
	// Notify turn end
	if e.callbacks.OnTurnEnd != nil {
		e.callbacks.OnTurnEnd(turn)
	}
	
	return turn, err
}

func (e *Engine) executeTurn(ctx context.Context, turn *Turn) error {
	turn.Status = TurnRunning
	
	// Build messages from session history
	messages := e.buildMessages()
	
	// Get tool definitions
	tools := e.getToolDefinitions()
	
	// Call LLM
	response, err := e.llmClient.Chat(ctx, messages, tools)
	if err != nil {
		turn.Status = TurnFailed
		return fmt.Errorf("LLM call failed: %w", err)
	}
	
	turn.ModelResponse = response.Content
	turn.Thinking = response.Thinking
	turn.ToolCalls = response.ToolCalls
	turn.TokenUsage = response.Usage
	
	// Update session usage
	e.updateSessionUsage(response.Usage)
	
	// Notify thinking updates
	if e.callbacks.OnThinking != nil && response.Thinking != "" {
		e.callbacks.OnThinking(response.Thinking)
	}
	
	// Notify token usage
	if e.callbacks.OnTokenUsage != nil {
		e.callbacks.OnTokenUsage(response.Usage)
	}
	
	// Execute tool calls if any
	if len(response.ToolCalls) > 0 {
		if err := e.executeToolCalls(ctx, turn); err != nil {
			turn.Status = TurnFailed
			return fmt.Errorf("tool execution failed: %w", err)
		}
	}
	
	turn.Status = TurnComplete
	return nil
}

func (e *Engine) buildMessages() []Message {
	var messages []Message
	
	// Add system message based on mode
	systemMsg := e.getSystemMessage()
	if systemMsg != "" {
		messages = append(messages, Message{Role: "system", Content: systemMsg})
	}
	
	// Add previous turns
	for _, turn := range e.session.Turns {
		if turn.UserInput != "" {
			messages = append(messages, Message{Role: "user", Content: turn.UserInput})
		}
		if turn.ModelResponse != "" {
			messages = append(messages, Message{Role: "assistant", Content: turn.ModelResponse})
		}
		// Add tool results
		for _, result := range turn.ToolResults {
			if result.Error == "" {
				messages = append(messages, Message{
					Role:    "tool",
					Content: result.Result,
				})
			}
		}
	}
	
	return messages
}

func (e *Engine) getSystemMessage() string {
	switch e.session.Mode {
	case execpolicy.ModeAcme:
		return `You are in Acme (Plan) mode - a read-only exploration mode.
You can only use read-only tools: view, ls, grep, fetch, web_search, lsp.
Focus on understanding the codebase and providing analysis.
Do not attempt to modify files or run commands.`
		
	case execpolicy.ModeAgent:
		return `You are in Agent mode - an interactive mode with human oversight.
Before using any tool, explain what you plan to do and why.
Wait for user approval before executing tools.
Be thorough in your explanations.`
		
	case execpolicy.ModeYOLO:
		return `You are in YOLO mode - fully automated execution.
You can execute tools without explicit approval.
Still explain your actions clearly.
Take initiative to complete tasks efficiently.`
		
	default:
		return ""
	}
}

func (e *Engine) getToolDefinitions() []ToolDefinition {
	// Read-only tools available in all modes
	readOnlyTools := []ToolDefinition{
		{
			Name:        "view",
			Description: "View the contents of a file",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "Path to the file to view",
					},
				},
				"required": []string{"path"},
			},
		},
		{
			Name:        "ls",
			Description: "List files and directories in a path",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "Path to list (defaults to current directory)",
					},
				},
				"required": []string{},
			},
		},
		{
			Name:        "grep",
			Description: "Search for patterns in files using regular expressions",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"pattern": map[string]interface{}{
						"type":        "string",
						"description": "Regular expression pattern to search for",
					},
					"path": map[string]interface{}{
						"type":        "string",
						"description": "Directory or file to search in (defaults to current directory)",
					},
				},
				"required": []string{"pattern"},
			},
		},
		{
			Name:        "fetch",
			Description: "Fetch content from a URL",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"url": map[string]interface{}{
						"type":        "string",
						"description": "URL to fetch content from",
					},
				},
				"required": []string{"url"},
			},
		},
		{
			Name:        "web_search",
			Description: "Search the web for information",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "Search query",
					},
				},
				"required": []string{"query"},
			},
		},
		{
			Name:        "lsp",
			Description: "Get language server protocol information (symbols, definitions, references)",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "File path to analyze",
					},
					"query": map[string]interface{}{
						"type":        "string",
						"description": "LSP query type (symbols, definitions, references)",
					},
				},
				"required": []string{"path"},
			},
		},
	}

	// Write tools only available in Agent and YOLO modes
	writeTools := []ToolDefinition{
		{
			Name:        "edit",
			Description: "Edit or create a file with new content",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "Path to the file to edit",
					},
					"content": map[string]interface{}{
						"type":        "string",
						"description": "New content for the file",
					},
				},
				"required": []string{"path", "content"},
			},
		},
		{
			Name:        "bash",
			Description: "Execute a bash command",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"command": map[string]interface{}{
						"type":        "string",
						"description": "Bash command to execute",
					},
				},
				"required": []string{"command"},
			},
		},
	}

	// Return appropriate tool set based on mode
	switch e.session.Mode {
	case execpolicy.ModeAcme:
		return readOnlyTools
	case execpolicy.ModeAgent, execpolicy.ModeYOLO:
		return append(readOnlyTools, writeTools...)
	default:
		return readOnlyTools
	}
}

func (e *Engine) executeToolCalls(ctx context.Context, turn *Turn) error {
	for _, call := range turn.ToolCalls {
		// Check policy approval
		approval, err := e.session.Policy.ApproveTool(call.Name, call.Arguments, "Tool execution requested")
		if err != nil {
			turn.ToolResults = append(turn.ToolResults, ToolResult{
				ToolCallID: call.ID,
				Error:      err.Error(),
			})
			continue
		}
		
		if !approval.Approved {
			turn.ToolResults = append(turn.ToolResults, ToolResult{
				ToolCallID: call.ID,
				Error:      fmt.Sprintf("tool execution denied: %s", approval.Reason),
			})
			continue
		}
		
		// Notify tool call
		if e.callbacks.OnToolCall != nil {
			e.callbacks.OnToolCall(&call)
		}
		
		// Execute tool
		result, err := e.toolExecutor.Execute(ctx, call.Name, call.Arguments)
		
		var errStr string
		if err != nil {
			errStr = err.Error()
		}
		
		toolResult := ToolResult{
			ToolCallID: call.ID,
			Result:     result,
			Error:      errStr,
		}
		
		turn.ToolResults = append(turn.ToolResults, toolResult)
		
		// Notify tool result
		if e.callbacks.OnToolResult != nil {
			e.callbacks.OnToolResult(&toolResult)
		}
	}
	
	return nil
}

func (e *Engine) updateSessionUsage(usage *TokenUsage) {
	if usage == nil {
		return
	}
	
	if e.session.TotalUsage == nil {
		e.session.TotalUsage = &TokenUsage{}
	}
	
	e.session.TotalUsage.PromptTokens += usage.PromptTokens
	e.session.TotalUsage.CompletionTokens += usage.CompletionTokens
	e.session.TotalUsage.TotalTokens += usage.TotalTokens
	e.session.TotalUsage.CostUSD += usage.CostUSD
}

// SaveSession serializes the session to JSON
func (e *Engine) SaveSession() ([]byte, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	
	return json.MarshalIndent(e.session, "", "  ")
}

// LoadSession deserializes a session from JSON
func LoadSession(data []byte) (*Session, error) {
	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, err
	}
	
	// Initialize policy based on mode
	factory := execpolicy.NewPolicyFactory()
	policy, err := factory.Create(session.Mode)
	if err != nil {
		return nil, err
	}
	session.Policy = policy
	
	return &session, nil
}

// NewSession creates a new session
func NewSession(id, workspacePath string, mode execpolicy.ExecutionMode) (*Session, error) {
	factory := execpolicy.NewPolicyFactory()
	policy, err := factory.Create(mode)
	if err != nil {
		return nil, err
	}
	
	return &Session{
		ID:            id,
		Mode:          mode,
		Policy:        policy,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
		WorkspacePath: workspacePath,
		TotalUsage:    &TokenUsage{},
	}, nil
}
