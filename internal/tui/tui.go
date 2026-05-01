package tui

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/glamour"
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
	Role       string
	Content    string
	Timestamp  time.Time
}

// TokenUsage represents token usage information
type TokenUsage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

// Model is the Bubble Tea model for the TUI
type Model struct {
	State       State
	Messages    []Message
	Width       int
	Height      int
	Model       string
	TokenUsage  *TokenUsage
	ToolOutput  string
	SessionPath string
	MaxMessages int
	Styles      Styles
	Viewport    viewport.Model
	TextInput   textarea.Model
	GlamourRenderer *glamour.TermRenderer
	Err         error
}

// Styles holds all the lipgloss styles
type Styles struct {
	ChatMessage     lipgloss.Style
	UserMessage    lipgloss.Style
	AssistantMessage lipgloss.Style
	StatusBar      lipgloss.Style
	Input          lipgloss.Style
	ToolOutput     lipgloss.Style
}

// InitialModel returns a new initial model
func InitialModel() Model {
	// Initialize viewport
	vp := viewport.New(80, 20)
	vp.GotoTop()

	// Initialize textarea for multiline input
	ti := textarea.New()
	ti.Placeholder = "Type your message..."
	ti.Focus()

	// Initialize glamour renderer for markdown rendering
	gr, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(80),
	)

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
		GlamourRenderer: gr,
		Err:         err,
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
		m.Viewport.Width = m.Width
		m.Viewport.Height = m.Height - 6 // Leave room for input and status bar
		// Update textarea size
		m.TextInput.SetWidth(m.Width)
		m.TextInput.SetHeight(3)
		return m, nil

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		case tea.KeyEnter:
			// For now, always send message on Enter
			// TODO: Add proper multi-line support with modifier keys
			if m.TextInput.Value() != "" {
				m.AddMessage("user", m.TextInput.Value())
				m.TextInput.Reset()
				m.TextInput.Placeholder = "Type your message..."
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
	data, err := json.Marshal(m.Messages)
	if err != nil {
		return err
	}
	return os.WriteFile(m.SessionPath, data, 0644)
}

// LoadSession loads the session from disk
func (m *Model) LoadSession() error {
	data, err := os.ReadFile(m.SessionPath)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &m.Messages)
}

// DefaultStyles returns the default styles
func DefaultStyles() Styles {
	return Styles{
		ChatMessage:     lipgloss.NewStyle(),
		UserMessage:     lipgloss.NewStyle().Foreground(lipgloss.Color("12")),
		AssistantMessage: lipgloss.NewStyle().Foreground(lipgloss.Color("10")),
		StatusBar:       lipgloss.NewStyle().Background(lipgloss.Color("8")).Foreground(lipgloss.Color("15")),
		Input:           lipgloss.NewStyle().BorderForeground(lipgloss.Color("63")),
		ToolOutput:      lipgloss.NewStyle().Foreground(lipgloss.Color("11")),
	}
}

// renderChatView renders the chat messages
func (m Model) renderChatView() string {
	var sb strings.Builder
	
	for _, msg := range m.Messages {
		prefix := "You: "
		style := m.Styles.UserMessage
		
		if msg.Role == "assistant" {
			prefix = "Assistant: "
			style = m.Styles.AssistantMessage
		}
		
		// Apply role-specific styling to prefix
		sb.WriteString(style.Render(prefix))
		
		// Render content with glamour for markdown/code highlighting
		if m.GlamourRenderer != nil {
			rendered, err := m.GlamourRenderer.Render(msg.Content)
			if err != nil {
				// Fallback to plain text if rendering fails
				sb.WriteString(msg.Content)
			} else {
				sb.WriteString(rendered)
			}
		} else {
			sb.WriteString(msg.Content)
		}
		
		sb.WriteString("\n\n")
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
	sb.WriteString("]")
	return sb.String()
}

// renderToolOutput renders the tool output panel
func (m Model) renderToolOutput() string {
	return m.ToolOutput
}

// UpdateViewport updates the viewport size
func (m *Model) UpdateViewport() {
	// Calculate available height for viewport (total height - input area - status bar - padding)
	viewportHeight := m.Height - 6 // 3 lines for input, 1 for status bar, 2 for padding
	if viewportHeight < 5 {
		viewportHeight = 5 // Minimum height
	}
	
	m.Viewport.Width = m.Width
	m.Viewport.Height = viewportHeight
}
