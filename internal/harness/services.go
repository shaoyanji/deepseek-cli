package harness

// Service interfaces define the capability seams in the harness
// Each seam has three roles: Service Definition, Service Provider, and Consumer

// SessionService manages durable session event logs
type SessionService interface {
	// Create creates a new session with the given header
	Create(header SessionHeader) (SessionHandle, error)
	
	// Open opens an existing session for reading or writing
	Open(id string, mode SessionMode) (SessionHandle, error)
	
	// Stat returns metadata about a session without loading events
	Stat(id string) (SessionStat, error)
	
	// List returns all sessions matching optional filters
	List(filters SessionFilters) ([]SessionStat, error)
}

// SessionHandle provides access to a session's event log
type SessionHandle interface {
	// ID returns the session identifier
	ID() string
	
	// Append appends events to the session log
	Append(events ...SessionEvent) error
	
	// Events returns an iterator over session events
	Events() EventIterator
	
	// Close closes the session handle
	Close() error
}

// SessionEvent represents a durable fact appended to a session log
type SessionEvent struct {
	Type      string
	Timestamp int64
	Payload   interface{}
}

// SessionHeader contains session metadata
type SessionHeader struct {
	ID        string
	CreatedAt int64
	Model     string
	Provider  string
	Meta      map[string]interface{}
}

// SessionStat returns session metadata
type SessionStat struct {
	ID        string
	CreatedAt int64
	UpdatedAt int64
	Size      int64
}

// SessionMode defines access mode for opening sessions
type SessionMode string

const (
	SessionRead  SessionMode = "read"
	SessionWrite SessionMode = "write"
)

// SessionFilters defines filters for listing sessions
type SessionFilters struct {
	Limit  int
	Offset int
}

// EventIterator iterates over session events
type EventIterator interface {
	Next() (SessionEvent, bool)
	Close() error
}

// SystemPromptService assembles system prompts from sections and tool schemas
type SystemPromptService interface {
	// Assemble builds the complete system prompt
	Assemble(ctx *Context, sections []PromptSection) (string, error)
	
	// AddSection registers a prompt section provider
	AddSection(id string, provider PromptSectionProvider) error
	
	// RemoveSection removes a prompt section provider
	RemoveSection(id string) error
}

// PromptSection represents a section of the system prompt
type PromptSection struct {
	ID       string
	Content  string
	Priority int // Lower numbers are rendered first
}

// PromptSectionProvider dynamically generates prompt content
type PromptSectionProvider func(ctx *Context) (string, error)

// ToolRegistry manages model-facing tools
type ToolRegistry interface {
	// Register adds a tool to the registry
	Register(tool ToolDefinition) error
	
	// Unregister removes a tool from the registry
	Unregister(name string) error
	
	// Get retrieves a tool by name
	Get(name string) (ToolDefinition, error)
	
	// List returns all registered tools
	List() []ToolDefinition
	
	// Execute runs a tool with the given arguments
	Execute(ctx *Context, name string, args map[string]interface{}) (ToolResult, error)
}

// ToolDefinition defines a tool's schema and implementation
type ToolDefinition struct {
	Name        string
	Description string
	Parameters  interface{} // JSON Schema for parameters
	Handler     ToolHandler
}

// ToolHandler executes a tool call
type ToolHandler func(ctx *Context, args map[string]interface{}) (ToolResult, error)

// ToolResult is the outcome of a tool execution
type ToolResult struct {
	Content string
	Error   error
}

// AgentRegistry manages agent instances
type AgentRegistry interface {
	// Create creates a new agent instance
	Create(options AgentOptions) (AgentHandle, error)
	
	// Resume resumes a persisted agent session
	Resume(sessionID string) (AgentHandle, error)
	
	// Get retrieves an agent by ID
	Get(id string) (AgentHandle, error)
	
	// List returns all active agents
	List() []AgentHandle
	
	// Dispose tears down an agent
	Dispose(id string) error
}

// AgentOptions configures agent creation
type AgentOptions struct {
	ID             string
	SessionID      string
	Provider       string
	Model          string
	ReasoningEffort string
	CWD            string
	ParentAgent    AgentHandle
}

// AgentHandle provides control over an agent instance
type AgentHandle interface {
	// ID returns the agent identifier
	ID() string
	
	// SessionID returns the associated session ID
	SessionID() string
	
	// Cancel requests cancellation of current activity
	Cancel(keepInbox bool) error
	
	// IsRunning returns whether the agent is actively processing
	IsRunning() bool
	
	// Context returns the agent's scoped context
	Context() *Context
}

// LLMService provides language model adapters
type LLMService interface {
	// RegisterAdapter registers an LLM adapter for a provider
	RegisterAdapter(provider string, adapter LLMAdapter) error
	
	// GetAdapter retrieves an adapter for a provider
	GetAdapter(provider string) (LLMAdapter, error)
	
	// PrepareCall prepares a model call with resolved defaults
	PrepareCall(ctx *Context, request LLMRequest) (PreparedCall, error)
}

// LLMAdapter adapts a specific LLM provider to the harness interface
type LLMAdapter interface {
	// Stream performs a streaming chat completion
	Stream(ctx *Context, request LLMRequest) (StreamResponse, error)
}

// LLMRequest defines a model request
type LLMRequest struct {
	Provider       string
	Model          string
	Messages       []Message
	Tools          []ToolDefinition
	MaxTokens      int
	Temperature    float64
	ReasoningEffort string
}

// Message represents a conversation message
type Message struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content,omitempty"` // string or []ContentPart
}

// ContentPart represents a part of a message content
type ContentPart struct {
	Type     string `json:"type"` // "text" or "image_url"
	Text     string `json:"text,omitempty"`
	ImageURL string `json:"image_url,omitempty"`
}

// PreparedCall is a validated and ready-to-execute model call
type PreparedCall struct {
	Request           LLMRequest
	SystemPromptUpdate string // "in-history" or empty
}

// StreamResponse provides streaming model responses
type StreamResponse interface {
	// Next returns the next chunk or false when done
	Next() (StreamChunk, bool)
	
	// Err returns any error that occurred during streaming
	Err() error
	
	// Close closes the stream
	Close() error
}

// StreamChunk represents a streaming response chunk
type StreamChunk struct {
	Content      string
	FinishReason string
	Usage        Usage
}

// Usage tracks token usage
type Usage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

// FSService provides filesystem capabilities
type FSService interface {
	ReadFile(path string) ([]byte, error)
	WriteFile(path string, data []byte) error
	ReadDir(path string) ([]DirEntry, error)
	Stat(path string) (FileInfo, error)
}

// DirEntry represents a directory entry
type DirEntry struct {
	Name  string
	IsDir bool
}

// FileInfo represents file metadata
type FileInfo struct {
	Name    string
	Size    int64
	IsDir   bool
	ModTime int64
}

// ShellService provides shell execution capabilities
type ShellService interface {
	Execute(ctx *Context, command string, options ShellOptions) (ShellResult, error)
}

// ShellOptions configures shell execution
type ShellOptions struct {
	CWD       string
	Env       map[string]string
	Timeout   int64
	Sandboxed bool
}

// ShellResult is the outcome of shell execution
type ShellResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// SubagentService manages subagent delegation
type SubagentService interface {
	// Delegate creates a subagent for delegated work
	Delegate(ctx *Context, goal string, options SubagentOptions) (SubagentHandle, error)
}

// SubagentOptions configures subagent delegation
type SubagentOptions struct {
	Model       string
	MaxTurns    int
	Tools       []string
	Visibility  string // "transparent" or "opaque"
}

// SubagentHandle controls a delegated subagent
type SubagentHandle interface {
	Result() (string, error)
	Cancel() error
}

// SandboxService provides process confinement
type SandboxService interface {
	// Wrap wraps a command for sandboxed execution
	Wrap(command []string, options SandboxOptions) ([]string, error)
}

// SandboxOptions configures sandboxing
type SandboxOptions struct {
	NetworkDisabled bool
	WritableDirs    []string
	ReadOnlyDirs    []string
}

// CommandRegistry manages human commands (non-model)
type CommandRegistry interface {
	Register(cmd CommandDefinition) error
	Execute(ctx *Context, name string, args []string) error
}

// CommandDefinition defines a human-facing command
type CommandDefinition struct {
	Name        string
	Description string
	Handler     func(ctx *Context, args []string) error
}

// JobService manages background jobs
type JobService interface {
	Start(ctx *Context, fn func() error, options JobOptions) (JobHandle, error)
	Get(id string) (JobHandle, error)
	Cancel(id string) error
}

// JobOptions configures background jobs
type JobOptions struct {
	Name string
}

// JobHandle controls a background job
type JobHandle interface {
	ID() string
	Status() JobStatus
	Result() (interface{}, error)
	Cancel() error
}

// JobStatus represents job state
type JobStatus string

const (
	JobRunning   JobStatus = "running"
	JobCompleted JobStatus = "completed"
	JobFailed    JobStatus = "failed"
	JobCancelled JobStatus = "cancelled"
)

// WebhookRuntime handles authenticated webhook delivery
type WebhookRuntime interface {
	RegisterRule(rule WebhookRule) error
	Dispatch(ctx *Context, event WebhookEvent) error
}

// WebhookRule defines a trusted webhook rule
type WebhookRule struct {
	ID          string
	Pattern     string
	Handler     func(ctx *Context, event WebhookEvent) error
	WorkspaceID string
}

// WebhookEvent represents an external webhook event
type WebhookEvent struct {
	ID        string
	RuleID    string
	Payload   interface{}
	Timestamp int64
}

// SessionProjection folds session events into derived state
type SessionProjection interface {
	// Key returns the projection's unique key
	Key() string
	
	// Process processes a session event
	Process(event SessionEvent) error
	
	// State returns the current projected state
	State() interface{}
}
