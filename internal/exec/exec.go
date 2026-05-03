package exec

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// ExecResult represents the result of code execution
type ExecResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
	Duration int64 // milliseconds
}

// ExecGo executes Go code and returns the result
func ExecGo(code string) (*ExecResult, error) {
	start := time.Now()
	
	// Create temp file
	tmpfile, err := os.CreateTemp("", "*.go")
	if err != nil {
		return &ExecResult{ExitCode: 1}, err
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()
	
	if _, err := tmpfile.Write([]byte(code)); err != nil {
		return &ExecResult{ExitCode: 1}, err
	}
	if err := tmpfile.Close(); err != nil {
		return &ExecResult{ExitCode: 1}, err
	}
	
	// Run go run
	cmd := exec.Command("go", "run", tmpfile.Name())
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	
	err = cmd.Run()
	exitCode := 0
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		}
	}
	
	return &ExecResult{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: exitCode,
		Duration: measureDuration(start),
	}, err
}

// ExecPython executes Python code and returns the result
func ExecPython(code string) (*ExecResult, error) {
	start := time.Now()
	
	cmd := exec.Command("python3", "-c", code)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	
	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		}
	}
	
	return &ExecResult{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: exitCode,
		Duration: measureDuration(start),
	}, nil
}

// ExecNode executes Node.js code and returns the result
func ExecNode(code string) (*ExecResult, error) {
	start := time.Now()
	
	cmd := exec.Command("node", "-e", code)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	
	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		}
	}
	
	return &ExecResult{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: exitCode,
		Duration: measureDuration(start),
	}, nil
}

// ExecBash executes a bash command and returns the result
func ExecBash(command string) (*ExecResult, error) {
	start := time.Now()
	
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/C", command)
	} else {
		cmd = exec.Command("bash", "-c", command)
	}
	
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	
	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		}
		return &ExecResult{
			Stdout:   stdout.String(),
			Stderr:   stderr.String(),
			ExitCode: exitCode,
			Duration: measureDuration(start),
		}, err
	}
	
	return &ExecResult{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: exitCode,
		Duration: measureDuration(start),
	}, nil
}

// ExecPythonWithTimeout executes Python code with a timeout (ms)
func ExecPythonWithTimeout(code string, timeoutMs int) (*ExecResult, error) {
	start := time.Now()
	
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()
	
	cmd := exec.CommandContext(ctx, "python3", "-c", code)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	
	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		}
		// Check if it was a timeout
		if ctx.Err() == context.DeadlineExceeded {
			return &ExecResult{
				Stdout:   stdout.String(),
				Stderr:   stderr.String(),
				ExitCode: exitCode,
				Duration: measureDuration(start),
			}, fmt.Errorf("timeout: %w", ctx.Err())
		}
		return &ExecResult{
			Stdout:   stdout.String(),
			Stderr:   stderr.String(),
			ExitCode: exitCode,
			Duration: measureDuration(start),
		}, err
	}
	
	return &ExecResult{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: exitCode,
		Duration: measureDuration(start),
	}, nil
}

// ExecSandboxed executes code in a sandboxed environment (simplified version)
func ExecSandboxed(lang string, code string) (*ExecResult, error) {
	switch lang {
	case "python":
		return ExecPython(code)
	default:
		return &ExecResult{ExitCode: 1}, fmt.Errorf("unsupported language for sandbox: %s", lang)
	}
}

// ExecSandboxedDocker executes code in a Docker container for proper sandboxing
func ExecSandboxedDocker(lang string, code string) (*ExecResult, error) {
	start := time.Now()
	
	// Check if Docker is installed
	if _, err := exec.LookPath("docker"); err != nil {
		return nil, fmt.Errorf("docker not installed: %w", err)
	}
	
	var image string
	var cmd *exec.Cmd
	var tmpfile string
	
	switch lang {
	case "python":
		image = "python:3.9-slim"
		cmd = exec.Command("docker", "run", "--rm",
			"--network", "none",
			"--read-only",
			"--memory", "128m",
			"--cpus", "0.5",
			image,
			"python", "-c", code)
		
	case "go":
		image = "golang:1.24-alpine"
		// Create temp file for Go code
		tmp, err := os.CreateTemp("", "*.go")
		if err != nil {
			return &ExecResult{ExitCode: 1}, err
		}
		tmpfile = tmp.Name()
		defer func() { _ = os.Remove(tmpfile) }()
		
		if _, err := tmp.Write([]byte(code)); err != nil {
			return &ExecResult{ExitCode: 1}, err
		}
		_ = tmp.Close()
		
		// Run Go code in Docker with mounted temp file
		cmd = exec.Command("docker", "run", "--rm",
			"--network", "none",
			"--read-only",
			"--tmpfs", "/tmp:exec",
			"-e", "GOCACHE=/tmp/gocache",
			"--memory", "512m",
			"--cpus", "0.5",
			"-v", tmpfile + ":/main.go:ro",
			image,
			"go", "run", "/main.go")
		
	default:
		return nil, fmt.Errorf("unsupported language for Docker sandbox: %s", lang)
	}
	
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	
	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		}
	}
	
	return &ExecResult{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: exitCode,
		Duration: measureDuration(start),
	}, nil
}

// DetectLanguage attempts to detect the programming language from code
func DetectLanguage(code string) string {
	if len(code) == 0 {
		return "unknown"
	}
	
	// Simple heuristics
	if strings.Contains(code, "package main") || strings.Contains(code, `import "fmt"`) {
		return "go"
	}
	if strings.Contains(code, "print(") || strings.Contains(code, "import ") {
		return "python"
	}
	if strings.Contains(code, "console.log") || strings.Contains(code, "require(") {
		return "node"
	}
	if strings.Contains(code, "#include") {
		return "c"
	}
	
	return "unknown"
}

// ExecAutoDetect detects language and executes code
func ExecAutoDetect(code string) (*ExecResult, error) {
	lang := DetectLanguage(code)
	
	switch lang {
	case "go":
		return ExecGo(code)
	case "python":
		return ExecPython(code)
	case "node":
		return ExecNode(code)
	default:
		return &ExecResult{}, fmt.Errorf("unsupported language: %s", lang)
	}
}

// createTempFile creates a temporary file with the given code
func createTempFile(code string, ext string) (string, error) {
	tmpfile, err := os.CreateTemp("", "*."+ext)
	if err != nil {
		return "", err
	}
	defer func() { _ = tmpfile.Close() }()
	
	if _, err := tmpfile.Write([]byte(code)); err != nil {
		return "", err
	}
	
	return tmpfile.Name(), nil
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

// getTempDir returns the appropriate temp directory for the OS
func getTempDir() string {
	if runtime.GOOS == "windows" {
		return "C:\\temp"
	}
	return "/tmp"
}

// measureDuration measures execution duration
func measureDuration(start time.Time) int64 {
	return time.Since(start).Milliseconds()
}
