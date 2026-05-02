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
	StateChat  State = "chat"
	StateTools State = "tools"
	StateInput State = "input"
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
}

// Model is the Bubble Tea model for the TUI
type Model struct {
	State         State
	Messages      []Message
	Width         int
	Height        int
	Model         string
	TokenUsage    *TokenUsage
	ToolOutput    string
	SessionPath   string
	MaxMessages   int
	Styles        Styles
	Viewport      viewport.Model
	TextInput     textarea.Model
	Err           error
	Streaming     bool
	StreamContent string
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
		Streaming:   false,
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

// View implements tea.Model
func (m Model) View() string {
	return m.renderChatView() + "\n" + m.renderInputView() + "\n" + m.renderStatusBar()
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

// DefaultStyles returns the default styles
func DefaultStyles() Styles {
	return Styles{
		ChatMessage:      lipgloss.NewStyle(),
		UserMessage:      lipgloss.NewStyle().Foreground(lipgloss.Color("12")),
		AssistantMessage: lipgloss.NewStyle().Foreground(lipgloss.Color("10")),
		StatusBar:        lipgloss.NewStyle().Background(lipgloss.Color("236")).Foreground(lipgloss.Color("252")),
		Input:            lipgloss.NewStyle().BorderForeground(lipgloss.Color("63")),
		ToolOutput:       lipgloss.NewStyle().Foreground(lipgloss.Color("11")),
		Welcome:          lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Bold(true),
		Error:            lipgloss.NewStyle().Foreground(lipgloss.Color("9")),
	}
}

// renderChatView renders the chat messages
func (m Model) renderChatView() string {
	var sb strings.Builder

	// Welcome message if no messages
	if len(m.Messages) == 0 {
		welcomeText := "Welcome to DeepSeek CLI TUI\n\nStart chatting by typing a message below.\nPress Ctrl+C to exit."
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

// renderInputView renders the input area
func (m Model) renderInputView() string {
	return m.Styles.Input.Render(m.TextInput.View())
}

// renderStatusBar renders the status bar
func (m Model) renderStatusBar() string {
	var sb strings.Builder
	sb.WriteString("[")
	sb.WriteString(m.Model)
	if m.TokenUsage != nil {
		sb.WriteString(fmt.Sprintf(" | prompt: %d, completion: %d", m.TokenUsage.PromptTokens, m.TokenUsage.CompletionTokens))
	}
	if m.Streaming {
		sb.WriteString(" | streaming...")
	}
	sb.WriteString("]")
	return m.Styles.StatusBar.Render(sb.String())
}

// renderToolOutput renders the tool output panel
func (m Model) renderToolOutput() string {
	return m.ToolOutput
}

// UpdateViewport updates the viewport size
func (m *Model) UpdateViewport() {
	// Calculate available height for viewport (total height - input area - status bar - padding)
	viewportHeight := m.Height - 8
	if viewportHeight < 5 {
		viewportHeight = 5 // Minimum height
	}

	m.Viewport.Width = m.Width - 4
	m.Viewport.Height = viewportHeight
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
