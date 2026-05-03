// Package config provides configuration management with TOML support.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/BurntSushi/toml"
)

// Config represents the full application configuration
type Config struct {
	// API settings
	API     APISettings `toml:"api"`
	
	// TUI settings
	TUI     TUIModeSettings `toml:"tui"`
	
	// Key bindings
	KeyBindings KeyBindingsConfig `toml:"keybindings"`
	
	// Tool permissions
	Tools     ToolSettings `toml:"tools"`
	
	// Session settings
	Session   SessionSettings `toml:"session"`
	
	// LSP settings
	LSP       LSPSettings `toml:"lsp"`
}

// APISettings holds API configuration
type APISettings struct {
	Key       string `toml:"key"`
	BaseURL   string `toml:"base_url"`
	Model     string `toml:"model"`
	Timeout   int    `toml:"timeout_seconds"`
}

// TUIModeSettings holds TUI-specific settings
type TUIModeSettings struct {
	DefaultMode      string `toml:"default_mode"`
	ShowThinking     bool   `toml:"show_thinking"`
	ShowTokenUsage   bool   `toml:"show_token_usage"`
	ShowCost         bool   `toml:"show_cost"`
	Theme            string `toml:"theme"`
	AutoSaveSession  bool   `toml:"auto_save_session"`
	ShowDiagnostics  bool   `toml:"show_diagnostics"`
}

// KeyBindingsConfig holds customizable key bindings
type KeyBindingsConfig struct {
	Send           string `toml:"send"`
	Cancel         string `toml:"cancel"`
	HistoryUp      string `toml:"history_up"`
	HistoryDown    string `toml:"history_down"`
	ClearScreen    string `toml:"clear_screen"`
	EnterCommand   string `toml:"enter_command"`
	SaveSession    string `toml:"save_session"`
	Quit           string `toml:"quit"`
	ToggleDiag     string `toml:"toggle_diagnostics"`
}

// ToolSettings holds tool execution permissions
type ToolSettings struct {
	AllowedTools    []string `toml:"allowed_tools"`
	BlockedTools    []string `toml:"blocked_tools"`
	ShellTimeoutSec int      `toml:"shell_timeout_seconds"`
	MaxOutputSize   int      `toml:"max_output_size_bytes"`
}

// SessionSettings holds session persistence settings
type SessionSettings struct {
	Directory string `toml:"directory"`
	AutoSave  bool   `toml:"auto_save"`
	MaxTurns  int    `toml:"max_turns"`
}

// LSPSettings holds LSP client configuration
type LSPSettings struct {
	Enabled bool                    `toml:"enabled"`
	Servers map[string]ServerConfig `toml:"servers"`
	Timeout int                     `toml:"timeout_seconds"`
}

// ServerConfig holds configuration for an LSP server
type ServerConfig struct {
	Command string   `toml:"command"`
	Args    []string `toml:"args"`
	RootURI string   `toml:"root_uri"`
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		API: APISettings{
			Key:       "", // Must be set via env or config file
			BaseURL:   "https://api.deepseek.com",
			Model:     "deepseek-chat",
			Timeout:   120,
		},
		TUI: TUIModeSettings{
			DefaultMode:     "agent",
			ShowThinking:    true,
			ShowTokenUsage:  true,
			ShowCost:        true,
			Theme:           "deepseek",
			AutoSaveSession: false,
			ShowDiagnostics: true,
		},
		KeyBindings: KeyBindingsConfig{
			Send:         "enter",
			Cancel:       "ctrl+c",
			HistoryUp:    "up",
			HistoryDown:  "down",
			ClearScreen:  "ctrl+l",
			EnterCommand: "esc",
			SaveSession:  "ctrl+s",
			Quit:         "ctrl+c",
			ToggleDiag:   "ctrl+d",
		},
		Tools: ToolSettings{
			AllowedTools:    []string{}, // Empty means all allowed
			BlockedTools:    []string{},
			ShellTimeoutSec: 30,
			MaxOutputSize:   1024 * 1024, // 1MB
		},
		Session: SessionSettings{
			Directory: "", // Will use XDG default
			AutoSave:  false,
			MaxTurns:  100,
		},
		LSP: LSPSettings{
			Enabled: false, // Disabled by default, requires explicit configuration
			Servers: make(map[string]ServerConfig),
			Timeout: 5,
		},
	}
}

// Load loads configuration from the XDG config directory
func Load() (*Config, error) {
	configPath, err := getConfigPath()
	if err != nil {
		return DefaultConfig(), nil // Return defaults on error
	}

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return DefaultConfig(), nil
	}

	// Read and parse TOML
	data, err := os.ReadFile(configPath)
	if err != nil {
		return DefaultConfig(), nil
	}

	cfg := DefaultConfig()
	if _, err := toml.Decode(string(data), cfg); err != nil {
		return DefaultConfig(), fmt.Errorf("parsing config file: %w", err)
	}

	// Override API key from environment if not set in config
	if cfg.API.Key == "" {
		cfg.API.Key = os.Getenv("DEEPSEEK_API_KEY")
	}

	return cfg, nil
}

// Save saves the configuration to disk
func (c *Config) Save() error {
	configPath, err := getConfigPath()
	if err != nil {
		return err
	}

	// Create directory if needed
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	file, err := os.Create(configPath)
	if err != nil {
		return fmt.Errorf("creating config file: %w", err)
	}
	defer file.Close()

	encoder := toml.NewEncoder(file)
	return encoder.Encode(c)
}

// getConfigPath returns the XDG config path for deepseek-cli
func getConfigPath() (string, error) {
	var configDir string

	switch runtime.GOOS {
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			return "", fmt.Errorf("APPDATA environment variable not set")
		}
		configDir = filepath.Join(appData, "deepseek-cli")
	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		configDir = filepath.Join(home, "Library", "Application Support", "deepseek-cli")
	default:
		xdgConfigHome := os.Getenv("XDG_CONFIG_HOME")
		if xdgConfigHome != "" {
			configDir = filepath.Join(xdgConfigHome, "deepseek-cli")
		} else {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", fmt.Errorf("failed to get home directory: %w", err)
			}
			configDir = filepath.Join(home, ".config", "deepseek-cli")
		}
	}

	return filepath.Join(configDir, "config.toml"), nil
}

// GetConfigPath returns the config file path for display purposes
func GetConfigPath() (string, error) {
	return getConfigPath()
}

// CreateSampleConfig creates a sample configuration file
func CreateSampleConfig() error {
	configPath, err := getConfigPath()
	if err != nil {
		return err
	}

	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	sampleConfig := `# DeepSeek CLI Configuration
# Place this file in your XDG config directory:
# Linux: ~/.config/deepseek-cli/config.toml
# macOS: ~/Library/Application Support/deepseek-cli/config.toml
# Windows: %APPDATA%\\deepseek-cli\\config.toml

[api]
# Your DeepSeek API key (can also be set via DEEPSEEK_API_KEY env var)
key = ""
# API base URL
base_url = "https://api.deepseek.com"
# Default model to use
model = "deepseek-chat"
# Request timeout in seconds
timeout_seconds = 120

[tui]
# Default mode: "agent", "yolo", or "acme"
default_mode = "agent"
# Show model's thinking/reasoning process
show_thinking = true
# Show token usage statistics
show_token_usage = true
# Show cost estimates
show_cost = true
# Color theme: "deepseek", "dark", "light"
theme = "deepseek"
# Automatically save session on exit
auto_save_session = false

[keybindings]
# Send message
send = "enter"
# Cancel streaming / Quit
cancel = "ctrl+c"
# Navigate command history up
history_up = "up"
# Navigate command history down
history_down = "down"
# Clear screen
clear_screen = "ctrl+l"
# Enter command mode (prefix with /)
enter_command = "esc"
# Save session
save_session = "ctrl+s"
# Quit application
quit = "ctrl+c"

[tools]
# List of allowed tools (empty = all allowed)
allowed_tools = []
# List of blocked tools
blocked_tools = []
# Shell command timeout in seconds
shell_timeout_seconds = 30
# Maximum tool output size in bytes
max_output_size_bytes = 1048576

[session]
# Session storage directory (empty = XDG default)
directory = ""
# Auto-save session periodically
auto_save = false
# Maximum number of turns to keep in memory
max_turns = 100

[lsp]
# Enable LSP diagnostics (requires server configuration)
enabled = false
# Timeout for diagnostic requests in seconds
timeout_seconds = 5

# Configure LSP servers by language ID
[lsp.servers.go]
command = "gopls"
args = ["serve"]
root_uri = ""

[lsp.servers.python]
command = "pyright-langserver"
args = ["--stdio"]
root_uri = ""

[lsp.servers.rust]
command = "rust-analyzer"
args = []
root_uri = ""

[lsp.servers.typescript]
command = "typescript-language-server"
args = ["--stdio"]
root_uri = ""
`

	return os.WriteFile(configPath, []byte(sampleConfig), 0644)
}
