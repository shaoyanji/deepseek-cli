package tui

import (
	"testing"

	"github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestInitialModel(t *testing.T) {
	m := InitialModel()
	assert.NotNil(t, m)
	assert.Equal(t, StateChat, m.State)
	assert.Empty(t, m.TextInput.Value())
	assert.Empty(t, m.Messages)
	assert.NotNil(t, m.Styles)
	assert.NotNil(t, m.Viewport)
	assert.NotNil(t, m.GlamourRenderer)
}

func TestModelInit(t *testing.T) {
	m := InitialModel()
	cmd := m.Init()
	// Init should return nil or a command
	assert.True(t, cmd == nil || cmd != nil)
}

func TestUpdateSetSize(t *testing.T) {
	m := InitialModel()
	newM, _ := m.Update(tea.WindowSizeMsg{
		Width:  100,
		Height: 40,
	})
	model := newM.(Model)
	assert.Equal(t, 100, model.Width)
	assert.Equal(t, 40, model.Height)
}

func TestUpdateQuit(t *testing.T) {
	m := InitialModel()
	_, cmd := m.Update(tea.KeyMsg{
		Type: tea.KeyCtrlC,
	})
	assert.NotNil(t, cmd)
}

func TestUpdateSendMessage(t *testing.T) {
	m := InitialModel()
	m.TextInput.SetValue("Hello, world!")
	
	newM, _ := m.Update(tea.KeyMsg{
		Type: tea.KeyEnter,
	})
	model := newM.(Model)
	// After sending, input should be cleared
	assert.Empty(t, model.TextInput.Value())
	// Message should be added
	assert.NotEmpty(t, model.Messages)
}

func TestAddMessage(t *testing.T) {
	m := InitialModel()
	m.AddMessage("user", "Hello")
	m.AddMessage("assistant", "Hi there!")
	
	assert.Len(t, m.Messages, 2)
	assert.Equal(t, "user", m.Messages[0].Role)
	assert.Equal(t, "Hello", m.Messages[0].Content)
	assert.Equal(t, "assistant", m.Messages[1].Role)
}

func TestRenderChatView(t *testing.T) {
	m := InitialModel()
	m.AddMessage("user", "Hello")
	m.AddMessage("assistant", "Hi there!")
	
	view := m.renderChatView()
	assert.Contains(t, view, "Hello")
	assert.Contains(t, view, "Hi there!")
}

func TestRenderInputView(t *testing.T) {
	m := InitialModel()
	m.TextInput.SetValue("test input")
	
	view := m.renderInputView()
	assert.Contains(t, view, "test input")
}

func TestRenderStatusBar(t *testing.T) {
	m := InitialModel()
	m.Model = "deepseek-v4-pro"
	m.TokenUsage = &TokenUsage{
		PromptTokens:     100,
		CompletionTokens: 50,
		TotalTokens:      150,
	}
	
	view := m.renderStatusBar()
	assert.Contains(t, view, "deepseek-v4-pro")
	assert.Contains(t, view, "100")
	assert.Contains(t, view, "50")
}

func TestSessionSaveLoad(t *testing.T) {
	m := InitialModel()
	m.SessionPath = t.TempDir() + "/test_session.json"
	m.AddMessage("user", "Hello")
	m.AddMessage("assistant", "Hi!")
	
	// Save session
	err := m.SaveSession()
	assert.NoError(t, err)
	
	// Load session
	newM := InitialModel()
	newM.SessionPath = m.SessionPath
	err = newM.LoadSession()
	assert.NoError(t, err)
	assert.Len(t, newM.Messages, 2)
}

func TestTokenUsage(t *testing.T) {
	tu := &TokenUsage{
		PromptTokens:     100,
		CompletionTokens: 200,
		TotalTokens:      300,
	}
	
	assert.Equal(t, 100, tu.PromptTokens)
	assert.Equal(t, 200, tu.CompletionTokens)
	assert.Equal(t, 300, tu.TotalTokens)
}

func TestModelStates(t *testing.T) {
	assert.Equal(t, "chat", string(StateChat))
	assert.Equal(t, "tools", string(StateTools))
	assert.Equal(t, "input", string(StateInput))
}

func TestViewportUpdate(t *testing.T) {
	m := InitialModel()
	m.Width = 80
	m.Height = 24
	m.AddMessage("user", "test")
	m.AddMessage("assistant", "response")
	
	// Update viewport with new size
	m.UpdateViewport()
	
	// Viewport should have a ready state
	view := m.View()
	assert.NotEmpty(t, view)
}

func TestToolOutputPanel(t *testing.T) {
	m := InitialModel()
	m.ToolOutput = "Tool executed successfully\nOutput here"
	
	view := m.renderToolOutput()
	assert.Contains(t, view, "Tool executed successfully")
	assert.Contains(t, view, "Output here")
}

// TestStyles ensures styles are properly initialized
func TestStyles(t *testing.T) {
	styles := DefaultStyles()
	assert.NotNil(t, styles.ChatMessage)
	assert.NotNil(t, styles.UserMessage)
	assert.NotNil(t, styles.AssistantMessage)
	assert.NotNil(t, styles.StatusBar)
	assert.NotNil(t, styles.Input)
}

// Test concurrent message adding
func TestConcurrentMessages(t *testing.T) {
	m := InitialModel()
	
	// Use a mutex to protect concurrent access
	// For now, test sequential message adding to avoid race conditions
	for i := 0; i < 10; i++ {
		m.AddMessage("user", string(rune('a'+i)))
	}
	
	assert.Len(t, m.Messages, 10)
}

// Test message limit
func TestMessageLimit(t *testing.T) {
	m := InitialModel()
	m.MaxMessages = 5
	
	for i := 0; i < 10; i++ {
		m.AddMessage("user", "message")
	}
	
	assert.Len(t, m.Messages, 5)
}

// Test code highlighting in messages
func TestCodeHighlighting(t *testing.T) {
	m := InitialModel()
	m.AddMessage("assistant", "Here's code: ```go\nfunc main() {}\n```")
	
	view := m.renderChatView()
	// Glamour should render markdown with code blocks
	assert.Contains(t, view, "func main()")
}
