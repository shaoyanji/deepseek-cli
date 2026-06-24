package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"gopkg.in/yaml.v3"
)

// Config represents the configuration file structure
type Config struct {
	// API settings
	APIKey  string `yaml:"api_key,omitempty"`
	BaseURL string `yaml:"base_url,omitempty"`

	// Chat defaults
	Chat ChatConfig `yaml:"chat,omitempty"`

	// FIM defaults
	FIM FIMConfig `yaml:"fim,omitempty"`

	// Security settings
	Security SecurityConfig `yaml:"security,omitempty"`
}

// SecurityConfig represents security-related settings
type SecurityConfig struct {
	ScanOutput bool `yaml:"scan_output,omitempty"`
}

// ChatConfig represents chat completion defaults
type ChatConfig struct {
	Model            string  `yaml:"model,omitempty"`
	System           string  `yaml:"system,omitempty"`
	Temperature      float64 `yaml:"temperature,omitempty"`
	TopP             float64 `yaml:"top_p,omitempty"`
	MaxTokens        int     `yaml:"max_tokens,omitempty"`
	FrequencyPenalty float64 `yaml:"frequency_penalty,omitempty"`
	PresencePenalty  float64 `yaml:"presence_penalty,omitempty"`
	Thinking         string  `yaml:"thinking,omitempty"`
	ReasoningEffort  string  `yaml:"reasoning_effort,omitempty"`
	Stream           bool    `yaml:"stream,omitempty"`
	IncludeUsage     bool    `yaml:"include_usage,omitempty"`
	JSONMode         bool    `yaml:"json_mode,omitempty"`
	Beta             bool    `yaml:"beta,omitempty"`
}

// FIMConfig represents FIM completion defaults
type FIMConfig struct {
	Model            string  `yaml:"model,omitempty"`
	MaxTokens        int     `yaml:"max_tokens,omitempty"`
	Temperature      float64 `yaml:"temperature,omitempty"`
	TopP             float64 `yaml:"top_p,omitempty"`
	FrequencyPenalty float64 `yaml:"frequency_penalty,omitempty"`
	PresencePenalty  float64 `yaml:"presence_penalty,omitempty"`
	Stream           bool    `yaml:"stream,omitempty"`
	IncludeUsage     bool    `yaml:"include_usage,omitempty"`
	Echo             bool    `yaml:"echo,omitempty"`
	Beta             bool    `yaml:"beta,omitempty"`
}

// GetDefaultChatConfig returns default chat configuration
func GetDefaultChatConfig() ChatConfig {
	return ChatConfig{
		Model:            "deepseek-v4-pro",
		Temperature:      1.0,
		TopP:             1.0,
		MaxTokens:        0,
		FrequencyPenalty: 0.0,
		PresencePenalty:  0.0,
		Thinking:         "enabled",
		ReasoningEffort:  "high",
		Stream:           true, // Changed to true - streaming is now default
		IncludeUsage:     true, // Enable usage by default for stats
		JSONMode:         false,
		Beta:             false,
	}
}

// GetDefaultFIMConfig returns default FIM configuration
func GetDefaultFIMConfig() FIMConfig {
	return FIMConfig{
		Model:            "deepseek-v4-pro",
		MaxTokens:        128,
		Temperature:      0.2,
		TopP:             1.0,
		FrequencyPenalty: 0.0,
		PresencePenalty:  0.0,
		Stream:           false,
		IncludeUsage:     false,
		Echo:             false,
		Beta:             true,
	}
}

// LoadConfig loads the configuration from XDG config directory
func LoadConfig() (*Config, error) {
	configPath, err := getConfigPath()
	if err != nil {
		return nil, err
	}

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Return empty config if file doesn't exist
		return &Config{}, nil
	}

	// Read config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse YAML
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

// getConfigPath returns the XDG config path for deepseek-cli
func getConfigPath() (string, error) {
	var configDir string

	switch runtime.GOOS {
	case "windows":
		// Windows: %APPDATA%\deepseek-cli\config.yaml
		appData := os.Getenv("APPDATA")
		if appData == "" {
			return "", fmt.Errorf("APPDATA environment variable not set")
		}
		configDir = filepath.Join(appData, "deepseek-cli")
	case "darwin":
		// macOS: ~/Library/Application Support/deepseek-cli/config.yaml
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		configDir = filepath.Join(home, "Library", "Application Support", "deepseek-cli")
	default:
		// Linux and others: ~/.config/deepseek-cli/config.yaml
		// Check for XDG_CONFIG_HOME first
		xdgConfigHome := os.Getenv("XDG_CONFIG_HOME")
		if xdgConfigHome != "" {
			configDir = filepath.Join(xdgConfigHome, "deepseek-cli")
		} else {
			// Fallback to ~/.config
			home, err := os.UserHomeDir()
			if err != nil {
				return "", fmt.Errorf("failed to get home directory: %w", err)
			}
			configDir = filepath.Join(home, ".config", "deepseek-cli")
		}
	}

	// Try YAML first, then JSON
	yamlPath := filepath.Join(configDir, "config.yaml")
	if _, err := os.Stat(yamlPath); err == nil {
		return yamlPath, nil
	}

	jsonPath := filepath.Join(configDir, "config.json")
	if _, err := os.Stat(jsonPath); err == nil {
		return jsonPath, nil
	}

	// Return YAML path as default (for creation)
	return yamlPath, nil
}

// CreateSampleConfig creates a sample config file
func CreateSampleConfig() error {
	configPath, err := getConfigPath()
	if err != nil {
		return err
	}

	// Create config directory if it doesn't exist
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Sample config
	sampleConfig := `# DeepSeek CLI Configuration (Optional)
# This file is optional - the CLI will use defaults if it doesn't exist.
# Place this file in your XDG config directory:
# Linux: ~/.config/deepseek-cli/config.yaml
# macOS: ~/Library/Application Support/deepseek-cli/config.yaml
# Windows: %APPDATA%\deepseek-cli\config.yaml

# API settings
api_key: ""  # Your DeepSeek API key (can also be set via DEEPSEEK_API_KEY env var)
base_url: ""  # Custom base URL (optional, defaults to https://api.deepseek.com)

# Chat completion defaults
chat:
  model: "deepseek-v4-pro"  # Default model
  system: ""  # Default system message
  temperature: 1.0  # Sampling temperature (0.0 to 2.0)
  top_p: 1.0  # Nucleus sampling threshold (0.0 to 1.0)
  max_tokens: 0  # Maximum tokens to generate (0 = no limit)
  frequency_penalty: 0.0  # Frequency penalty (-2.0 to 2.0)
  presence_penalty: 0.0  # Presence penalty (-2.0 to 2.0)
  thinking: "enabled"  # Thinking mode: enabled or disabled
  reasoning_effort: "high"  # Reasoning effort: high or max
  stream: true  # Enable streaming by default (use --no-stream to disable)
  include_usage: true  # Include usage info in streaming for cost stats
  json_mode: false  # Enable JSON mode by default
  beta: false  # Use beta endpoint by default

# Security settings
security:
  scan_output: true  # Scan AI output for dangerous commands

# FIM completion defaults
fim:
  model: "deepseek-v4-pro"  # Default model for FIM
  max_tokens: 128  # Maximum tokens to generate (max 4096 for FIM)
  temperature: 0.2  # Sampling temperature (lower = more focused)
  top_p: 1.0  # Nucleus sampling threshold (0.0 to 1.0)
  frequency_penalty: 0.0  # Frequency penalty (-2.0 to 2.0)
  presence_penalty: 0.0  # Presence penalty (-2.0 to 2.0)
  stream: false  # Enable streaming by default
  include_usage: false  # Include usage info in streaming
  echo: false  # Echo back the prompt with completion
  beta: true  # Use beta endpoint by default for FIM
`

	// Write sample config
	if err := os.WriteFile(configPath, []byte(sampleConfig), 0644); err != nil {
		return fmt.Errorf("failed to write sample config: %w", err)
	}

	return nil
}

// GetConfigPath returns the config file path for display purposes
func GetConfigPath() (string, error) {
	return getConfigPath()
}
