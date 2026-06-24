package session

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"deepseek-cli/internal/engine"
	"deepseek-cli/internal/execpolicy"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestSessionDir creates a temporary directory for testing
func setupTestSessionDir(t *testing.T) (string, func()) {
	tmpDir := t.TempDir()
	sessionDir := filepath.Join(tmpDir, "sessions")
	err := os.MkdirAll(sessionDir, 0755)
	require.NoError(t, err)

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return sessionDir, cleanup
}

// createTestSession creates a test session with some data
func createTestSession(t *testing.T, id string) *engine.Session {
	sess, err := engine.NewSession(id, "/tmp/test", execpolicy.ModeAgent)
	require.NoError(t, err)
	return sess
}

func TestNewManager(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewManager(tmpDir)

	assert.NotNil(t, manager)
	assert.Equal(t, tmpDir, manager.sessionDir)
}

func TestSaveAndLoad(t *testing.T) {
	sessionDir, _ := setupTestSessionDir(t)
	manager := NewManager(sessionDir)

	// Create and save a session
	sess := createTestSession(t, "test-session-1")
	sess.Model = "deepseek-v4-pro"

	err := manager.Save(sess)
	assert.NoError(t, err)

	// Load the session back
	loaded, err := manager.Load("test-session-1")
	assert.NoError(t, err)
	assert.NotNil(t, loaded)
	assert.Equal(t, sess.ID, loaded.ID)
	assert.Equal(t, sess.Model, loaded.Model)
	assert.Equal(t, sess.WorkspacePath, loaded.WorkspacePath)
}

func TestSaveCreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	sessionDir := filepath.Join(tmpDir, "nested", "sessions")
	manager := NewManager(sessionDir)

	sess := createTestSession(t, "test-session-2")
	err := manager.Save(sess)

	assert.NoError(t, err)
	_, err = os.Stat(sessionDir)
	assert.NoError(t, err)
}

func TestLoadNotFound(t *testing.T) {
	sessionDir, _ := setupTestSessionDir(t)
	manager := NewManager(sessionDir)

	_, err := manager.Load("nonexistent-session")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "session not found")
}

func TestList(t *testing.T) {
	sessionDir, _ := setupTestSessionDir(t)
	manager := NewManager(sessionDir)

	// Initially empty
	sessions, err := manager.List()
	assert.NoError(t, err)
	assert.Empty(t, sessions)

	// Add some sessions with simple names (no path separators)
	for i := 1; i <= 3; i++ {
		sess := createTestSession(t, fmt.Sprintf("test-session-%d", i))
		err := manager.Save(sess)
		assert.NoError(t, err)
	}

	// List should return all sessions
	sessions, err = manager.List()
	assert.NoError(t, err)
	assert.Len(t, sessions, 3)
}

func TestListEmptyDirectory(t *testing.T) {
	sessionDir, _ := setupTestSessionDir(t)
	manager := NewManager(sessionDir)

	sessions, err := manager.List()
	assert.NoError(t, err)
	assert.Empty(t, sessions)
}

func TestListNonExistentDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewManager(filepath.Join(tmpDir, "nonexistent"))

	sessions, err := manager.List()
	assert.NoError(t, err)
	assert.Empty(t, sessions)
}

func TestDelete(t *testing.T) {
	sessionDir, _ := setupTestSessionDir(t)
	manager := NewManager(sessionDir)

	// Create and save a session
	sess := createTestSession(t, "test-delete")
	err := manager.Save(sess)
	assert.NoError(t, err)

	// Delete it
	err = manager.Delete("test-delete")
	assert.NoError(t, err)

	// Verify it's gone
	_, err = manager.Load("test-delete")
	assert.Error(t, err)
}

func TestDeleteNotFound(t *testing.T) {
	sessionDir, _ := setupTestSessionDir(t)
	manager := NewManager(sessionDir)

	err := manager.Delete("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "session not found")
}

func TestCreateCheckpoint(t *testing.T) {
	sessionDir, _ := setupTestSessionDir(t)
	manager := NewManager(sessionDir)

	sess := createTestSession(t, "test-checkpoint")
	sess.CurrentTurn = 5

	err := manager.CreateCheckpoint(sess)
	assert.NoError(t, err)

	// Check if checkpoint file exists
	checkpointPath := filepath.Join(sessionDir, "checkpoints", "test-checkpoint_turn5.json")
	_, err = os.Stat(checkpointPath)
	assert.NoError(t, err)
}

func TestRestoreCheckpoint(t *testing.T) {
	sessionDir, _ := setupTestSessionDir(t)
	manager := NewManager(sessionDir)

	sess := createTestSession(t, "test-restore")
	sess.CurrentTurn = 3
	sess.Model = "deepseek-coder"

	err := manager.CreateCheckpoint(sess)
	assert.NoError(t, err)

	// Restore the checkpoint
	restored, err := manager.RestoreCheckpoint("test-restore", 3)
	assert.NoError(t, err)
	assert.NotNil(t, restored)
	assert.Equal(t, 3, restored.CurrentTurn)
	assert.Equal(t, "deepseek-coder", restored.Model)
}

func TestRestoreCheckpointNotFound(t *testing.T) {
	sessionDir, _ := setupTestSessionDir(t)
	manager := NewManager(sessionDir)

	_, err := manager.RestoreCheckpoint("nonexistent", 1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "checkpoint not found")
}

func TestListCheckpoints(t *testing.T) {
	sessionDir, _ := setupTestSessionDir(t)
	manager := NewManager(sessionDir)

	sess := createTestSession(t, "test-list-cp")

	// Create multiple checkpoints
	for turn := 1; turn <= 5; turn++ {
		sess.CurrentTurn = turn
		err := manager.CreateCheckpoint(sess)
		assert.NoError(t, err)
	}

	turns, err := manager.ListCheckpoints("test-list-cp")
	assert.NoError(t, err)
	assert.Len(t, turns, 5)
	assert.Contains(t, turns, 1)
	assert.Contains(t, turns, 3)
	assert.Contains(t, turns, 5)
}

func TestGetLatestCheckpoint(t *testing.T) {
	sessionDir, _ := setupTestSessionDir(t)
	manager := NewManager(sessionDir)

	sess := createTestSession(t, "test-latest")

	// Create checkpoints out of order
	for _, turn := range []int{2, 5, 1, 8, 3} {
		sess.CurrentTurn = turn
		err := manager.CreateCheckpoint(sess)
		assert.NoError(t, err)
	}

	latest, err := manager.GetLatestCheckpoint("test-latest")
	assert.NoError(t, err)
	assert.Equal(t, 8, latest)
}

func TestGetLatestCheckpointNoCheckpoints(t *testing.T) {
	sessionDir, _ := setupTestSessionDir(t)
	manager := NewManager(sessionDir)

	latest, err := manager.GetLatestCheckpoint("nonexistent")
	assert.NoError(t, err)
	assert.Equal(t, 0, latest)
}

func TestDefaultSessionDir(t *testing.T) {
	dir, err := DefaultSessionDir()
	assert.NoError(t, err)
	assert.NotEmpty(t, dir)
	assert.Contains(t, dir, "deepseek-cli")
	assert.Contains(t, dir, "sessions")
}

func TestCreateSession(t *testing.T) {
	sess, err := CreateSession(execpolicy.ModeAgent, "/tmp/workspace")
	assert.NoError(t, err)
	assert.NotNil(t, sess)
	assert.Equal(t, execpolicy.ModeAgent, sess.Mode)
	assert.Equal(t, "/tmp/workspace", sess.WorkspacePath)
}

func TestCreateNamedSession(t *testing.T) {
	sess, err := CreateNamedSession("my-project", execpolicy.ModeAgent, "/home/user/project")
	assert.NoError(t, err)
	assert.NotNil(t, sess)
	assert.Equal(t, "my-project", sess.ID)
	assert.Equal(t, execpolicy.ModeAgent, sess.Mode)
	assert.Equal(t, "/home/user/project", sess.WorkspacePath)
}

func TestCreateNamedSessionEmptyName(t *testing.T) {
	sess, err := CreateNamedSession("", execpolicy.ModeAgent, "/tmp")
	assert.Error(t, err)
	assert.Nil(t, sess)
	assert.Contains(t, err.Error(), "session name cannot be empty")
}

func TestGetSessionPath(t *testing.T) {
	sessionDir, _ := setupTestSessionDir(t)
	manager := NewManager(sessionDir)

	path := manager.GetSessionPath("my-session")
	expected := filepath.Join(sessionDir, "my-session.json")
	assert.Equal(t, expected, path)
}

func TestSessionExists(t *testing.T) {
	sessionDir, _ := setupTestSessionDir(t)
	manager := NewManager(sessionDir)

	// Initially doesn't exist
	assert.False(t, manager.SessionExists("test-exists"))

	// Create and save
	sess := createTestSession(t, "test-exists")
	err := manager.Save(sess)
	assert.NoError(t, err)

	// Now exists
	assert.True(t, manager.SessionExists("test-exists"))
}

func TestListWithInfo(t *testing.T) {
	sessionDir, _ := setupTestSessionDir(t)
	manager := NewManager(sessionDir)

	// Initially empty
	infos, err := manager.ListWithInfo()
	assert.NoError(t, err)
	assert.Empty(t, infos)

	// Create and save a session
	sess := createTestSession(t, "test-info")
	sess.Model = "deepseek-v4-pro"
	sess.Mode = execpolicy.ModeAgent

	// Add some turns to the session
	sess.Turns = []*engine.Turn{
		{UserInput: "Hello", ModelResponse: "", Status: engine.TurnComplete},
		{UserInput: "", ModelResponse: "Hi there!", Status: engine.TurnComplete},
	}

	err = manager.Save(sess)
	assert.NoError(t, err)

	// Wait a tiny bit to ensure different mod time
	time.Sleep(10 * time.Millisecond)

	// List with info
	infos, err = manager.ListWithInfo()
	assert.NoError(t, err)
	assert.Len(t, infos, 1)

	info := infos[0]
	assert.Equal(t, "test-info", info.Name)
	assert.Equal(t, 2, info.TurnCount)
	assert.Equal(t, "deepseek-v4-pro", info.Model)
	assert.Equal(t, "agent", info.Mode)
	assert.False(t, info.LastModified.IsZero())
}

func TestListWithInfoMultipleSessions(t *testing.T) {
	sessionDir, _ := setupTestSessionDir(t)
	manager := NewManager(sessionDir)

	// Create multiple sessions with different models
	sessions := []struct {
		name  string
		model string
		mode  execpolicy.ExecutionMode
	}{
		{"session1", "deepseek-v4-pro", execpolicy.ModeAgent},
		{"session2", "deepseek-coder", execpolicy.ModeAgent},
		{"session3", "deepseek-chat", execpolicy.ModeYOLO},
	}

	for _, s := range sessions {
		sess := createTestSession(t, s.name)
		sess.Model = s.model
		sess.Mode = s.mode
		sess.Turns = []*engine.Turn{
			{UserInput: "test", ModelResponse: "", Status: engine.TurnComplete},
		}
		err := manager.Save(sess)
		assert.NoError(t, err)
	}

	infos, err := manager.ListWithInfo()
	assert.NoError(t, err)
	assert.Len(t, infos, 3)

	// Verify each session info
	foundModels := make(map[string]bool)
	for _, info := range infos {
		foundModels[info.Model] = true
		assert.NotEmpty(t, info.Name)
		assert.False(t, info.LastModified.IsZero())
	}

	assert.True(t, foundModels["deepseek-v4-pro"])
	assert.True(t, foundModels["deepseek-coder"])
	assert.True(t, foundModels["deepseek-chat"])
}

func TestListWithInfoCorruptedFile(t *testing.T) {
	sessionDir, _ := setupTestSessionDir(t)
	manager := NewManager(sessionDir)

	// Create a valid session
	sess := createTestSession(t, "valid-session")
	err := manager.Save(sess)
	assert.NoError(t, err)

	// Create a corrupted session file
	corruptedPath := filepath.Join(sessionDir, "corrupted.json")
	err = os.WriteFile(corruptedPath, []byte("not valid json"), 0644)
	assert.NoError(t, err)

	// Should still list the valid session, skipping the corrupted one
	infos, err := manager.ListWithInfo()
	assert.NoError(t, err)
	assert.Len(t, infos, 1)
	assert.Equal(t, "valid-session", infos[0].Name)
}

func TestSaveLoadRoundTrip(t *testing.T) {
	sessionDir, _ := setupTestSessionDir(t)
	manager := NewManager(sessionDir)

	// Create a session with complex state
	sess := createTestSession(t, "roundtrip")
	sess.Model = "deepseek-v4-pro"
	sess.Mode = execpolicy.ModeAgent
	sess.WorkspacePath = "/home/user/myproject"
	sess.Turns = []*engine.Turn{
		{UserInput: "", ModelResponse: "You are helpful", Status: engine.TurnComplete},
		{UserInput: "Write a function", ModelResponse: "Here's the function...", Status: engine.TurnComplete},
		{UserInput: "Thanks!", ModelResponse: "You're welcome!", Status: engine.TurnComplete},
	}
	sess.CurrentTurn = 2

	// Save and load
	err := manager.Save(sess)
	assert.NoError(t, err)

	loaded, err := manager.Load("roundtrip")
	assert.NoError(t, err)

	// Verify all fields
	assert.Equal(t, sess.ID, loaded.ID)
	assert.Equal(t, sess.Model, loaded.Model)
	assert.Equal(t, sess.Mode, loaded.Mode)
	assert.Equal(t, sess.WorkspacePath, loaded.WorkspacePath)
	assert.Equal(t, len(sess.Turns), len(loaded.Turns))
	assert.Equal(t, sess.CurrentTurn, loaded.CurrentTurn)

	// Verify turn contents
	for i := range sess.Turns {
		assert.Equal(t, sess.Turns[i].UserInput, loaded.Turns[i].UserInput)
		assert.Equal(t, sess.Turns[i].ModelResponse, loaded.Turns[i].ModelResponse)
	}
}

func TestCheckpointPreservesState(t *testing.T) {
	sessionDir, _ := setupTestSessionDir(t)
	manager := NewManager(sessionDir)

	sess := createTestSession(t, "checkpoint-state")
	sess.Model = "deepseek-coder"
	sess.Turns = []*engine.Turn{
		{UserInput: "First message", ModelResponse: "", Status: engine.TurnComplete},
		{UserInput: "", ModelResponse: "First response", Status: engine.TurnComplete},
		{UserInput: "Second message", ModelResponse: "", Status: engine.TurnComplete},
	}
	sess.CurrentTurn = 2

	// Create checkpoint
	err := manager.CreateCheckpoint(sess)
	assert.NoError(t, err)

	// Modify original session
	sess.Turns = append(sess.Turns, &engine.Turn{UserInput: "", ModelResponse: "Second response", Status: engine.TurnComplete})
	sess.CurrentTurn = 3

	// Restore from checkpoint
	restored, err := manager.RestoreCheckpoint("checkpoint-state", 2)
	assert.NoError(t, err)

	// Should have original state
	assert.Equal(t, 2, restored.CurrentTurn)
	assert.Len(t, restored.Turns, 3)
	assert.Equal(t, "deepseek-coder", restored.Model)
}
