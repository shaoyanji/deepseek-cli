// Package rollback provides workspace snapshot and rollback capabilities.
// It implements a "side-git" system that takes pre- and post-turn snapshots,
// enabling rollbacks without affecting the actual .git history.
package rollback

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Snapshot represents a point-in-time capture of workspace state
type Snapshot struct {
	ID          string
	TurnID      int
	SessionID   string
	CreatedAt   time.Time
	Path        string
	Size        int64
	FileCount   int
	Description string
}

// Manager handles workspace snapshots and rollbacks
type Manager struct {
	snapshotDir  string
	workspaceDir string
}

// NewManager creates a new rollback manager
func NewManager(snapshotDir, workspaceDir string) *Manager {
	return &Manager{
		snapshotDir:  snapshotDir,
		workspaceDir: workspaceDir,
	}
}

// CreateSnapshot creates a snapshot of the current workspace state
func (m *Manager) CreateSnapshot(sessionID string, turnID int, description string) (*Snapshot, error) {
	if err := os.MkdirAll(m.snapshotDir, 0755); err != nil {
		return nil, fmt.Errorf("creating snapshot directory: %w", err)
	}

	// Generate snapshot ID and filename
	timestamp := time.Now().Format("20060102_150405")
	snapshotID := fmt.Sprintf("%s_turn%d_%s", sessionID, turnID, timestamp)
	filename := filepath.Join(m.snapshotDir, fmt.Sprintf("%s.tar.gz", snapshotID))

	// Create the snapshot archive
	fileCount, size, err := m.createArchive(filename)
	if err != nil {
		return nil, fmt.Errorf("creating archive: %w", err)
	}

	snapshot := &Snapshot{
		ID:          snapshotID,
		TurnID:      turnID,
		SessionID:   sessionID,
		CreatedAt:   time.Now(),
		Path:        filename,
		Size:        size,
		FileCount:   fileCount,
		Description: description,
	}

	return snapshot, nil
}

// createArchive creates a compressed tar archive of the workspace
func (m *Manager) createArchive(filename string) (int, int64, error) {
	// Create the file
	f, err := os.Create(filename)
	if err != nil {
		return 0, 0, fmt.Errorf("creating snapshot file: %w", err)
	}
	defer f.Close()

	// Create gzip writer
	gzWriter := gzip.NewWriter(f)
	defer gzWriter.Close()

	// Create tar writer
	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

	fileCount := 0
	var totalSize int64

	// Walk the workspace directory
	err = filepath.Walk(m.workspaceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip the snapshot directory itself
		relPath, _ := filepath.Rel(m.workspaceDir, path)
		if strings.HasPrefix(relPath, ".deepseek-snapshots") {
			return nil
		}

		// Skip .git directory to avoid conflicts with actual git history
		if strings.Contains(relPath, ".git") {
			return nil
		}

		// Skip symlinks
		if info.Mode()&os.ModeSymlink != 0 {
			return nil
		}

		// Get relative path for the archive
		relPath, err = filepath.Rel(m.workspaceDir, path)
		if err != nil {
			return err
		}

		// Create tar header
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return fmt.Errorf("creating header for %s: %w", path, err)
		}

		// Set the name to relative path
		header.Name = relPath

		// Write header
		if err := tarWriter.WriteHeader(header); err != nil {
			return fmt.Errorf("writing header for %s: %w", path, err)
		}

		// If it's a file, write its contents
		if !info.IsDir() {
			file, err := os.Open(path)
			if err != nil {
				return fmt.Errorf("opening file %s: %w", path, err)
			}
			defer file.Close()

			written, err := io.Copy(tarWriter, file)
			if err != nil {
				return fmt.Errorf("writing file %s: %w", path, err)
			}

			fileCount++
			totalSize += written
		}

		return nil
	})

	if err != nil {
		return 0, 0, fmt.Errorf("walking workspace: %w", err)
	}

	// Close writers to flush data
	if err := tarWriter.Close(); err != nil {
		return 0, 0, fmt.Errorf("closing tar writer: %w", err)
	}
	if err := gzWriter.Close(); err != nil {
		return 0, 0, fmt.Errorf("closing gzip writer: %w", err)
	}

	return fileCount, totalSize, nil
}

// Restore restores the workspace from a snapshot
func (m *Manager) Restore(snapshotID string) error {
	filename := filepath.Join(m.snapshotDir, fmt.Sprintf("%s.tar.gz", snapshotID))
	
	// Check if snapshot exists
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return fmt.Errorf("snapshot not found: %s", snapshotID)
	}

	// First, clean the workspace (except .git and snapshot dir)
	if err := m.cleanWorkspace(); err != nil {
		return fmt.Errorf("cleaning workspace: %w", err)
	}

	// Open the snapshot file
	f, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("opening snapshot file: %w", err)
	}
	defer f.Close()

	// Create gzip reader
	gzReader, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("creating gzip reader: %w", err)
	}
	defer gzReader.Close()

	// Create tar reader
	tarReader := tar.NewReader(gzReader)

	// Extract files
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("reading tar: %w", err)
		}

		targetPath := filepath.Join(m.workspaceDir, header.Name)

		// Ensure parent directory exists
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return fmt.Errorf("creating directory %s: %w", targetPath, err)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			// Create directory
			if err := os.MkdirAll(targetPath, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("creating directory %s: %w", targetPath, err)
			}

		case tar.TypeReg:
			// Create file
			outFile, err := os.Create(targetPath)
			if err != nil {
				return fmt.Errorf("creating file %s: %w", targetPath, err)
			}

			written, err := io.Copy(outFile, tarReader)
			outFile.Close()
			
			if err != nil {
				return fmt.Errorf("extracting file %s: %w", targetPath, err)
			}

			if written != header.Size {
				return fmt.Errorf("size mismatch for %s: expected %d, got %d", 
					targetPath, header.Size, written)
			}

			// Restore file permissions
			if err := os.Chmod(targetPath, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("setting permissions for %s: %w", targetPath, err)
			}

		default:
			// Skip unsupported types (symlinks, etc.)
			continue
		}
	}

	return nil
}

// cleanWorkspace removes all files except .git and .deepseek-snapshots
func (m *Manager) cleanWorkspace() error {
	entries, err := os.ReadDir(m.workspaceDir)
	if err != nil {
		return fmt.Errorf("reading workspace directory: %w", err)
	}

	for _, entry := range entries {
		name := entry.Name()
		
		// Skip .git directory
		if name == ".git" {
			continue
		}
		
		// Skip snapshot directory
		if name == ".deepseek-snapshots" {
			continue
		}

		path := filepath.Join(m.workspaceDir, name)
		if err := os.RemoveAll(path); err != nil {
			return fmt.Errorf("removing %s: %w", path, err)
		}
	}

	return nil
}

// ListSnapshots returns all snapshots for a session
func (m *Manager) ListSnapshots(sessionID string) ([]*Snapshot, error) {
	files, err := os.ReadDir(m.snapshotDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []*Snapshot{}, nil
		}
		return nil, fmt.Errorf("reading snapshot directory: %w", err)
	}

	var snapshots []*Snapshot
	prefix := sessionID + "_turn"

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		name := file.Name()
		// Check for .tar.gz extension (handle double extension)
		if !strings.HasSuffix(name, ".tar.gz") {
			continue
		}

		// Remove .tar.gz suffix
		name = strings.TrimSuffix(name, ".tar.gz")
		
		if !strings.HasPrefix(name, prefix) {
			continue
		}

		// Parse turn ID from filename
		turnID := 0
		fmt.Sscanf(strings.TrimPrefix(name, prefix), "%d_", &turnID)

		// Get file info
		info, err := file.Info()
		if err != nil {
			continue
		}

		snapshot := &Snapshot{
			ID:        name,
			TurnID:    turnID,
			SessionID: sessionID,
			CreatedAt: info.ModTime(),
			Path:      filepath.Join(m.snapshotDir, file.Name()),
			Size:      info.Size(),
		}

		snapshots = append(snapshots, snapshot)
	}

	// Sort by creation time (newest first)
	sort.Slice(snapshots, func(i, j int) bool {
		return snapshots[i].CreatedAt.After(snapshots[j].CreatedAt)
	})

	return snapshots, nil
}

// GetLatestSnapshot returns the most recent snapshot for a session
func (m *Manager) GetLatestSnapshot(sessionID string) (*Snapshot, error) {
	snapshots, err := m.ListSnapshots(sessionID)
	if err != nil {
		return nil, err
	}

	if len(snapshots) == 0 {
		return nil, fmt.Errorf("no snapshots found for session %s", sessionID)
	}

	return snapshots[0], nil
}

// GetSnapshotByTurn returns a snapshot for a specific turn
func (m *Manager) GetSnapshotByTurn(sessionID string, turnID int) (*Snapshot, error) {
	snapshots, err := m.ListSnapshots(sessionID)
	if err != nil {
		return nil, err
	}

	for _, snapshot := range snapshots {
		if snapshot.TurnID == turnID {
			return snapshot, nil
		}
	}

	return nil, fmt.Errorf("no snapshot found for turn %d", turnID)
}

// DeleteSnapshot deletes a specific snapshot
func (m *Manager) DeleteSnapshot(snapshotID string) error {
	filename := filepath.Join(m.snapshotDir, fmt.Sprintf("%s.tar.gz", snapshotID))
	
	if err := os.Remove(filename); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("snapshot not found: %s", snapshotID)
		}
		return fmt.Errorf("deleting snapshot file: %w", err)
	}

	return nil
}

// CleanupOldSnapshots removes snapshots older than the specified duration
func (m *Manager) CleanupOldSnapshots(maxAge time.Duration) (int, error) {
	files, err := os.ReadDir(m.snapshotDir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("reading snapshot directory: %w", err)
	}

	cutoff := time.Now().Add(-maxAge)
	deleted := 0

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		name := file.Name()
		// Check for .tar.gz extension using HasSuffix (handles double extension correctly)
		if !strings.HasSuffix(name, ".tar.gz") {
			continue
		}

		info, err := file.Info()
		if err != nil {
			continue
		}

		if info.ModTime().Before(cutoff) {
			filename := filepath.Join(m.snapshotDir, file.Name())
			if err := os.Remove(filename); err != nil {
				continue // Skip files we can't delete
			}
			deleted++
		}
	}

	return deleted, nil
}

// GetSnapshotInfo returns detailed information about a snapshot
func (m *Manager) GetSnapshotInfo(snapshotID string) (*Snapshot, error) {
	filename := filepath.Join(m.snapshotDir, fmt.Sprintf("%s.tar.gz", snapshotID))
	
	info, err := os.Stat(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("snapshot not found: %s", snapshotID)
		}
		return nil, fmt.Errorf("getting snapshot info: %w", err)
	}

	// Parse metadata from filename
	name := strings.TrimSuffix(info.Name(), ".tar.gz")
	parts := strings.Split(name, "_turn")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid snapshot filename format: %s", name)
	}

	sessionID := parts[0]
	remaining := parts[1]
	
	turnID := 0
	fmt.Sscanf(remaining, "%d_%s", &turnID)

	snapshot := &Snapshot{
		ID:        name,
		TurnID:    turnID,
		SessionID: sessionID,
		CreatedAt: info.ModTime(),
		Path:      filename,
		Size:      info.Size(),
	}

	// Count files in archive
	fileCount, err := m.countFilesInArchive(filename)
	if err == nil {
		snapshot.FileCount = fileCount
	}

	return snapshot, nil
}

// countFilesInArchive counts the number of files in a tar.gz archive
func (m *Manager) countFilesInArchive(filename string) (int, error) {
	f, err := os.Open(filename)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	gzReader, err := gzip.NewReader(f)
	if err != nil {
		return 0, err
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)
	count := 0

	for {
		_, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return 0, err
		}
		count++
	}

	return count, nil
}

// DefaultSnapshotDir returns the default snapshot directory
func DefaultSnapshotDir(workspaceDir string) string {
	return filepath.Join(workspaceDir, ".deepseek-snapshots")
}
