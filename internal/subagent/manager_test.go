package subagent

import (
	"context"
	"testing"
	"time"
)

// mockExecutor is a mock task executor for testing
type mockExecutor struct {
	result string
	err    error
	delay  time.Duration
}

func (e *mockExecutor) Execute(ctx context.Context, prompt string, allowedTools []string) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case <-time.After(e.delay):
		if e.err != nil {
			return "", e.err
		}
		return e.result, nil
	}
}

func TestNewManager(t *testing.T) {
	mgr := NewManager(5, nil)
	if mgr == nil {
		t.Fatal("NewManager returned nil")
	}
	if mgr.maxConcurrent != 5 {
		t.Errorf("maxConcurrent = %d, want 5", mgr.maxConcurrent)
	}
	
	mgr2 := NewManager(0, nil)
	if mgr2.maxConcurrent != 3 {
		t.Errorf("maxConcurrent with 0 = %d, want 3 (default)", mgr2.maxConcurrent)
	}
}

func TestSpawnAndGetTask(t *testing.T) {
	executor := &mockExecutor{
		result: "test result",
		delay:  50 * time.Millisecond,
	}
	mgr := NewManager(3, executor)
	
	ctx := context.Background()
	id, err := mgr.Spawn(ctx, "test prompt", []string{"tool1", "tool2"})
	if err != nil {
		t.Fatalf("Spawn failed: %v", err)
	}
	if id != 1 {
		t.Errorf("task ID = %d, want 1", id)
	}
	
	// Wait for completion
	result, err := mgr.WaitForTask(ctx, id)
	if err != nil {
		t.Fatalf("WaitForTask failed: %v", err)
	}
	if result != "test result" {
		t.Errorf("result = %q, want %q", result, "test result")
	}
}

func TestMaxConcurrent(t *testing.T) {
	executor := &mockExecutor{
		result: "slow result",
		delay:  500 * time.Millisecond,
	}
	mgr := NewManager(2, executor)
	
	ctx := context.Background()
	
	// Spawn 2 tasks (should succeed)
	id1, err1 := mgr.Spawn(ctx, "task1", nil)
	id2, err2 := mgr.Spawn(ctx, "task2", nil)
	
	if err1 != nil || err2 != nil {
		t.Fatalf("Failed to spawn initial tasks: %v, %v", err1, err2)
	}
	
	// Try to spawn a 3rd task (should fail)
	_, err3 := mgr.Spawn(ctx, "task3", nil)
	if err3 == nil {
		t.Error("Spawn should fail when at max concurrent limit")
	}
	
	_ = id1
	_ = id2
}

func TestGetActiveCount(t *testing.T) {
	executor := &mockExecutor{
		result: "result",
		delay:  200 * time.Millisecond,
	}
	mgr := NewManager(5, executor)
	
	ctx := context.Background()
	
	// Initially 0 active
	if count := mgr.GetActiveCount(); count != 0 {
		t.Errorf("initial active count = %d, want 0", count)
	}
	
	// Spawn a task
	id, _ := mgr.Spawn(ctx, "task", nil)
	
	// Should be 1 active (briefly)
	time.Sleep(50 * time.Millisecond)
	if count := mgr.GetActiveCount(); count != 1 {
		t.Errorf("active count after spawn = %d, want 1", count)
	}
	
	// Wait for completion
	mgr.WaitForTask(ctx, id)
	
	// Should be 0 active again
	if count := mgr.GetActiveCount(); count != 0 {
		t.Errorf("active count after completion = %d, want 0", count)
	}
}

func TestCancelTask(t *testing.T) {
	executor := &mockExecutor{
		result: "result",
		delay:  1 * time.Second,
	}
	mgr := NewManager(5, executor)
	
	ctx := context.Background()
	id, _ := mgr.Spawn(ctx, "task", nil)
	
	// Give it a moment to start running
	time.Sleep(50 * time.Millisecond)
	
	// Cancel the task
	err := mgr.CancelTask(id)
	if err != nil {
		t.Fatalf("CancelTask failed: %v", err)
	}
	
	// Wait a bit and check status
	time.Sleep(50 * time.Millisecond)
	task, _ := mgr.GetTask(id)
	if task.Status != TaskCancelled && task.Status != TaskRunning {
		t.Errorf("task status = %s, want cancelled or running", task.Status)
	}
}

func TestGetAllTasks(t *testing.T) {
	executor := &mockExecutor{
		result: "result",
		delay:  10 * time.Millisecond,
	}
	mgr := NewManager(5, executor)
	
	ctx := context.Background()
	
	// Spawn multiple tasks
	for i := 0; i < 3; i++ {
		mgr.Spawn(ctx, "task", nil)
	}
	
	// Wait for all to complete
	time.Sleep(100 * time.Millisecond)
	
	tasks := mgr.GetAllTasks()
	if len(tasks) != 3 {
		t.Errorf("got %d tasks, want 3", len(tasks))
	}
}

func TestCleanup(t *testing.T) {
	executor := &mockExecutor{
		result: "result",
		delay:  10 * time.Millisecond,
	}
	mgr := NewManager(5, executor)
	
	ctx := context.Background()
	
	// Spawn and complete a task
	id, _ := mgr.Spawn(ctx, "task", nil)
	mgr.WaitForTask(ctx, id)
	
	// Small delay to ensure EndTime is set
	time.Sleep(50 * time.Millisecond)
	
	// Cleanup with very old cutoff should remove completed tasks
	removed := mgr.Cleanup(-1 * time.Second)
	if removed != 1 {
		t.Errorf("cleanup(-1s) removed %d tasks, want 1", removed)
	}
	
	// Verify task was removed
	tasks := mgr.GetAllTasks()
	if len(tasks) != 0 {
		t.Errorf("expected 0 tasks after cleanup, got %d", len(tasks))
	}
}

func TestTaskNotFound(t *testing.T) {
	mgr := NewManager(3, nil)
	
	_, err := mgr.GetTask(999)
	if err == nil {
		t.Error("GetTask should return error for non-existent task")
	}
	
	err = mgr.CancelTask(999)
	if err == nil {
		t.Error("CancelTask should return error for non-existent task")
	}
}

func TestWaitForTaskTimeout(t *testing.T) {
	executor := &mockExecutor{
		delay: 5 * time.Second, // Very slow
	}
	mgr := NewManager(3, executor)
	
	ctx := context.Background()
	id, _ := mgr.Spawn(ctx, "slow task", nil)
	
	// Wait with short timeout
	waitCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()
	
	_, err := mgr.WaitForTask(waitCtx, id)
	if err == nil {
		t.Error("WaitForTask should timeout")
	}
}
