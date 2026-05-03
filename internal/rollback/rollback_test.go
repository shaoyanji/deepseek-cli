package rollback

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewManager(t *testing.T) {
	tmpDir := t.TempDir()
	workspaceDir := t.TempDir()

	mgr := NewManager(tmpDir, workspaceDir)
	if mgr == nil {
		t.Fatal("NewManager returned nil")
	}
	if mgr.snapshotDir != tmpDir {
		t.Errorf("expected snapshotDir %s, got %s", tmpDir, mgr.snapshotDir)
	}
	if mgr.workspaceDir != workspaceDir {
		t.Errorf("expected workspaceDir %s, got %s", workspaceDir, mgr.workspaceDir)
	}
}

func TestCreateSnapshot(t *testing.T) {
	tmpDir := t.TempDir()
	workspaceDir := t.TempDir()

	// Create some test files in workspace
	testFile := filepath.Join(workspaceDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("hello world"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	subDir := filepath.Join(workspaceDir, "subdir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}
	
	subFile := filepath.Join(subDir, "nested.txt")
	if err := os.WriteFile(subFile, []byte("nested content"), 0644); err != nil {
		t.Fatalf("failed to create nested file: %v", err)
	}

	mgr := NewManager(tmpDir, workspaceDir)
	
	snapshot, err := mgr.CreateSnapshot("test-session", 1, "Test snapshot")
	if err != nil {
		t.Fatalf("CreateSnapshot failed: %v", err)
	}

	if snapshot.SessionID != "test-session" {
		t.Errorf("expected SessionID test-session, got %s", snapshot.SessionID)
	}
	if snapshot.TurnID != 1 {
		t.Errorf("expected TurnID 1, got %d", snapshot.TurnID)
	}
	if snapshot.Description != "Test snapshot" {
		t.Errorf("expected Description 'Test snapshot', got %s", snapshot.Description)
	}
	if snapshot.FileCount != 2 {
		t.Errorf("expected FileCount 2, got %d", snapshot.FileCount)
	}
	if snapshot.Size == 0 {
		t.Error("expected non-zero Size")
	}
	if !snapshot.CreatedAt.Before(time.Now()) {
		t.Error("expected CreatedAt to be in the past")
	}
}

func TestCreateSnapshotSkipsGitAndSnapshotDirs(t *testing.T) {
	tmpDir := t.TempDir()
	workspaceDir := t.TempDir()

	// Create .git directory
	gitDir := filepath.Join(workspaceDir, ".git", "objects")
	if err := os.MkdirAll(gitDir, 0755); err != nil {
		t.Fatalf("failed to create .git dir: %v", err)
	}
	gitFile := filepath.Join(gitDir, "test.obj")
	if err := os.WriteFile(gitFile, []byte("git object"), 0644); err != nil {
		t.Fatalf("failed to create git file: %v", err)
	}

	// Create snapshot directory
	snapDir := filepath.Join(workspaceDir, ".deepseek-snapshots")
	if err := os.MkdirAll(snapDir, 0755); err != nil {
		t.Fatalf("failed to create snapshot dir: %v", err)
	}
	snapFile := filepath.Join(snapDir, "should-not-be-included.tar.gz")
	if err := os.WriteFile(snapFile, []byte("fake snapshot"), 0644); err != nil {
		t.Fatalf("failed to create fake snapshot: %v", err)
	}

	// Create a regular file that should be included
	testFile := filepath.Join(workspaceDir, "included.txt")
	if err := os.WriteFile(testFile, []byte("should be included"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	mgr := NewManager(tmpDir, workspaceDir)
	
	snapshot, err := mgr.CreateSnapshot("test-session", 1, "Test snapshot")
	if err != nil {
		t.Fatalf("CreateSnapshot failed: %v", err)
	}

	// Should only include the one regular file
	if snapshot.FileCount != 1 {
		t.Errorf("expected FileCount 1 (excluding .git and .deepseek-snapshots), got %d", snapshot.FileCount)
	}
}

func TestCreateSnapshot_IncludesGitignoreAndGitHub(t *testing.T) {
	tmpDir := t.TempDir()
	workspaceDir := t.TempDir()

	// Create .git directory (should be skipped)
	gitDir := filepath.Join(workspaceDir, ".git", "objects")
	if err := os.MkdirAll(gitDir, 0755); err != nil {
		t.Fatalf("failed to create .git dir: %v", err)
	}
	gitFile := filepath.Join(gitDir, "test.obj")
	if err := os.WriteFile(gitFile, []byte("git object"), 0644); err != nil {
		t.Fatalf("failed to create git file: %v", err)
	}

	// Create .gitignore (should be included)
	gitignoreFile := filepath.Join(workspaceDir, ".gitignore")
	if err := os.WriteFile(gitignoreFile, []byte("*.log\n"), 0644); err != nil {
		t.Fatalf("failed to create .gitignore: %v", err)
	}

	// Create .github directory (should be included)
	githubDir := filepath.Join(workspaceDir, ".github", "workflows")
	if err := os.MkdirAll(githubDir, 0755); err != nil {
		t.Fatalf("failed to create .github dir: %v", err)
	}
	githubFile := filepath.Join(githubDir, "ci.yml")
	if err := os.WriteFile(githubFile, []byte("workflow config"), 0644); err != nil {
		t.Fatalf("failed to create github file: %v", err)
	}

	// Create regular file
	regularFile := filepath.Join(workspaceDir, "regular.txt")
	if err := os.WriteFile(regularFile, []byte("regular content"), 0644); err != nil {
		t.Fatalf("failed to create regular file: %v", err)
	}

	mgr := NewManager(tmpDir, workspaceDir)

	snapshot, err := mgr.CreateSnapshot("test-session", 1, "Test snapshot")
	if err != nil {
		t.Fatalf("failed to create snapshot: %v", err)
	}

	// Snapshot should contain .gitignore, .github/workflows/ci.yml, and regular.txt (not .git)
	// Expected: .gitignore, .github/workflows/ci.yml, regular.txt = 3 files
	if snapshot.FileCount != 3 {
		t.Errorf("expected FileCount 3 (.gitignore, .github/workflows/ci.yml, regular.txt), got %d", snapshot.FileCount)
	}

	// Verify by restoring and checking contents
	if err := mgr.Restore(snapshot.ID); err != nil {
		t.Fatalf("failed to restore: %v", err)
	}

	// .gitignore should exist
	if _, err := os.Stat(gitignoreFile); os.IsNotExist(err) {
		t.Error(".gitignore should exist after restore")
	}

	// .github directory should exist
	if _, err := os.Stat(githubDir); os.IsNotExist(err) {
		t.Error(".github directory should exist after restore")
	}

	// .github/workflows/ci.yml should exist
	if _, err := os.Stat(githubFile); os.IsNotExist(err) {
		t.Error(".github/workflows/ci.yml should exist after restore")
	}

	// Regular file should exist
	if _, err := os.Stat(regularFile); os.IsNotExist(err) {
		t.Error("regular.txt should exist after restore")
	}

	// Note: .git directory will still exist after restore because cleanWorkspace preserves it
	// This is intentional behavior - we don't want to destroy the actual git history
}

func TestRestore(t *testing.T) {
	tmpDir := t.TempDir()
	workspaceDir := t.TempDir()

	// Create initial files
	initialFile := filepath.Join(workspaceDir, "initial.txt")
	if err := os.WriteFile(initialFile, []byte("initial content"), 0644); err != nil {
		t.Fatalf("failed to create initial file: %v", err)
	}

	subDir := filepath.Join(workspaceDir, "subdir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}
	nestedFile := filepath.Join(subDir, "nested.txt")
	if err := os.WriteFile(nestedFile, []byte("nested content"), 0644); err != nil {
		t.Fatalf("failed to create nested file: %v", err)
	}

	mgr := NewManager(tmpDir, workspaceDir)
	
	// Create snapshot
	snapshot, err := mgr.CreateSnapshot("test-session", 1, "Initial state")
	if err != nil {
		t.Fatalf("CreateSnapshot failed: %v", err)
	}

	// Modify workspace
	if err := os.WriteFile(initialFile, []byte("modified content"), 0644); err != nil {
		t.Fatalf("failed to modify file: %v", err)
	}
	if err := os.RemoveAll(subDir); err != nil {
		t.Fatalf("failed to remove subdir: %v", err)
	}
	newFile := filepath.Join(workspaceDir, "new.txt")
	if err := os.WriteFile(newFile, []byte("new file"), 0644); err != nil {
		t.Fatalf("failed to create new file: %v", err)
	}

	// Restore from snapshot
	if err := mgr.Restore(snapshot.ID); err != nil {
		t.Fatalf("Restore failed: %v", err)
	}

	// Verify restoration
	content, err := os.ReadFile(initialFile)
	if err != nil {
		t.Fatalf("failed to read initial file: %v", err)
	}
	if string(content) != "initial content" {
		t.Errorf("expected 'initial content', got '%s'", string(content))
	}

	if _, err := os.Stat(nestedFile); os.IsNotExist(err) {
		t.Error("nested file should exist after restore")
	}

	if _, err := os.Stat(newFile); err == nil {
		t.Error("new.txt should not exist after restore")
	}
}

func TestRestoreNonExistentSnapshot(t *testing.T) {
	tmpDir := t.TempDir()
	workspaceDir := t.TempDir()

	mgr := NewManager(tmpDir, workspaceDir)
	
	err := mgr.Restore("non-existent")
	if err == nil {
		t.Error("expected error for non-existent snapshot")
	}
}

func TestListSnapshots(t *testing.T) {
	tmpDir := t.TempDir()
	workspaceDir := t.TempDir()

	// Create a test file in workspace (needed for snapshot creation)
	testFile := filepath.Join(workspaceDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	mgr := NewManager(tmpDir, workspaceDir)
	
	// Create multiple snapshots
	for i := 1; i <= 3; i++ {
		_, err := mgr.CreateSnapshot("session-1", i, "Snapshot")
		if err != nil {
			t.Fatalf("CreateSnapshot failed: %v", err)
		}
		time.Sleep(10 * time.Millisecond) // Ensure different timestamps
		
		// Recreate test file after each snapshot (since Restore will clean it)
		if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
			t.Fatalf("failed to recreate test file: %v", err)
		}
	}

	// Create snapshots for another session
	_, err := mgr.CreateSnapshot("session-2", 1, "Snapshot")
	if err != nil {
		t.Fatalf("CreateSnapshot failed: %v", err)
	}

	// List snapshots for session-1
	snapshots, err := mgr.ListSnapshots("session-1")
	if err != nil {
		t.Fatalf("ListSnapshots failed: %v", err)
	}

	if len(snapshots) != 3 {
		t.Errorf("expected 3 snapshots, got %d", len(snapshots))
	}

	// Verify sorting (newest first) - only if we have snapshots
	if len(snapshots) >= 2 && !snapshots[0].CreatedAt.After(snapshots[1].CreatedAt) {
		t.Error("expected snapshots to be sorted newest first")
	}

	// List snapshots for session-2
	snapshots2, err := mgr.ListSnapshots("session-2")
	if err != nil {
		t.Fatalf("ListSnapshots failed: %v", err)
	}

	if len(snapshots2) != 1 {
		t.Errorf("expected 1 snapshot for session-2, got %d", len(snapshots2))
	}
}

func TestGetLatestSnapshot(t *testing.T) {
	tmpDir := t.TempDir()
	workspaceDir := t.TempDir()

	testFile := filepath.Join(workspaceDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	mgr := NewManager(tmpDir, workspaceDir)
	
	// Create snapshots
	for i := 1; i <= 3; i++ {
		_, err := mgr.CreateSnapshot("test-session", i, "Snapshot")
		if err != nil {
			t.Fatalf("CreateSnapshot failed: %v", err)
		}
		time.Sleep(10 * time.Millisecond)
	}

	latest, err := mgr.GetLatestSnapshot("test-session")
	if err != nil {
		t.Fatalf("GetLatestSnapshot failed: %v", err)
	}

	if latest.TurnID != 3 {
		t.Errorf("expected TurnID 3, got %d", latest.TurnID)
	}
}

func TestGetLatestSnapshotNoSnapshots(t *testing.T) {
	tmpDir := t.TempDir()
	workspaceDir := t.TempDir()

	mgr := NewManager(tmpDir, workspaceDir)
	
	_, err := mgr.GetLatestSnapshot("non-existent")
	if err == nil {
		t.Error("expected error when no snapshots exist")
	}
}

func TestGetSnapshotByTurn(t *testing.T) {
	tmpDir := t.TempDir()
	workspaceDir := t.TempDir()

	testFile := filepath.Join(workspaceDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	mgr := NewManager(tmpDir, workspaceDir)
	
	// Create snapshots with specific turn IDs
	for i := 1; i <= 3; i++ {
		_, err := mgr.CreateSnapshot("test-session", i*2, "Snapshot")
		if err != nil {
			t.Fatalf("CreateSnapshot failed: %v", err)
		}
	}

	// Get snapshot by turn ID
	snapshot, err := mgr.GetSnapshotByTurn("test-session", 4)
	if err != nil {
		t.Fatalf("GetSnapshotByTurn failed: %v", err)
	}

	if snapshot.TurnID != 4 {
		t.Errorf("expected TurnID 4, got %d", snapshot.TurnID)
	}

	// Try non-existent turn
	_, err = mgr.GetSnapshotByTurn("test-session", 99)
	if err == nil {
		t.Error("expected error for non-existent turn")
	}
}

func TestDeleteSnapshot(t *testing.T) {
	tmpDir := t.TempDir()
	workspaceDir := t.TempDir()

	testFile := filepath.Join(workspaceDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	mgr := NewManager(tmpDir, workspaceDir)
	
	snapshot, err := mgr.CreateSnapshot("test-session", 1, "Test")
	if err != nil {
		t.Fatalf("CreateSnapshot failed: %v", err)
	}

	// Delete the snapshot
	if err := mgr.DeleteSnapshot(snapshot.ID); err != nil {
		t.Fatalf("DeleteSnapshot failed: %v", err)
	}

	// Verify deletion
	_, err = mgr.GetSnapshotInfo(snapshot.ID)
	if err == nil {
		t.Error("expected error after deletion")
	}
}

func TestDeleteNonExistentSnapshot(t *testing.T) {
	tmpDir := t.TempDir()
	workspaceDir := t.TempDir()

	mgr := NewManager(tmpDir, workspaceDir)
	
	err := mgr.DeleteSnapshot("non-existent")
	if err == nil {
		t.Error("expected error for non-existent snapshot")
	}
}

func TestCleanupOldSnapshots(t *testing.T) {
	tmpDir := t.TempDir()
	workspaceDir := t.TempDir()

	testFile := filepath.Join(workspaceDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	mgr := NewManager(tmpDir, workspaceDir)
	
	// Create recent snapshots
	for i := 1; i <= 3; i++ {
		_, err := mgr.CreateSnapshot("test-session", i, "Recent")
		if err != nil {
			t.Fatalf("CreateSnapshot failed: %v", err)
		}
		time.Sleep(10 * time.Millisecond) // Ensure different timestamps
		
		// Recreate test file after each snapshot
		if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
			t.Fatalf("failed to recreate test file: %v", err)
		}
	}

	// Verify we have snapshots before cleanup
	snapshots, _ := mgr.ListSnapshots("test-session")
	if len(snapshots) == 0 {
		t.Fatal("no snapshots created")
	}
	
	// Modify the timestamp of ALL snapshots to make them old
	files, _ := os.ReadDir(tmpDir)
	oldCount := 0
	for _, f := range files {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ".tar.gz") {
			oldFile := filepath.Join(tmpDir, f.Name())
			oldTime := time.Now().Add(-24 * time.Hour)
			_ = os.Chtimes(oldFile, oldTime, oldTime)
			oldCount++
		}
	}
	
	if oldCount == 0 {
		t.Fatal("no snapshot files found to age")
	}

	// Cleanup snapshots older than 1 hour
	deleted, err := mgr.CleanupOldSnapshots(1 * time.Hour)
	if err != nil {
		t.Fatalf("CleanupOldSnapshots failed: %v", err)
	}

	// All should be deleted since we aged them all
	if deleted != oldCount {
		t.Errorf("expected %d deleted snapshots, got %d", oldCount, deleted)
	}
}

func TestGetSnapshotInfo(t *testing.T) {
	tmpDir := t.TempDir()
	workspaceDir := t.TempDir()

	testFile := filepath.Join(workspaceDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	mgr := NewManager(tmpDir, workspaceDir)
	
	snapshot, err := mgr.CreateSnapshot("test-session", 5, "Test description")
	if err != nil {
		t.Fatalf("CreateSnapshot failed: %v", err)
	}

	info, err := mgr.GetSnapshotInfo(snapshot.ID)
	if err != nil {
		t.Fatalf("GetSnapshotInfo failed: %v", err)
	}

	if info.SessionID != "test-session" {
		t.Errorf("expected SessionID test-session, got %s", info.SessionID)
	}
	if info.TurnID != 5 {
		t.Errorf("expected TurnID 5, got %d", info.TurnID)
	}
	// FileCount may include additional metadata files, so just check it's > 0
	if info.FileCount < 1 {
		t.Errorf("expected FileCount >= 1, got %d", info.FileCount)
	}
	if info.Size == 0 {
		t.Error("expected non-zero Size")
	}
}

func TestGetSnapshotInfoNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	workspaceDir := t.TempDir()

	mgr := NewManager(tmpDir, workspaceDir)
	
	_, err := mgr.GetSnapshotInfo("non-existent")
	if err == nil {
		t.Error("expected error for non-existent snapshot")
	}
}

func TestDefaultSnapshotDir(t *testing.T) {
	workspaceDir := "/tmp/test-workspace"
	expected := filepath.Join(workspaceDir, ".deepseek-snapshots")

	result := DefaultSnapshotDir(workspaceDir)
	if result != expected {
		t.Errorf("expected %s, got %s", expected, result)
	}
}

func TestValidatePath_AbsolutePath(t *testing.T) {
	tmpDir := t.TempDir()
	workspaceDir := t.TempDir()

	mgr := NewManager(tmpDir, workspaceDir)

	// Test absolute paths (Unix-style)
	absolutePaths := []string{
		"/etc/passwd",
		"/tmp/test",
		"/usr/local/bin",
	}

	for _, path := range absolutePaths {
		err := mgr.validatePath(path)
		if err == nil {
			t.Errorf("expected error for absolute path %s", path)
		}
		if !strings.Contains(err.Error(), "absolute path not allowed") {
			t.Errorf("expected 'absolute path not allowed' error for %s, got: %v", path, err)
		}
	}
}

func TestValidatePath_PathTraversal(t *testing.T) {
	tmpDir := t.TempDir()
	workspaceDir := t.TempDir()

	mgr := NewManager(tmpDir, workspaceDir)

	// Test path traversal attempts
	traversalPaths := []string{
		"../etc/passwd",
		"../../tmp/test",
		"test/../../../etc",
		"./../test",
	}

	for _, path := range traversalPaths {
		err := mgr.validatePath(path)
		if err == nil {
			t.Errorf("expected error for traversal path %s", path)
		}
		if !strings.Contains(err.Error(), "path traversal not allowed") {
			t.Errorf("expected 'path traversal not allowed' error for %s, got: %v", path, err)
		}
	}
}

func TestValidatePath_ValidPaths(t *testing.T) {
	tmpDir := t.TempDir()
	workspaceDir := t.TempDir()

	mgr := NewManager(tmpDir, workspaceDir)

	// Test valid paths
	validPaths := []string{
		"test.txt",
		"subdir/test.txt",
		"deep/nested/path/file.txt",
		"./relative/path.txt",
	}

	for _, path := range validPaths {
		err := mgr.validatePath(path)
		if err != nil {
			t.Errorf("unexpected error for valid path %s: %v", path, err)
		}
	}
}

func TestValidatePath_EscapesWorkspace(t *testing.T) {
	tmpDir := t.TempDir()
	workspaceDir := t.TempDir()

	mgr := NewManager(tmpDir, workspaceDir)

	// Create a subdirectory in workspace
	subdir := filepath.Join(workspaceDir, "subdir")
	if err := os.MkdirAll(subdir, 0755); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}

	// Test paths that might escape through symlinks or other means
	escapePaths := []string{
		"subdir/../../etc",
		"./subdir/../../../tmp",
	}

	for _, path := range escapePaths {
		err := mgr.validatePath(path)
		if err == nil {
			t.Errorf("expected error for escape path %s", path)
		}
	}
}
