package tui

import (
	"fmt"
	"strings"
	"time"

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
)

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

// Model is the Bubble Tea model for the TUI
type Model struct {
	State           State
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
	ti.Placeholder = "Type your message... (Enter to send, Ctrl+C to quit)"
	ti.Focus()
	ti.SetWidth(76)
	ti.SetHeight(3)

	// Initialize thinking viewport
	tv := viewport.New(80, 10)
	tv.GotoTop()

	return Model{
		State:       StateChat,
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
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		case tea.KeyEnter:
			// Send message on Enter if not empty
			if m.TextInput.Value() != "" && !m.Streaming {
				m.AddMessage("user", m.TextInput.Value())
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
	// Implementation for session saving
	// For now, just return nil as we're not persisting sessions yet
	return nil
}

// LoadSession loads the session from disk
func (m *Model) LoadSession() error {
	// Implementation for session loading
	// For now, just return nil as we're not persisting sessions yet
	return nil
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

// UpdateViewport updates all viewport sizes based on window dimensions
func (m *Model) UpdateViewport() {
	// Calculate available height for main viewport
	// Reserve space for: input (4), status bar (1), thinking panel (variable), cost panel (6), padding (2)
	thinkingHeight := 0
	if m.Thinking.Active || m.Thinking.Content != "" {
		thinkingHeight = 8
	}
	
	viewportHeight := m.Height - 4 - 1 - 6 - 2 - thinkingHeight
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
