// Package subagent provides sub-agent orchestration for parallel task execution.
package subagent

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Task represents a sub-agent task
type Task struct {
	ID          int
	Prompt      string
	AllowedTools []string
	Result      string
	Error       error
	Status      TaskStatus
	StartTime   time.Time
	EndTime     time.Time
}

// TaskStatus represents the current status of a sub-agent task
type TaskStatus string

const (
	TaskPending   TaskStatus = "pending"
	TaskRunning   TaskStatus = "running"
	TaskCompleted TaskStatus = "completed"
	TaskFailed    TaskStatus = "failed"
	TaskCancelled TaskStatus = "cancelled"
)

// Manager manages a pool of sub-agent tasks
type Manager struct {
	mu          sync.Mutex
	tasks       map[int]*Task
	nextID      int
	maxConcurrent int
	executor    TaskExecutor
}

// TaskExecutor is an interface for executing sub-agent tasks
type TaskExecutor interface {
	Execute(ctx context.Context, prompt string, allowedTools []string) (string, error)
}

// DefaultExecutor is the default task executor implementation
type DefaultExecutor struct {
	// In a real implementation, this would have access to the engine/API client
}

// Execute executes a sub-agent task (placeholder implementation)
func (e *DefaultExecutor) Execute(ctx context.Context, prompt string, allowedTools []string) (string, error) {
	// This is a placeholder - in reality this would spawn a mini agent loop
	// For now, return a simulated result
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case <-time.After(100 * time.Millisecond):
		// Simulate some work
		return fmt.Sprintf("Sub-agent completed task: %s", prompt), nil
	}
}

// NewManager creates a new sub-agent manager
func NewManager(maxConcurrent int, executor TaskExecutor) *Manager {
	if maxConcurrent <= 0 {
		maxConcurrent = 3
	}
	if executor == nil {
		executor = &DefaultExecutor{}
	}
	return &Manager{
		tasks:       make(map[int]*Task),
		nextID:      1,
		maxConcurrent: maxConcurrent,
		executor:    executor,
	}
}

// Spawn starts a new sub-agent task asynchronously
func (m *Manager) Spawn(ctx context.Context, prompt string, allowedTools []string) (int, error) {
	m.mu.Lock()
	
	// Check if we're at capacity
	runningCount := 0
	for _, task := range m.tasks {
		if task.Status == TaskRunning || task.Status == TaskPending {
			runningCount++
		}
	}
	
	if runningCount >= m.maxConcurrent {
		m.mu.Unlock()
		return 0, fmt.Errorf("maximum concurrent sub-agents (%d) reached", m.maxConcurrent)
	}
	
	// Create new task
	task := &Task{
		ID:           m.nextID,
		Prompt:       prompt,
		AllowedTools: allowedTools,
		Status:       TaskPending,
		StartTime:    time.Now(),
	}
	m.nextID++
	m.tasks[task.ID] = task
	
	m.mu.Unlock()
	
	// Start task in background
	go m.runTask(ctx, task)
	
	return task.ID, nil
}

// runTask executes a single sub-agent task
func (m *Manager) runTask(ctx context.Context, task *Task) {
	m.mu.Lock()
	task.Status = TaskRunning
	m.mu.Unlock()
	
	execCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	
	result, err := m.executor.Execute(execCtx, task.Prompt, task.AllowedTools)
	
	m.mu.Lock()
	defer m.mu.Unlock()
	
	task.EndTime = time.Now()
	if err != nil {
		task.Error = err
		task.Status = TaskFailed
	} else {
		task.Result = result
		task.Status = TaskCompleted
	}
}

// GetTask returns a task by ID
func (m *Manager) GetTask(id int) (*Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	task, ok := m.tasks[id]
	if !ok {
		return nil, fmt.Errorf("task %d not found", id)
	}
	
	return task, nil
}

// GetActiveCount returns the number of active (running or pending) tasks
func (m *Manager) GetActiveCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	count := 0
	for _, task := range m.tasks {
		if task.Status == TaskRunning || task.Status == TaskPending {
			count++
		}
	}
	return count
}

// GetAllTasks returns all tasks
func (m *Manager) GetAllTasks() []*Task {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	tasks := make([]*Task, 0, len(m.tasks))
	for _, task := range m.tasks {
		tasks = append(tasks, task)
	}
	return tasks
}

// CancelTask attempts to cancel a running task
func (m *Manager) CancelTask(id int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	task, ok := m.tasks[id]
	if !ok {
		return fmt.Errorf("task %d not found", id)
	}
	
	if task.Status == TaskCompleted || task.Status == TaskFailed {
		return fmt.Errorf("cannot cancel task with status %s", task.Status)
	}
	
	task.Status = TaskCancelled
	return nil
}

// WaitForTask waits for a task to complete and returns the result
func (m *Manager) WaitForTask(ctx context.Context, id int) (string, error) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-ticker.C:
			m.mu.Lock()
			task, ok := m.tasks[id]
			if !ok {
				m.mu.Unlock()
				return "", fmt.Errorf("task %d not found", id)
			}
			
			// Copy the values we need while holding the lock
			status := task.Status
			result := task.Result
			taskErr := task.Error
			m.mu.Unlock()
			
			switch status {
			case TaskCompleted:
				return result, nil
			case TaskFailed:
				return "", taskErr
			case TaskCancelled:
				return "", fmt.Errorf("task cancelled")
			}
		}
	}
}

// Cleanup removes completed/failed tasks older than the given duration
func (m *Manager) Cleanup(maxAge time.Duration) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	removed := 0
	cutoff := time.Now().Add(-maxAge)
	
	for id, task := range m.tasks {
		if (task.Status == TaskCompleted || task.Status == TaskFailed || task.Status == TaskCancelled) &&
			task.EndTime.Before(cutoff) {
			delete(m.tasks, id)
			removed++
		}
	}
	
	return removed
}
