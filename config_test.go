package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetDefaultChatConfig(t *testing.T) {
	config := GetDefaultChatConfig()
	assert.NotNil(t, config)
	assert.Equal(t, "deepseek-v4-pro", config.Model)
	assert.Equal(t, 1.0, config.Temperature)
	assert.Equal(t, 1.0, config.TopP)
	assert.Equal(t, "enabled", config.Thinking)
	assert.Equal(t, "high", config.ReasoningEffort)
	assert.True(t, config.Stream) // Streaming is now default (Feature 2)
	assert.True(t, config.IncludeUsage) // Usage stats enabled by default
}

func TestGetDefaultFIMConfig(t *testing.T) {
	config := GetDefaultFIMConfig()
	assert.NotNil(t, config)
	assert.Equal(t, "deepseek-v4-pro", config.Model)
	assert.Equal(t, 128, config.MaxTokens)
	assert.Equal(t, 0.2, config.Temperature)
	assert.Equal(t, 1.0, config.TopP)
	assert.True(t, config.Beta)
}

func TestCreateSampleConfig(t *testing.T) {
	// Set HOME to a temp directory
	tmpDir := os.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Unsetenv("HOME")
	
	// Also set XDG_CONFIG_HOME to control where config is created
	xdgConfigDir := filepath.Join(tmpDir, ".config")
	os.Setenv("XDG_CONFIG_HOME", xdgConfigDir)
	defer os.Unsetenv("XDG_CONFIG_HOME")
	
	err := CreateSampleConfig()
	assert.NoError(t, err)
	
	// Verify file was created
	configPath := filepath.Join(xdgConfigDir, "deepseek-cli", "config.yaml")
	_, err = os.Stat(configPath)
	assert.NoError(t, err)
	
	// Read and verify content
	data, err := os.ReadFile(configPath)
	assert.NoError(t, err)
	assert.Contains(t, string(data), "api_key:")
	assert.Contains(t, string(data), "deepseek-v4-pro")
	
	// Clean up
	os.RemoveAll(configPath)
}

func TestGetConfigPath(t *testing.T) {
	// Test with HOME set
	os.Setenv("HOME", "/tmp/test")
	defer os.Unsetenv("HOME")
	
	path, err := GetConfigPath()
	assert.NoError(t, err)
	assert.NotEmpty(t, path)
	assert.Contains(t, path, ".config/deepseek-cli/config.yaml")
}

func TestLoadConfigNoFile(t *testing.T) {
	// Set HOME to a temp dir with no config file
	tmpDir := os.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Unsetenv("HOME")
	
	config, err := LoadConfig()
	// Should return empty config, not error (file doesn't exist)
	assert.NoError(t, err)
	assert.NotNil(t, config)
}

func TestLoadConfigWithInvalidYAML(t *testing.T) {
	// Create a config file with invalid YAML
	tmpDir := os.TempDir()
	configPath := filepath.Join(tmpDir, ".config", "deepseek-cli")
	_ = os.MkdirAll(configPath, 0755)
	configFile := filepath.Join(configPath, "config.yaml")
	
	_ = os.WriteFile(configFile, []byte("invalid: [yaml: content"), 0644)
	defer os.RemoveAll(configPath)
	
	os.Setenv("HOME", tmpDir)
	defer os.Unsetenv("HOME")
	
	config, err := LoadConfig()
	assert.Error(t, err)
	assert.Nil(t, config)
}

func TestGetConfigPathWindows(t *testing.T) {
	// Can't actually test Windows-specific code on Linux
	// But we can test the function doesn't panic
	// Set HOME to ensure the function has a home directory to work with
	os.Setenv("HOME", "/tmp/test")
	defer os.Unsetenv("HOME")
	
	path, err := GetConfigPath()
	assert.NoError(t, err)
	assert.NotEmpty(t, path)
}
