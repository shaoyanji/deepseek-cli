// Package tui provides the terminal user interface for DeepSeek CLI.
package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"deepseek-cli/internal/config"
	"deepseek-cli/internal/hooks"
	"deepseek-cli/internal/lsp"
	"deepseek-cli/internal/mcp"
	"deepseek-cli/internal/subagent"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// State represents the TUI state
type State string

const (
	StateChat      State = "chat"
	StateTools     State = "tools"
	StateInput     State = "input"
	StateThinking  State = "thinking"
	StateConfirm   State = "confirm"
)

// CommandMode represents the current input mode
type CommandMode string

const (
	ModeNormal  CommandMode = "normal"
	ModeCommand CommandMode = "command"
)

// SlashCommand represents a slash command handler
type SlashCommand struct {
	Name        string
	Description string
	Handler     func(args string) (string, error)
}

// KeyBindings holds configurable key bindings
type KeyBindings struct {
	Send         tea.KeyType
	Cancel       tea.KeyType
	HistoryUp    tea.KeyType
	HistoryDown  tea.KeyType
	ClearScreen  tea.KeyType
	EnterCommand tea.KeyType
	SaveSession  tea.KeyType
}

// DefaultKeyBindings returns the default key bindings
func DefaultKeyBindings() KeyBindings {
	return KeyBindings{
		Send:         tea.KeyEnter,
		Cancel:       tea.KeyCtrlC,
		HistoryUp:    tea.KeyUp,
		HistoryDown:  tea.KeyDown,
		ClearScreen:  tea.KeyCtrlL,
		EnterCommand: tea.KeyEsc,
		SaveSession:  tea.KeyCtrlS,
	}
}

// Message represents a chat message
type Message struct {
	Role      string
	Content   string
	Timestamp time.Time
}

// TokenUsage represents token usage information
type TokenUsage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
	CostUSD          float64
}

// TurnCost tracks per-turn token usage and cost
type TurnCost struct {
	TurnID        int
	PromptTokens  int
	CompTokens    int
	TotalTokens   int
	CostUSD       float64
	Duration      time.Duration
}

// ThinkingState represents the current thinking content
type ThinkingState struct {
	Active    bool
	Content   string
	StartTime time.Time
}

// Diagnostic represents an LSP diagnostic result
type Diagnostic struct {
	Severity  int    `json:"severity"`
	Line      int    `json:"line"`
	Column    int    `json:"column"`
	EndLine   int    `json:"endLine"`
	EndColumn int    `json:"endColumn"`
	Message   string `json:"message"`
	Source    string `json:"source"`
	Code      string `json:"code,omitempty"`
}

// LintPanel represents the diagnostics panel state
type LintPanel struct {
	Visible     bool
	Diagnostics []Diagnostic
	FilePath    string
	LastRun     time.Time
	Error       error
	mu          sync.Mutex
}

// SubAgentTask represents a sub-agent task status
type SubAgentTask struct {
	ID        int
	Prompt    string
	Status    string
	Result    string
	StartTime time.Time
	EndTime   time.Time
}

// Model is the Bubble Tea model for the TUI
type Model struct {
	State           State
	CommandMode     CommandMode
	Messages        []Message
	Width           int
	Height          int
	Model           string
	TokenUsage      *TokenUsage
	TurnCosts       []TurnCost
	CurrentTurn     int
	ToolOutput      string
	SessionPath     string
	MaxMessages     int
	Styles          Styles
	Viewport        viewport.Model
	TextInput       textarea.Model
	ThinkingViewport viewport.Model
	Err             error
	Streaming       bool
	StreamContent   string
	Thinking        ThinkingState
	Mode            string
	WorkspacePath   string
	KeyBindings     KeyBindings
	SlashCommands   map[string]*SlashCommand
	CommandHistory  []string
	HistoryIndex    int
	LastKeyMsg      tea.KeyMsg
	StatusMessage   string
	StatusTime      time.Time
	
	// Phase 1: LSP Diagnostics
	LSPClient       *lsp.Client
	LSPHook         *hooks.LSPHook
	LintPanel       LintPanel
	Config          *config.Config
	
	// Phase 2: Sub-agent orchestration
	SubAgentManager *subagent.Manager
	ActiveSubAgents []SubAgentTask
	
	// Phase 3: MCP integration
	MCPClient       *mcp.Client
}

// Styles holds all the lipgloss styles
type Styles struct {
	ChatMessage      lipgloss.Style
	UserMessage      lipgloss.Style
	AssistantMessage lipgloss.Style
	StatusBar        lipgloss.Style
	Input            lipgloss.Style
	ToolOutput       lipgloss.Style
	Welcome          lipgloss.Style
	Error            lipgloss.Style
	Thinking         lipgloss.Style
	ThinkingHeader   lipgloss.Style
	CostHighlight    lipgloss.Style
	ModeIndicator    lipgloss.Style
}

// InitialModel returns a new initial model
func InitialModel() Model {
	// Initialize viewport
	vp := viewport.New(80, 20)
	vp.GotoTop()

	// Initialize textarea for multiline input
	ti := textarea.New()
	ti.Placeholder = "Type your message... (Enter to send, Ctrl+C to quit, / for commands)"
	ti.Focus()
	ti.SetWidth(76)
	ti.SetHeight(3)

	// Initialize thinking viewport
	tv := viewport.New(80, 10)
	tv.GotoTop()

	// Create slash commands
	slashCommands := make(map[string]*SlashCommand)

	m := Model{
		State:       StateChat,
		CommandMode: ModeNormal,
		Messages:    make([]Message, 0),
		Width:       80,
		Height:      24,
		Model:       "deepseek-v4-pro",
		MaxMessages: 100,
		Styles:      DefaultStyles(),
		Viewport:    vp,
		TextInput:   ti,
		ThinkingViewport: tv,
		Streaming:   false,
		Mode:        "agent",
		KeyBindings: DefaultKeyBindings(),
		SlashCommands: slashCommands,
		CommandHistory: make([]string, 0),
		HistoryIndex: -1,
		LintPanel: LintPanel{
			Visible:     false,
			Diagnostics: make([]Diagnostic, 0),
		},
		ActiveSubAgents: make([]SubAgentTask, 0),
	}

	// Load configuration
	cfg, err := config.Load()
	if err == nil {
		m.Config = cfg
		
		// Initialize LSP client if enabled
		if cfg.LSP.Enabled && len(cfg.LSP.Servers) > 0 {
			lspConfigs := make(map[string]lsp.ServerConfig)
			for langID, srv := range cfg.LSP.Servers {
				lspConfigs[langID] = lsp.ServerConfig{
					Command: srv.Command,
					Args:    srv.Args,
					RootURI: srv.RootURI,
				}
			}
			timeout := time.Duration(cfg.LSP.Timeout) * time.Second
			if timeout == 0 {
				timeout = 5 * time.Second
			}
			m.LSPClient = lsp.NewClient(lspConfigs, timeout)
			m.LSPHook = hooks.NewLSPHook(m.LSPClient)
		}
		
		// Initialize sub-agent manager
		m.SubAgentManager = subagent.NewManager(3, nil) // Default executor
		
		// Initialize MCP client if configured
		// Note: MCP server configs would be loaded from cfg.MCP.Servers when added
	}

	// Register built-in slash commands
	m.registerSlashCommands()

	return m
}

// registerSlashCommands registers all built-in slash commands
func (m *Model) registerSlashCommands() {
	m.SlashCommands["agent"] = &SlashCommand{
		Name:        "agent",
		Description: "Switch to Agent mode (asks for confirmation before running tools)",
		Handler: func(args string) (string, error) {
			m.Mode = "agent"
			return "Switched to Agent mode - will ask for confirmation before running tools", nil
		},
	}
	m.SlashCommands["yolo"] = &SlashCommand{
		Name:        "yolo",
		Description: "Switch to YOLO mode (auto-approve all tools)",
		Handler: func(args string) (string, error) {
			m.Mode = "yolo"
			return "Switched to YOLO mode - tools will run automatically", nil
		},
	}
	m.SlashCommands["acme"] = &SlashCommand{
		Name:        "acme",
		Description: "Switch to Acme mode (read-only, no tool execution)",
		Handler: func(args string) (string, error) {
			m.Mode = "acme"
			return "Switched to Acme (Plan) mode - read-only operations only", nil
		},
	}
	m.SlashCommands["clear"] = &SlashCommand{
		Name:        "clear",
		Description: "Clear the conversation",
		Handler: func(args string) (string, error) {
			m.Messages = make([]Message, 0)
			m.Viewport.GotoTop()
			return "Conversation cleared", nil
		},
	}
	m.SlashCommands["help"] = &SlashCommand{
		Name:        "help",
		Description: "Show available commands",
		Handler: func(args string) (string, error) {
			var sb strings.Builder
			sb.WriteString("Available slash commands:\n\n")
			for _, cmd := range m.SlashCommands {
				sb.WriteString(fmt.Sprintf("  /%s - %s\n", cmd.Name, cmd.Description))
			}
			sb.WriteString("\nKeyboard shortcuts:\n")
			sb.WriteString("  Enter       - Send message\n")
			sb.WriteString("  Ctrl+C      - Quit\n")
			sb.WriteString("  Ctrl+S      - Save session\n")
			sb.WriteString("  Ctrl+L      - Clear screen\n")
			sb.WriteString("  Esc         - Enter command mode (/)\n")
			sb.WriteString("  Up/Down     - Command history\n")
			return sb.String(), nil
		},
	}
	m.SlashCommands["exit"] = &SlashCommand{
		Name:        "exit",
		Description: "Quit the application",
		Handler: func(args string) (string, error) {
			return "", nil
		},
	}
	m.SlashCommands["save"] = &SlashCommand{
		Name:        "save",
		Description: "Save the current session",
		Handler: func(args string) (string, error) {
			if err := m.SaveSession(); err != nil {
				return "", fmt.Errorf("failed to save session: %w", err)
			}
			return "Session saved successfully", nil
		},
	}
	m.SlashCommands["restore"] = &SlashCommand{
		Name:        "restore",
		Description: "Restore last saved session",
		Handler: func(args string) (string, error) {
			if err := m.LoadSession(); err != nil {
				return "", fmt.Errorf("failed to restore session: %w", err)
			}
			return "Session restored successfully", nil
		},
	}
	m.SlashCommands["file"] = &SlashCommand{
		Name:        "file",
		Description: "Read a file and insert its content into the conversation",
		Handler: func(args string) (string, error) {
			if args == "" {
				return "", fmt.Errorf("usage: /file <path>")
			}
			content, err := os.ReadFile(args)
			if err != nil {
				return "", fmt.Errorf("failed to read file: %w", err)
			}
			m.TextInput.SetValue(m.TextInput.Value() + "\n```\n" + string(content) + "\n```")
			return fmt.Sprintf("File content inserted: %s", args), nil
		},
	}
	m.SlashCommands["shell"] = &SlashCommand{
		Name:        "shell",
		Description: "Run a shell command and capture output",
		Handler: func(args string) (string, error) {
			if args == "" {
				return "", fmt.Errorf("usage: /shell <command>")
			}
			// For now, just show a placeholder - actual implementation in exec package
			return fmt.Sprintf("Shell command would execute: %s", args), nil
		},
	}
	m.SlashCommands["web"] = &SlashCommand{
		Name:        "web",
		Description: "Search the web (using DuckDuckGo/browserless API)",
		Handler: func(args string) (string, error) {
			if args == "" {
				return "", fmt.Errorf("usage: /web <query>")
			}
			// Placeholder - actual implementation would call web search API
			return fmt.Sprintf("Web search would be performed for: %s", args), nil
		},
	}
}

// Init implements tea.Model
func (m Model) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	// Handle errors
	if m.Err != nil {
		return m, tea.Quit
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		// Update viewport size
		m.Viewport.Width = m.Width - 4
		m.Viewport.Height = m.Height - 8 // Leave room for input and status bar
		// Update textarea size
		m.TextInput.SetWidth(m.Width - 4)
		return m, nil

	case tea.KeyMsg:
		m.LastKeyMsg = msg
		
		// Check for configured key bindings
		switch msg.Type {
		case m.KeyBindings.Cancel:
			// Cancel streaming or quit
			if m.Streaming {
				m.Streaming = false
				m.setStatusMessage("Streaming cancelled")
				return m, nil
			}
			return m, tea.Quit
			
		case m.KeyBindings.SaveSession:
			// Save session on Ctrl+S
			if err := m.SaveSession(); err != nil {
				m.setStatusMessage(fmt.Sprintf("Save failed: %v", err))
			} else {
				m.setStatusMessage("Session saved")
			}
			return m, nil
			
		case m.KeyBindings.ClearScreen:
			// Clear screen on Ctrl+L
			m.Viewport.GotoTop()
			m.setStatusMessage("Screen cleared")
			return m, nil
			
		case m.KeyBindings.EnterCommand:
			// Enter command mode on Esc - prefix with /
			if !strings.HasPrefix(m.TextInput.Value(), "/") {
				m.TextInput.SetValue("/" + m.TextInput.Value())
			}
			return m, nil
			
		case m.KeyBindings.HistoryUp:
			// Navigate command history up
			if len(m.CommandHistory) > 0 && m.HistoryIndex < len(m.CommandHistory)-1 {
				m.HistoryIndex++
				m.TextInput.SetValue(m.CommandHistory[len(m.CommandHistory)-1-m.HistoryIndex])
			}
			return m, nil
			
		case m.KeyBindings.HistoryDown:
			// Navigate command history down
			if m.HistoryIndex > 0 {
				m.HistoryIndex--
				m.TextInput.SetValue(m.CommandHistory[len(m.CommandHistory)-1-m.HistoryIndex])
			} else if m.HistoryIndex == 0 {
				m.HistoryIndex = -1
				m.TextInput.SetValue("")
			}
			return m, nil
			
		case m.KeyBindings.Send:
			// Send message on Enter if not empty
			if m.TextInput.Value() != "" && !m.Streaming {
				input := m.TextInput.Value()
				
				// Check if it's a slash command
				if strings.HasPrefix(input, "/") {
					result, quitCmd := m.executeSlashCommand(input)
					m.TextInput.Reset()
					if result != "" {
						m.AddMessage("system", result)
					}
					if quitCmd != nil {
						return m, quitCmd
					}
				} else {
					// Regular message
					m.AddMessage("user", input)
					// Add to command history
					m.CommandHistory = append(m.CommandHistory, input)
					m.HistoryIndex = -1
				}
				m.TextInput.Reset()
				m.TextInput.Placeholder = "Type your message... (Enter to send, Ctrl+C to quit)"
				// Auto-scroll to bottom
				m.Viewport.GotoBottom()
			}
			return m, nil
			
		default:
			// Update textarea
			m.TextInput, cmd = m.TextInput.Update(msg)
			return m, cmd
		}
	}

	// Update viewport
	m.Viewport, cmd = m.Viewport.Update(msg)
	return m, cmd
}

// executeSlashCommand parses and executes a slash command
func (m *Model) executeSlashCommand(input string) (string, tea.Cmd) {
	// Remove leading slash and split into command and args
	input = strings.TrimPrefix(input, "/")
	parts := strings.SplitN(input, " ", 2)
	cmdName := parts[0]
	args := ""
	if len(parts) > 1 {
		args = parts[1]
	}

	// Look up command
	cmd, exists := m.SlashCommands[cmdName]
	if !exists {
		return fmt.Sprintf("Command not found: /%s. Type /help for available commands.", cmdName), nil
	}

	// Execute command handler
	result, err := cmd.Handler(args)
	if err != nil {
		return fmt.Sprintf("Error: %v", err), nil
	}

	// Special case for /exit which should quit
	if cmdName == "exit" {
		return "", tea.Quit
	}

	return result, nil
}

// setStatusMessage sets a temporary status message
func (m *Model) setStatusMessage(msg string) {
	m.StatusMessage = msg
	m.StatusTime = time.Now()
}

// getAndClearStatusMessage returns and clears the status message if it's old
func (m *Model) getAndClearStatusMessage() string {
	if m.StatusMessage == "" {
		return ""
	}
	if time.Since(m.StatusTime) > 3*time.Second {
		msg := m.StatusMessage
		m.StatusMessage = ""
		return msg
	}
	return m.StatusMessage
}

// AddMessage adds a message to the chat
func (m *Model) AddMessage(role, content string) {
	m.Messages = append(m.Messages, Message{
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
	})

	// Trim if over max
	if m.MaxMessages > 0 && len(m.Messages) > m.MaxMessages {
		m.Messages = m.Messages[len(m.Messages)-m.MaxMessages:]
	}
}

// SaveSession saves the session to disk
func (m *Model) SaveSession() error {
	if m.SessionPath == "" {
		// Default session path
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		sessionDir := filepath.Join(home, ".local", "share", "deepseek-cli", "sessions")
		if err := os.MkdirAll(sessionDir, 0755); err != nil {
			return fmt.Errorf("failed to create session directory: %w", err)
		}
		m.SessionPath = filepath.Join(sessionDir, "last_session.json")
	}

	// Create session data structure
	sessionData := map[string]interface{}{
		"messages":    m.Messages,
		"mode":        m.Mode,
		"model":       m.Model,
		"timestamp":   time.Now(),
		"tokenUsage":  m.TokenUsage,
		"turnCosts":   m.TurnCosts,
	}

	data, err := json.MarshalIndent(sessionData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	if err := os.WriteFile(m.SessionPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write session file: %w", err)
	}

	return nil
}

// LoadSession loads the session from disk
func (m *Model) LoadSession() error {
	if m.SessionPath == "" {
		// Default session path
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		m.SessionPath = filepath.Join(home, ".local", "share", "deepseek-cli", "sessions", "last_session.json")
	}

	// Check if session file exists
	if _, err := os.Stat(m.SessionPath); os.IsNotExist(err) {
		return fmt.Errorf("no saved session found")
	}

	data, err := os.ReadFile(m.SessionPath)
	if err != nil {
		return fmt.Errorf("failed to read session file: %w", err)
	}

	var sessionData map[string]interface{}
	if err := json.Unmarshal(data, &sessionData); err != nil {
		return fmt.Errorf("failed to unmarshal session: %w", err)
	}

	// Restore messages
	if msgs, ok := sessionData["messages"].([]interface{}); ok {
		m.Messages = make([]Message, 0)
		for _, msg := range msgs {
			if msgMap, ok := msg.(map[string]interface{}); ok {
				m.Messages = append(m.Messages, Message{
					Role:    getString(msgMap, "role"),
					Content: getString(msgMap, "content"),
				})
			}
		}
	}

	// Restore mode
	if mode, ok := sessionData["mode"].(string); ok {
		m.Mode = mode
	}

	// Restore model
	if model, ok := sessionData["model"].(string); ok {
		m.Model = model
	}

	m.Viewport.GotoBottom()
	return nil
}

// getString safely extracts a string from an interface{} map
func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

// DefaultStyles returns the default styles with DeepSeek-blue theme
func DefaultStyles() Styles {
	return Styles{
		ChatMessage:      lipgloss.NewStyle(),
		UserMessage:      lipgloss.NewStyle().Foreground(lipgloss.Color("#60A5FA")), // Light blue
		AssistantMessage: lipgloss.NewStyle().Foreground(lipgloss.Color("#4ADE80")), // Green
		StatusBar:        lipgloss.NewStyle().Background(lipgloss.Color("#1E3A5F")).Foreground(lipgloss.Color("#93C5FD")), // DeepSeek blue theme
		Input:            lipgloss.NewStyle().BorderForeground(lipgloss.Color("#60A5FA")),
		ToolOutput:       lipgloss.NewStyle().Foreground(lipgloss.Color("#FCD34D")), // Amber
		Welcome:          lipgloss.NewStyle().Foreground(lipgloss.Color("#60A5FA")).Bold(true),
		Error:            lipgloss.NewStyle().Foreground(lipgloss.Color("#EF4444")), // Red
		Thinking:         lipgloss.NewStyle().Foreground(lipgloss.Color("#A78BFA")), // Purple for thinking content
		ThinkingHeader:   lipgloss.NewStyle().Foreground(lipgloss.Color("#C084FC")).Bold(true), // Bright purple header
		CostHighlight:    lipgloss.NewStyle().Foreground(lipgloss.Color("#34D399")).Bold(true), // Green for costs
		ModeIndicator:    lipgloss.NewStyle().Foreground(lipgloss.Color("#F472B6")).Bold(true), // Pink for mode
	}
}

// renderChatView renders the chat messages
func (m Model) renderChatView() string {
	var sb strings.Builder

	// Welcome message if no messages
	if len(m.Messages) == 0 {
		welcomeText := fmt.Sprintf("Welcome to DeepSeek CLI TUI\n\nMode: %s | Model: %s\n\nStart chatting by typing a message below.\nPress Ctrl+C to exit.", m.Mode, m.Model)
		sb.WriteString(m.Styles.Welcome.Render(welcomeText))
		sb.WriteString("\n\n")
	} else {
		for _, msg := range m.Messages {
			prefix := "You: "
			style := m.Styles.UserMessage

			if msg.Role == "assistant" {
				prefix = "Assistant: "
				style = m.Styles.AssistantMessage
			}

			// Apply role-specific styling to prefix
			sb.WriteString(style.Render(prefix))
			sb.WriteString(msg.Content)
			sb.WriteString("\n\n")
		}
	}

	// Set the viewport content
	m.Viewport.SetContent(sb.String())

	return m.Viewport.View()
}

// renderThinkingView renders the thinking/reasoning panel
func (m Model) renderThinkingView() string {
	if !m.Thinking.Active && m.Thinking.Content == "" {
		return ""
	}

	var sb strings.Builder
	
	// Header with thinking indicator and duration
	header := "🧠 Thinking"
	if m.Thinking.Active {
		header += " (processing...)"
		duration := time.Since(m.Thinking.StartTime).Round(time.Millisecond)
		header += fmt.Sprintf(" [%s]", duration.String())
	}
	
	sb.WriteString(m.Styles.ThinkingHeader.Render(header))
	sb.WriteString("\n")
	
	// Render thinking content with styling
	content := m.Thinking.Content
	if len(content) > 0 {
		sb.WriteString(m.Styles.Thinking.Render(content))
	}
	
	m.ThinkingViewport.SetContent(sb.String())
	return m.ThinkingViewport.View()
}

// renderCostPanel renders the live cost tracking panel
func (m Model) renderCostPanel() string {
	var sb strings.Builder
	
	sb.WriteString(m.Styles.CostHighlight.Render("💰 Cost Tracking"))
	sb.WriteString("\n")
	
	if m.TokenUsage != nil {
		sb.WriteString(fmt.Sprintf("Session Total: %s\n", 
			m.Styles.CostHighlight.Render(fmt.Sprintf("$%.4f USD", m.TokenUsage.CostUSD))))
		sb.WriteString(fmt.Sprintf("Tokens: %d (prompt: %d, completion: %d)\n",
			m.TokenUsage.TotalTokens, m.TokenUsage.PromptTokens, m.TokenUsage.CompletionTokens))
	}
	
	// Show last turn costs
	if len(m.TurnCosts) > 0 {
		lastTurn := m.TurnCosts[len(m.TurnCosts)-1]
		sb.WriteString(fmt.Sprintf("\nLast Turn (#%d): %s", 
			lastTurn.TurnID,
			m.Styles.CostHighlight.Render(fmt.Sprintf("$%.4f USD", lastTurn.CostUSD))))
		sb.WriteString(fmt.Sprintf(" [%d tokens, %v]", lastTurn.TotalTokens, lastTurn.Duration.Round(time.Millisecond)))
	}
	
	return sb.String()
}

// renderInputView renders the input area
func (m Model) renderInputView() string {
	return m.Styles.Input.Render(m.TextInput.View())
}

// renderStatusBar renders the status bar with mode indicator and cost info
func (m Model) renderStatusBar() string {
	var sb strings.Builder
	
	// Mode indicator
	sb.WriteString(m.Styles.ModeIndicator.Render(fmt.Sprintf("Mode: %s", m.Mode)))
	sb.WriteString(" | ")
	
	// Model name
	sb.WriteString(m.Model)
	
	// Token usage summary
	if m.TokenUsage != nil {
		sb.WriteString(fmt.Sprintf(" | Tokens: %d", m.TokenUsage.TotalTokens))
	}
	
	// Streaming indicator
	if m.Streaming {
		sb.WriteString(" | ")
		sb.WriteString(m.Styles.CostHighlight.Render("⏳ streaming..."))
	}
	
	return m.Styles.StatusBar.Render(sb.String())
}

// renderToolOutput renders the tool output panel
func (m Model) renderToolOutput() string {
	return m.ToolOutput
}

// renderLintPanel renders the LSP diagnostics panel
func (m Model) renderLintPanel() string {
	if !m.LintPanel.Visible || len(m.LintPanel.Diagnostics) == 0 {
		return ""
	}

	var sb strings.Builder
	
	// Header with file name and diagnostic count
	header := fmt.Sprintf("🔍 Diagnostics: %s (%d issues)", 
		filepath.Base(m.LintPanel.FilePath), 
		len(m.LintPanel.Diagnostics))
	sb.WriteString(m.Styles.ThinkingHeader.Render(header))
	sb.WriteString("\n\n")

	// Group diagnostics by severity
	errors := make([]Diagnostic, 0)
	warnings := make([]Diagnostic, 0)
	info := make([]Diagnostic, 0)

	for _, d := range m.LintPanel.Diagnostics {
		switch d.Severity {
		case 1:
			errors = append(errors, d)
		case 2:
			warnings = append(warnings, d)
		default:
			info = append(info, d)
		}
	}

	// Render errors in red
	if len(errors) > 0 {
		sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#EF4444")).Bold(true).Render(fmt.Sprintf("✗ Errors (%d):\n", len(errors))))
		for _, e := range errors {
			sb.WriteString(fmt.Sprintf("  Line %d: %s\n", e.Line+1, e.Message))
		}
		sb.WriteString("\n")
	}

	// Render warnings in yellow
	if len(warnings) > 0 {
		sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#FCD34D")).Bold(true).Render(fmt.Sprintf("⚠ Warnings (%d):\n", len(warnings))))
		for _, w := range warnings {
			sb.WriteString(fmt.Sprintf("  Line %d: %s\n", w.Line+1, w.Message))
		}
		sb.WriteString("\n")
	}

	// Render info in blue
	if len(info) > 0 {
		sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#60A5FA")).Bold(true).Render(fmt.Sprintf("ℹ Info (%d):\n", len(info))))
		for _, i := range info {
			sb.WriteString(fmt.Sprintf("  Line %d: %s\n", i.Line+1, i.Message))
		}
	}

	return sb.String()
}

// runDiagnostics runs LSP diagnostics on a file
func (m *Model) runDiagnostics(filePath string) {
	if m.LSPClient == nil || m.LSPHook == nil {
		m.setStatusMessage("LSP not configured")
		return
	}

	m.setStatusMessage("Running diagnostics...")
	
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		lspDiags, err := m.LSPHook.OnFileWrite(ctx, filePath)
		
		m.LintPanel.mu.Lock()
		defer m.LintPanel.mu.Unlock()

		if err != nil {
			m.LintPanel.Error = err
			m.LintPanel.Visible = true
		} else {
			// Convert lsp.Diagnostic to our Diagnostic type
			diags := make([]Diagnostic, len(lspDiags))
			for i, d := range lspDiags {
				diags[i] = Diagnostic{
					Severity:  d.Severity,
					Line:      d.Line,
					Column:    d.Column,
					EndLine:   d.EndLine,
					EndColumn: d.EndColumn,
					Message:   d.Message,
					Source:    d.Source,
					Code:      d.Code,
				}
			}
			m.LintPanel.Diagnostics = diags
			m.LintPanel.FilePath = filePath
			m.LintPanel.LastRun = time.Now()
			m.LintPanel.Error = nil
			m.LintPanel.Visible = len(diags) > 0
		}
	}()
}

// toggleLintPanel toggles the visibility of the diagnostics panel
func (m *Model) toggleLintPanel() {
	m.LintPanel.Visible = !m.LintPanel.Visible
	if m.LintPanel.Visible && len(m.LintPanel.Diagnostics) == 0 && m.LintPanel.Error == nil {
		m.setStatusMessage("No diagnostics available - edit a file first")
	}
}

// UpdateViewport updates all viewport sizes based on window dimensions
func (m *Model) UpdateViewport() {
	// Calculate available height for main viewport
	// Reserve space for: input (4), status bar (1), thinking panel (variable), cost panel (6), lint panel (variable), padding (2)
	thinkingHeight := 0
	if m.Thinking.Active || m.Thinking.Content != "" {
		thinkingHeight = 8
	}

	lintHeight := 0
	if m.LintPanel.Visible && len(m.LintPanel.Diagnostics) > 0 {
		lintHeight = 10
	}
	
	viewportHeight := m.Height - 4 - 1 - 6 - 2 - thinkingHeight - lintHeight
	if viewportHeight < 5 {
		viewportHeight = 5 // Minimum height
	}

	m.Viewport.Width = m.Width - 4
	m.Viewport.Height = viewportHeight
	
	// Update thinking viewport
	m.ThinkingViewport.Width = m.Width - 4
	m.ThinkingViewport.Height = thinkingHeight
}

// SetStreaming sets the streaming state
func (m *Model) SetStreaming(streaming bool) {
	m.Streaming = streaming
	if streaming {
		m.StreamContent = ""
	}
}

// AppendToLastMessage appends content to the last assistant message or creates a new one
func (m *Model) AppendToLastMessage(content string) {
	if len(m.Messages) > 0 && m.Messages[len(m.Messages)-1].Role == "assistant" {
		m.Messages[len(m.Messages)-1].Content += content
	} else {
		m.AddMessage("assistant", content)
	}
}

// UpdateThinking updates the thinking state with new content
func (m *Model) UpdateThinking(content string, active bool) {
	m.Thinking.Content = content
	m.Thinking.Active = active
	if active && m.Thinking.StartTime.IsZero() {
		m.Thinking.StartTime = time.Now()
	} else if !active {
		m.Thinking.StartTime = time.Time{} // Reset for next turn
	}
}

// AddTurnCost records a turn's token usage and cost
func (m *Model) AddTurnCost(turnID int, promptTokens, compTokens, totalTokens int, costUSD float64, duration time.Duration) {
	m.TurnCosts = append(m.TurnCosts, TurnCost{
		TurnID:       turnID,
		PromptTokens: promptTokens,
		CompTokens:   compTokens,
		TotalTokens:  totalTokens,
		CostUSD:      costUSD,
		Duration:     duration,
	})
}

// SetMode sets the execution mode
func (m *Model) SetMode(mode string) {
	m.Mode = mode
}

// SetWorkspacePath sets the workspace path
func (m *Model) SetWorkspacePath(path string) {
	m.WorkspacePath = path
}

// View implements tea.Model with enhanced layout including thinking and cost panels
func (m Model) View() string {
	var sb strings.Builder
	
	// Top section: Chat viewport
	sb.WriteString(m.renderChatView())
	sb.WriteString("\n")
	
	// Thinking panel (if active)
	if thinkingView := m.renderThinkingView(); thinkingView != "" {
		sb.WriteString(thinkingView)
		sb.WriteString("\n")
	}
	
	// Cost tracking panel
	sb.WriteString(m.renderCostPanel())
	sb.WriteString("\n")
	
	// Tool output (if any)
	if m.ToolOutput != "" {
		sb.WriteString(m.Styles.ToolOutput.Render("🔧 Tool Output:\n"))
		sb.WriteString(m.renderToolOutput())
		sb.WriteString("\n")
	}
	
	// Input area
	sb.WriteString(m.renderInputView())
	sb.WriteString("\n")
	
	// Status bar at bottom
	sb.WriteString(m.renderStatusBar())
	
	return sb.String()
}
