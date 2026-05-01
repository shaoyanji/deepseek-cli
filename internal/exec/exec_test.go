package exec

import (
	"os/exec"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestExecGoCode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}
	
	code := `package main; import "fmt"; func main() { fmt.Println("hello") }`
	result, err := ExecGo(code)
	
	assert.NoError(t, err)
	assert.Equal(t, "hello\n", result.Stdout)
	assert.Equal(t, 0, result.ExitCode)
	assert.Empty(t, result.Stderr)
}

func TestExecGoCodeCompilationError(t *testing.T) {
	code := `package main; invalid syntax`
	result, err := ExecGo(code)
	
	assert.Error(t, err)
	assert.NotEqual(t, 0, result.ExitCode)
	assert.NotEmpty(t, result.Stderr)
}

func TestExecPythonCode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}
	
	code := `print("hello from python")`
	result, err := ExecPython(code)
	
	assert.NoError(t, err)
	assert.Equal(t, "hello from python\n", result.Stdout)
	assert.Equal(t, 0, result.ExitCode)
}

func TestExecPythonCodeError(t *testing.T) {
	code := `import sys; sys.exit(1)`
	result, err := ExecPython(code)
	
	assert.NoError(t, err) // Exec itself doesn't error, but exit code is non-zero
	assert.Equal(t, 1, result.ExitCode)
}

func TestExecNodeCode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}
	
	code := `console.log("hello from node");`
	result, err := ExecNode(code)
	
	assert.NoError(t, err)
	assert.Contains(t, result.Stdout, "hello from node")
	assert.Equal(t, 0, result.ExitCode)
}

func TestExecBashCommand(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}
	
	result, err := ExecBash("echo hello")
	
	assert.NoError(t, err)
	assert.Equal(t, "hello\n", result.Stdout)
	assert.Equal(t, 0, result.ExitCode)
}

func TestExecBashCommandError(t *testing.T) {
	result, err := ExecBash("nonexistentcommand123")
	
	assert.Error(t, err)
	assert.NotEqual(t, 0, result.ExitCode)
}

func TestExecResult(t *testing.T) {
	result := &ExecResult{
		Stdout:   "output",
		Stderr:   "error",
		ExitCode: 1,
		Duration: 100,
	}
	
	assert.Equal(t, "output", result.Stdout)
	assert.Equal(t, "error", result.Stderr)
	assert.Equal(t, 1, result.ExitCode)
	assert.Equal(t, int64(100), result.Duration)
}

func TestDetectLanguage(t *testing.T) {
	tests := []struct {
		code     string
		expected string
	}{
		{`package main; func main() {}`, "go"},
		{`print("hello")`, "python"},
		{`console.log("hello")`, "node"},
		{`#include <stdio.h>`, "c"},
		{"", "unknown"},
	}
	
	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			lang := DetectLanguage(tt.code)
			assert.Equal(t, tt.expected, lang)
		})
	}
}

func TestExecAutoDetect(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}
	
	// Go code
	goCode := `package main; import "fmt"; func main() { fmt.Println("go") }`
	result, err := ExecAutoDetect(goCode)
	assert.NoError(t, err)
	assert.Equal(t, "go\n", result.Stdout)
	
	// Python code
	pyCode := `print("python")`
	result, err = ExecAutoDetect(pyCode)
	assert.NoError(t, err)
	assert.Equal(t, "python\n", result.Stdout)
}

func TestExecWithTimeout(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}
	
	// Code that runs too long
	code := `import time; time.sleep(10)`
	result, err := ExecPythonWithTimeout(code, 100) // 100ms timeout
	
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "timeout")
	assert.NotEqual(t, 0, result.ExitCode)
}

func TestExecSandboxed(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}
	
	// Test sandboxed execution (no network, restricted filesystem)
	result, err := ExecSandboxed("python", `print("sandboxed")`)
	
	assert.NoError(t, err)
	assert.Equal(t, "sandboxed\n", result.Stdout)
}

func TestTempFileCreation(t *testing.T) {
	code := "test code"
	filename, err := createTempFile(code, "go")
	
	assert.NoError(t, err)
	assert.NotEmpty(t, filename)
	
	// Cleanup
	// In real implementation, temp file would be cleaned up
}

func TestCaptureStderr(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}
	
	code := `import sys; sys.stderr.write("error output")`
	result, err := ExecPython(code)
	
	assert.NoError(t, err)
	assert.Equal(t, "error output", result.Stderr)
}

func TestExecSandboxedDocker(t *testing.T) {
	// Skip if Docker is not installed
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("Docker not installed, skipping Docker sandbox test")
	}
	
	// Test Python in Docker sandbox
	result, err := ExecSandboxedDocker("python", `print("docker sandboxed")`)
	assert.NoError(t, err)
	assert.Equal(t, "docker sandboxed\n", result.Stdout)
	
	// Test Go in Docker sandbox
	result, err = ExecSandboxedDocker("go", `package main; import "fmt"; func main() { fmt.Println("go docker") }`)
	assert.NoError(t, err)
	assert.Equal(t, "go docker\n", result.Stdout)
}

func TestExecSandboxedDockerInvalidLang(t *testing.T) {
	result, err := ExecSandboxedDocker("invalid", "")
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestContains(t *testing.T) {
	// Test contains function
	result := contains("hello world", "world")
	assert.True(t, result)

	result = contains("hello world", "xyz")
	assert.False(t, result)

	result = contains("", "test")
	assert.False(t, result)

	result = contains("test", "")
	assert.True(t, result)
}

func TestGetTempDir(t *testing.T) {
	// Test getTempDir function
	tempDir := getTempDir()
	assert.NotEmpty(t, tempDir)
	
	// On non-Windows systems, should return /tmp
	// On Windows, should return C:\temp
	// We just verify it's not empty
	assert.NotEqual(t, "", tempDir)
}

func TestMeasureDuration(t *testing.T) {
	// Test measureDuration function
	start := time.Now()
	time.Sleep(10 * time.Millisecond)
	duration := measureDuration(start)
	
	// Duration should be at least 10ms
	assert.GreaterOrEqual(t, duration, int64(10))
}
