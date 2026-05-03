// Package session provides session management and persistence.
package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"deepseek-cli/internal/engine"
	"deepseek-cli/internal/execpolicy"
)

// Manager handles session persistence and lifecycle
type Manager struct {
	sessionDir string
}

// NewManager creates a new session manager
func NewManager(sessionDir string) *Manager {
	return &Manager{
		sessionDir: sessionDir,
	}
}

// Save saves a session to disk
func (m *Manager) Save(sess *engine.Session) error {
	if err := os.MkdirAll(m.sessionDir, 0755); err != nil {
		return fmt.Errorf("creating session directory: %w", err)
	}

	filename := filepath.Join(m.sessionDir, fmt.Sprintf("session_%s.json", sess.ID))
	
	data, err := json.MarshalIndent(sess, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling session: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("writing session file: %w", err)
	}

	return nil
}

// Load loads a session from disk
func (m *Manager) Load(sessionID string) (*engine.Session, error) {
	filename := filepath.Join(m.sessionDir, fmt.Sprintf("session_%s.json", sessionID))
	
	data, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("session not found: %s", sessionID)
		}
		return nil, fmt.Errorf("reading session file: %w", err)
	}

	sess, err := engine.LoadSession(data)
	if err != nil {
		return nil, fmt.Errorf("parsing session: %w", err)
	}

	return sess, nil
}

// List lists all saved sessions
func (m *Manager) List() ([]*engine.Session, error) {
	files, err := os.ReadDir(m.sessionDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []*engine.Session{}, nil
		}
		return nil, fmt.Errorf("reading session directory: %w", err)
	}

	var sessions []*engine.Session
	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".json" {
			continue
		}

		filename := filepath.Join(m.sessionDir, file.Name())
		data, err := os.ReadFile(filename)
		if err != nil {
			continue // Skip unreadable files
		}

		sess, err := engine.LoadSession(data)
		if err != nil {
			continue // Skip unparseable files
		}

		sessions = append(sessions, sess)
	}

	return sessions, nil
}

// Delete deletes a session
func (m *Manager) Delete(sessionID string) error {
	filename := filepath.Join(m.sessionDir, fmt.Sprintf("session_%s.json", sessionID))
	
	if err := os.Remove(filename); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("session not found: %s", sessionID)
		}
		return fmt.Errorf("deleting session file: %w", err)
	}

	return nil
}

// CreateCheckpoint creates a checkpoint of the current session state
func (m *Manager) CreateCheckpoint(sess *engine.Session) error {
	checkpointDir := filepath.Join(m.sessionDir, "checkpoints")
	if err := os.MkdirAll(checkpointDir, 0755); err != nil {
		return fmt.Errorf("creating checkpoint directory: %w", err)
	}

	filename := filepath.Join(checkpointDir, fmt.Sprintf("%s_turn%d.json", sess.ID, sess.CurrentTurn))
	
	data, err := json.MarshalIndent(sess, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling checkpoint: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("writing checkpoint file: %w", err)
	}

	return nil
}

// RestoreCheckpoint restores a session from a checkpoint
func (m *Manager) RestoreCheckpoint(sessionID string, turn int) (*engine.Session, error) {
	checkpointDir := filepath.Join(m.sessionDir, "checkpoints")
	filename := filepath.Join(checkpointDir, fmt.Sprintf("%s_turn%d.json", sessionID, turn))
	
	data, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("checkpoint not found for session %s at turn %d", sessionID, turn)
		}
		return nil, fmt.Errorf("reading checkpoint file: %w", err)
	}

	sess, err := engine.LoadSession(data)
	if err != nil {
		return nil, fmt.Errorf("parsing checkpoint: %w", err)
	}

	return sess, nil
}

// ListCheckpoints lists all checkpoints for a session
func (m *Manager) ListCheckpoints(sessionID string) ([]int, error) {
	checkpointDir := filepath.Join(m.sessionDir, "checkpoints")
	
	files, err := os.ReadDir(checkpointDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []int{}, nil
		}
		return nil, fmt.Errorf("reading checkpoint directory: %w", err)
	}

	var turns []int
	prefix := fmt.Sprintf("%s_turn", sessionID)
	
	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".json" {
			continue
		}

		name := file.Name()
		if len(name) > len(prefix) && name[:len(prefix)] == prefix {
			// Extract turn number from filename
			turnStr := name[len(prefix):len(name)-5] // Remove "_turn" and ".json"
			var turn int
			if _, err := fmt.Sscanf(turnStr, "%d", &turn); err == nil {
				turns = append(turns, turn)
			}
		}
	}

	return turns, nil
}

// GetLatestCheckpoint returns the latest checkpoint turn number
func (m *Manager) GetLatestCheckpoint(sessionID string) (int, error) {
	turns, err := m.ListCheckpoints(sessionID)
	if err != nil {
		return 0, err
	}

	if len(turns) == 0 {
		return 0, nil
	}

	latest := turns[0]
	for _, turn := range turns {
		if turn > latest {
			latest = turn
		}
	}

	return latest, nil
}

// DefaultSessionDir returns the default session directory based on OS
func DefaultSessionDir() (string, error) {
	// Use XDG Base Directory specification
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	// Check for XDG_DATA_HOME
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		// Default locations by OS
		switch {
		case os.Getenv("APPDATA") != "":
			// Windows
			dataHome = os.Getenv("APPDATA")
		default:
			// Linux/macOS
			dataHome = filepath.Join(homeDir, ".local", "share")
		}
	}

	return filepath.Join(dataHome, "deepseek-cli", "sessions"), nil
}

// CreateSession creates a new session with the given parameters
func CreateSession(mode execpolicy.ExecutionMode, workspacePath string) (*engine.Session, error) {
	sessionID := fmt.Sprintf("%d", time.Now().UnixNano())
	return engine.NewSession(sessionID, workspacePath, mode)
}
