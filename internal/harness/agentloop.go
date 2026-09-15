package harness

import (
	"context"
	"fmt"
	"sync"
)

// AgentLoop is the default agent driver that creates agents and runs turn/step lifecycles
type AgentLoop struct {
	mu                   sync.Mutex
	ctx                  *Context
	maxParallelToolCalls int
	agents               map[string]*agentInstance
	config               AgentLoopConfig
}

// AgentLoopConfig configures the agent loop
type AgentLoopConfig struct {
	MaxParallelToolCalls int
	Agents               []AgentConfig
}

// AgentConfig defines a declarative agent configuration
type AgentConfig struct {
	ID              string `yaml:"id"`
	Provider        string `yaml:"provider"`
	Model           string `yaml:"model"`
	ReasoningEffort string `yaml:"reasoningEffort,omitempty"`
	CWD             string `yaml:"cwd,omitempty"`
	SessionID       string `yaml:"sessionId,omitempty"`
	ResumeSessionID string `yaml:"resumeSessionId,omitempty"`
	MaxTokens       int    `yaml:"maxTokens,omitempty"`
}

// agentInstance represents a running agent
type agentInstance struct {
	id          string
	sessionID   string
	handle      AgentHandle
	ctx         *Context
	cancel      context.CancelFunc
	isRunning   bool
	inbox       []Message
	currentStep int
}

// NewAgentLoop creates a new agent loop with the given configuration
func NewAgentLoop(config AgentLoopConfig) *AgentLoop {
	return &AgentLoop{
		maxParallelToolCalls: config.MaxParallelToolCalls,
		agents:               make(map[string]*agentInstance),
		config:               config,
	}
}

// Name returns the plugin name
func (l *AgentLoop) Name() string {
	return "dsh-agent-loop"
}

// Install mounts the agent loop plugin into the context
func (l *AgentLoop) Install(ctx *Context) error {
	l.mu.Lock()
	l.ctx = ctx
	
	// Register as the agent registry
	ctx.agents = l
	l.mu.Unlock()
	
	// Start declarative agents from config (outside lock to avoid deadlock)
	for _, agentConfig := range l.config.Agents {
		if err := l.startDeclarativeAgent(agentConfig); err != nil {
			return fmt.Errorf("failed to start declarative agent %s: %w", agentConfig.ID, err)
		}
	}
	
	return nil
}

// Uninstall unmounts the agent loop plugin
func (l *AgentLoop) Uninstall(ctx *Context) error {
	l.mu.Lock()
	// Collect agent IDs first to avoid deadlock with Dispose
	agentIDs := make([]string, 0, len(l.agents))
	for id := range l.agents {
		agentIDs = append(agentIDs, id)
	}
	l.mu.Unlock()
	
	// Dispose all agents
	for _, id := range agentIDs {
		if err := l.Dispose(id); err != nil {
			return fmt.Errorf("failed to dispose agent %s: %w", id, err)
		}
	}
	
	return nil
}

// startDeclarativeAgent starts an agent from configuration
func (l *AgentLoop) startDeclarativeAgent(cfg AgentConfig) error {
	options := AgentOptions{
		ID:              cfg.ID,
		Provider:        cfg.Provider,
		Model:           cfg.Model,
		ReasoningEffort: cfg.ReasoningEffort,
		CWD:             cfg.CWD,
	}
	
	if cfg.SessionID != "" {
		options.SessionID = cfg.SessionID
	} else if cfg.ResumeSessionID != "" {
		// Resume existing session
		handle, err := l.Resume(cfg.ResumeSessionID)
		if err != nil {
			return err
		}
		l.mu.Lock()
		l.agents[cfg.ID] = &agentInstance{
			id:        cfg.ID,
			sessionID: handle.SessionID(),
			handle:    handle,
			isRunning: true,
		}
		l.mu.Unlock()
		return nil
	}
	
	handle, err := l.Create(options)
	if err != nil {
		return err
	}
	
	l.mu.Lock()
	l.agents[cfg.ID] = &agentInstance{
		id:        cfg.ID,
		sessionID: handle.SessionID(),
		handle:    handle,
		isRunning: true,
	}
	l.mu.Unlock()
	
	return nil
}

// Create creates a new agent instance
func (l *AgentLoop) Create(options AgentOptions) (AgentHandle, error) {
	// Generate session ID if not provided
	sessionID := options.SessionID
	if sessionID == "" {
		sessionID = fmt.Sprintf("%s-session-%d", options.ID, currentTimeMillis())
	}
	
	// Create session if sessions service is available
	var session SessionHandle
	if l.ctx != nil && l.ctx.sessions != nil {
		header := SessionHeader{
			ID:        sessionID,
			CreatedAt: currentTimeMillis(),
			Model:     options.Model,
			Provider:  options.Provider,
			Meta: map[string]interface{}{
				"cwd": options.CWD,
			},
		}
		
		var err error
		session, err = l.ctx.sessions.Create(header)
		if err != nil {
			return nil, fmt.Errorf("failed to create session: %w", err)
		}
	}
	
	// Create agent scope
	var scope *Scope
	if l.ctx != nil {
		scope = l.ctx.CreateScope(sessionID, l.ctx)
	}
	
	// Create agent instance
	instance := &agentInstance{
		id:        options.ID,
		sessionID: sessionID,
		ctx:       l.ctx,
		inbox:     make([]Message, 0),
		isRunning: true,
	}
	
	handle := &agentHandleImpl{
		instance:  instance,
		session:   session,
		scope:     scope,
		loop:      l,
	}
	
	l.mu.Lock()
	l.agents[sessionID] = instance
	l.mu.Unlock()
	
	return handle, nil
}

// Resume resumes a persisted agent session
func (l *AgentLoop) Resume(sessionID string) (AgentHandle, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	// Check if already running
	if instance, exists := l.agents[sessionID]; exists {
		return &agentHandleImpl{
			instance: instance,
		}, nil
	}
	
	// Open existing session
	if l.ctx.sessions == nil {
		return nil, fmt.Errorf("session service not available")
	}
	
	session, err := l.ctx.sessions.Open(sessionID, SessionWrite)
	if err != nil {
		return nil, fmt.Errorf("failed to open session: %w", err)
	}
	
	// Create agent instance
	instance := &agentInstance{
		id:        sessionID,
		sessionID: sessionID,
		ctx:       l.ctx,
		inbox:     make([]Message, 0),
		isRunning: true,
	}
	
	handle := &agentHandleImpl{
		instance:  instance,
		session:   session,
		scope:     l.ctx.CreateScope(sessionID, l.ctx),
		llm:       l.ctx.llm,
		tools:     l.ctx.tools,
		loop:      l,
	}
	
	l.agents[sessionID] = instance
	
	return handle, nil
}

// Get retrieves an agent by ID
func (l *AgentLoop) Get(id string) (AgentHandle, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	instance, exists := l.agents[id]
	if !exists {
		return nil, fmt.Errorf("agent not found: %s", id)
	}
	
	return &agentHandleImpl{
		instance: instance,
	}, nil
}

// List returns all active agents
func (l *AgentLoop) List() []AgentHandle {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	handles := make([]AgentHandle, 0, len(l.agents))
	for _, instance := range l.agents {
		handles = append(handles, &agentHandleImpl{
			instance: instance,
		})
	}
	
	return handles
}

// Dispose tears down an agent
func (l *AgentLoop) Dispose(id string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	instance, exists := l.agents[id]
	if !exists {
		return fmt.Errorf("agent not found: %s", id)
	}
	
	// Cancel current activity
	if instance.cancel != nil {
		instance.cancel()
	}
	
	// Close session if available
	// Note: In full implementation, session would be tracked on instance
	
	delete(l.agents, id)
	instance.isRunning = false
	
	return nil
}

// agentHandleImpl implements AgentHandle
type agentHandleImpl struct {
	instance *agentInstance
	session  SessionHandle
	scope    *Scope
	llm      LLMService
	tools    ToolRegistry
	loop     *AgentLoop
}

func (h *agentHandleImpl) ID() string {
	return h.instance.id
}

func (h *agentHandleImpl) SessionID() string {
	return h.instance.sessionID
}

func (h *agentHandleImpl) Cancel(keepInbox bool) error {
	if h.instance.cancel != nil {
		h.instance.cancel()
	}
	if !keepInbox {
		h.instance.inbox = make([]Message, 0)
	}
	return nil
}

func (h *agentHandleImpl) IsRunning() bool {
	return h.instance.isRunning
}

func (h *agentHandleImpl) Context() *Context {
	return h.instance.ctx
}

// Run executes the agent's main loop
func (h *agentHandleImpl) Run(ctx context.Context) error {
	for h.instance.isRunning {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if err := h.runTurn(); err != nil {
				return err
			}
		}
	}
	return nil
}

// runTurn executes one turn of the agent loop
func (h *agentHandleImpl) runTurn() error {
	// Claim next-step input from inbox
	if len(h.instance.inbox) == 0 {
		// No input, wait
		return nil
	}
	
	// Derive messages from session history
	messages, err := h.deriveMessages()
	if err != nil {
		return fmt.Errorf("failed to derive messages: %w", err)
	}
	
	// Get tool schemas
	toolDefs := h.tools.List()
	
	// Prepare LLM request
	request := LLMRequest{
		Messages: messages,
		Tools:    toolDefs,
	}
	
	// Execute model call
	if h.llm == nil {
		return fmt.Errorf("LLM service not available")
	}
	
	preparedCall, err := h.llm.PrepareCall(h.instance.ctx, request)
	if err != nil {
		return fmt.Errorf("failed to prepare call: %w", err)
	}
	
	// Stream response
	stream, err := preparedCall.Request.getAdapter(h.llm).Stream(h.instance.ctx, preparedCall.Request)
	if err != nil {
		return fmt.Errorf("failed to stream response: %w", err)
	}
	defer stream.Close()
	
	// Process streaming response
	var responseContent string
	for {
		chunk, ok := stream.Next()
		if !ok {
			break
		}
		responseContent += chunk.Content
	}
	
	if err := stream.Err(); err != nil {
		return fmt.Errorf("streaming error: %w", err)
	}
	
	// Parse response for tool calls (simplified)
	// In full implementation, this would parse structured tool calls
	
	// Append assistant message to session
	if h.session != nil {
		event := SessionEvent{
			Type:      "assistant/message",
			Timestamp: currentTimeMillis(),
			Payload: map[string]string{
				"content": responseContent,
			},
		}
		if err := h.session.Append(event); err != nil {
			return fmt.Errorf("failed to append message: %w", err)
		}
	}
	
	return nil
}

// deriveMessages reconstructs the message history from session events
func (h *agentHandleImpl) deriveMessages() ([]Message, error) {
	if h.session == nil {
		return make([]Message, 0), nil
	}
	
	messages := make([]Message, 0)
	iter := h.session.Events()
	defer iter.Close()
	
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		
		switch event.Type {
		case "user/message":
			if payload, ok := event.Payload.(map[string]interface{}); ok {
				if content, ok := payload["content"].(string); ok {
					messages = append(messages, Message{
						Role:    "user",
						Content: content,
					})
				}
			}
		case "assistant/message":
			if payload, ok := event.Payload.(map[string]interface{}); ok {
				if content, ok := payload["content"].(string); ok {
					messages = append(messages, Message{
						Role:    "assistant",
						Content: content,
					})
				}
			}
		}
	}
	
	return messages, nil
}

// getAdapter retrieves the appropriate LLM adapter
func (req *LLMRequest) getAdapter(llm LLMService) LLMAdapter {
	adapter, _ := llm.GetAdapter(req.Provider)
	return adapter
}

// currentTimeMillis returns current time in milliseconds
func currentTimeMillis() int64 {
	return 0 // Placeholder - use time.Now().UnixMilli() in production
}
