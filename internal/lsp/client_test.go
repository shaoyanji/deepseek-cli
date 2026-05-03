package lsp

import (
	"context"
	"testing"
	"time"

	"go.lsp.dev/protocol"
)

func TestGetLanguageID(t *testing.T) {
	tests := []struct {
		file     string
		expected string
	}{
		{"test.go", "go"},
		{"test.py", "python"},
		{"test.rs", "rust"},
		{"test.ts", "typescript"},
		{"test.js", "typescript"},
		{"test.java", "java"},
		{"test.c", "c"},
		{"test.cpp", "cpp"},
		{"test.unknown", ""},
	}

	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			result := getLanguageID(tt.file)
			if result != tt.expected {
				t.Errorf("getLanguageID(%q) = %q, want %q", tt.file, result, tt.expected)
			}
		})
	}
}

func TestAutodetectServer(t *testing.T) {
	tests := []struct {
		langID   string
		hasCommand bool
	}{
		{"go", true},
		{"python", true},
		{"rust", true},
		{"typescript", true},
		{"unknown", false},
	}

	for _, tt := range tests {
		t.Run(tt.langID, func(t *testing.T) {
			config := autodetectServer(tt.langID)
			if tt.hasCommand && config.Command == "" {
				t.Errorf("autodetectServer(%q) should have a command", tt.langID)
			}
			if !tt.hasCommand && config.Command != "" {
				t.Errorf("autodetectServer(%q) should not have a command", tt.langID)
			}
		})
	}
}

func TestNewClient(t *testing.T) {
	configs := map[string]ServerConfig{
		"go": {Command: "gopls", Args: []string{"serve"}},
	}
	
	client := NewClient(configs, 5*time.Second)
	if client == nil {
		t.Fatal("NewClient returned nil")
	}
	
	if client.timeout != 5*time.Second {
		t.Errorf("timeout = %v, want 5s", client.timeout)
	}
	
	client.Close()
}

func TestConvertDiagnostics(t *testing.T) {
	// Test with empty slice
	result := convertDiagnostics([]protocol.Diagnostic{})
	if len(result) != 0 {
		t.Errorf("convertDiagnostics(empty) = %d items, want 0", len(result))
	}
}

func TestClientRunDiagnosticsTimeout(t *testing.T) {
	// Test that RunDiagnostics handles unsupported file types gracefully
	client := NewClient(map[string]ServerConfig{}, 1*time.Second)
	defer client.Close()

	ctx := context.Background()
	_, err := client.RunDiagnostics(ctx, "test.unknown")
	if err == nil {
		t.Error("RunDiagnostics should return error for unsupported file type")
	}
}
