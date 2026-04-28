package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var version string

func main() {
	// Load config file (optional - will use defaults if not found)
	config, err := LoadConfig()
	if err != nil {
		config = &Config{}
	}

	// Set defaults if config is empty
	if config.Chat.Model == "" {
		config.Chat = GetDefaultChatConfig()
	}
	if config.FIM.Model == "" {
		config.FIM = GetDefaultFIMConfig()
	}

	root := &cobra.Command{
		Use:   "deepseek",
		Short: "DeepSeek API CLI",
		Long:  "Interact with DeepSeek API. Configure via DEEPSEEK_API_KEY, DEEPSEEK_API_BASE, or optional config file.",
		Version: getVersion(),
	}

	// Models command
	modelsCmd := &cobra.Command{
		Use:   "models",
		Short: "List available models",
		Long:  "List all available DeepSeek models with their details.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			base, apiKey := loadConfig()
			if apiKey == "" {
				return fmt.Errorf("DEEPSEEK_API_KEY not set")
			}
			client := NewClient(base, apiKey)
			out, err := client.do("GET", "/models", nil)
			if err != nil {
				return err
			}
			return formatModelsResponse(out)
		},
	}
	root.AddCommand(modelsCmd)

	// Balance command
	balanceCmd := &cobra.Command{
		Use:   "balance",
		Short: "Get user balance information",
		Long:  "Get current balance and account information for your DeepSeek API account.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			base, apiKey := loadConfig()
			if apiKey == "" {
				return fmt.Errorf("DEEPSEEK_API_KEY not set")
			}
			client := NewClient(base, apiKey)
			out, err := client.do("GET", "/user/balance", nil)
			if err != nil {
				return err
			}
			return formatBalanceResponse(out)
		},
	}
	root.AddCommand(balanceCmd)

	// Config command
	configCmd := &cobra.Command{
		Use:   "config",
		Short: "Manage configuration",
		Long:  "View the configuration file location and current settings. Config is optional - defaults will be used if file doesn't exist.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			configPath, err := GetConfigPath()
			if err != nil {
				return err
			}
			fmt.Printf("Config file location: %s\n", configPath)
			
			// Check if config exists
			if _, err := os.Stat(configPath); os.IsNotExist(err) {
				fmt.Println("Config file does not exist (using defaults)")
				fmt.Println("\nCurrent defaults:")
				fmt.Printf("  Model: %s\n", config.Chat.Model)
				fmt.Printf("  Temperature: %.1f\n", config.Chat.Temperature)
				fmt.Printf("  Thinking: %s\n", config.Chat.Thinking)
				fmt.Printf("  Reasoning Effort: %s\n", config.Chat.ReasoningEffort)
				fmt.Println("\nRun 'deepseek config init' to create a config file for custom settings.")
			} else {
				fmt.Println("Config file exists.")
			}
			return nil
		},
	}
	
	configInitCmd := &cobra.Command{
		Use:   "init",
		Short: "Create sample configuration file",
		Long:  "Create a sample configuration file in the XDG config directory. This is optional - the CLI works with defaults without a config file.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			configPath, err := GetConfigPath()
			if err != nil {
				return err
			}
			
			// Check if config already exists
			if _, err := os.Stat(configPath); err == nil {
				fmt.Printf("Config file already exists at: %s\n", configPath)
				fmt.Println("Edit it directly or remove it first to recreate.")
				return nil
			}
			
			if err := CreateSampleConfig(); err != nil {
				return err
			}
			
			fmt.Printf("Sample config created at: %s\n", configPath)
			fmt.Println("Edit the file to set your preferences.")
			return nil
		},
	}
	configCmd.AddCommand(configInitCmd)
	root.AddCommand(configCmd)

	// Chat completions command
	chatCmd := &cobra.Command{
		Use:   "chat [flags]",
		Short: "Create a chat completion",
		Long:  "Create chat completions using DeepSeek API. Supports thinking mode, tools, JSON output, and streaming. Use --json for raw JSON input or individual flags for parameters.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Load config (beta or regular based on --beta flag)
			var base, apiKey string
			useBeta, _ := cmd.Flags().GetBool("beta")
			if useBeta {
				base, apiKey = loadBetaConfig()
			} else {
				base, apiKey = loadConfig()
			}
			
			if apiKey == "" {
				return fmt.Errorf("DEEPSEEK_API_KEY not set")
			}

			// Check for base-url override
			if baseURL, _ := cmd.Flags().GetString("base-url"); baseURL != "" {
				base = baseURL
			}

			// Check if --json is provided for raw JSON mode
			jsonStr, _ := cmd.Flags().GetString("json")
			if jsonStr != "" {
				var payload interface{}
				if err := json.Unmarshal([]byte(jsonStr), &payload); err != nil {
					return fmt.Errorf("invalid JSON: %w", err)
				}
				client := NewClient(base, apiKey)
				out, err := client.do("POST", "/chat/completions", payload)
				if err != nil {
					return err
				}
				fmt.Println(string(out))
				return nil
			}

			// Build request from individual flags
			req, err := buildChatRequest(cmd)
			if err != nil {
				return err
			}

			// Validate request
			if err := ValidateChatRequest(req); err != nil {
				return err
			}

			client := NewClient(base, apiKey)

			// Handle streaming vs non-streaming
			if req.Stream {
				return client.streamChatCompletion(req)
			}

			out, err := client.do("POST", "/chat/completions", req)
			if err != nil {
				return err
			}
			
			// Format response based on JSON mode
			showCache, _ := cmd.Flags().GetBool("cache")
			if req.ResponseFormat != nil && req.ResponseFormat.Type == "json_object" {
				return formatJSONModeResponse(out, showCache)
			}
			return formatChatResponse(out, showCache)
		},
	}
	
	// Input flags
	chatCmd.Flags().String("json", "", "Chat completion request JSON (messages, model, etc.) - bypasses individual flags")
	chatCmd.Flags().String("model", config.Chat.Model, "Model to use (deepseek-v4-flash, deepseek-v4-pro)")
	chatCmd.Flags().String("system", config.Chat.System, "System message content")
	chatCmd.Flags().String("user", "", "User message content")
	chatCmd.Flags().String("assistant", "", "Assistant message content (for conversation history)")
	
	// Thinking and reasoning flags
	chatCmd.Flags().String("thinking", config.Chat.Thinking, "Thinking mode: enabled or disabled")
	chatCmd.Flags().String("reasoning-effort", config.Chat.ReasoningEffort, "Reasoning effort: high or max")
	
	// Sampling parameters
	chatCmd.Flags().Float64("temperature", config.Chat.Temperature, "Sampling temperature (0.0 to 2.0)")
	chatCmd.Flags().Float64("top-p", config.Chat.TopP, "Nucleus sampling threshold (0.0 to 1.0)")
	chatCmd.Flags().Int("max-tokens", config.Chat.MaxTokens, "Maximum tokens to generate (0 = no limit)")
	chatCmd.Flags().Float64("frequency-penalty", config.Chat.FrequencyPenalty, "Frequency penalty (-2.0 to 2.0)")
	chatCmd.Flags().Float64("presence-penalty", config.Chat.PresencePenalty, "Presence penalty (-2.0 to 2.0)")
	
	// Output format
	chatCmd.Flags().Bool("json-mode", config.Chat.JSONMode, "Enable JSON mode (response_format: json_object)")
	chatCmd.Flags().Bool("cache", false, "Show cache hit metrics in response")
	chatCmd.Flags().StringSlice("stop", []string{}, "Stop sequences (up to 16 strings)")
	
	// Streaming
	chatCmd.Flags().Bool("stream", config.Chat.Stream, "Enable streaming responses")
	chatCmd.Flags().Bool("include-usage", config.Chat.IncludeUsage, "Include usage info in streaming responses")
	
	// Tools
	chatCmd.Flags().String("tools", "", "Tools JSON array (e.g., '[{\"type\":\"function\",\"function\":{\"name\":\"weather\",\"parameters\":{}}}]')")
	chatCmd.Flags().String("tool-choice", "auto", "Tool choice: none, auto, required, or JSON for function")
	
	// Logprobs
	chatCmd.Flags().Bool("logprobs", false, "Return log probabilities")
	chatCmd.Flags().Int("top-logprobs", 0, "Number of top log probabilities to return (0-20)")
	
	// Beta features
	chatCmd.Flags().Bool("beta", config.Chat.Beta, "Use beta endpoint (https://api.deepseek.com/beta)")
	chatCmd.Flags().Bool("prefix-completion", false, "Enable prefix completion (beta feature)")
	
	// Other
	chatCmd.Flags().String("base-url", "", "Override base URL for this request")
	
	root.AddCommand(chatCmd)

	// FIM completions command
	fimCmd := &cobra.Command{
		Use:   "fim [flags]",
		Short: "Create a FIM (Fill-In-the-Middle) completion for code completion",
		Long:  "Create FIM completions for code completion in editors. Uses beta endpoint by default. Use --json for raw JSON input or individual flags for parameters.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Load config (beta or regular based on --beta flag)
			var base, apiKey string
			useBeta, _ := cmd.Flags().GetBool("beta")
			if useBeta {
				base, apiKey = loadBetaConfig()
			} else {
				base, apiKey = loadConfig()
			}
			
			if apiKey == "" {
				return fmt.Errorf("DEEPSEEK_API_KEY not set")
			}

			// Check for base-url override
			if baseURL, _ := cmd.Flags().GetString("base-url"); baseURL != "" {
				base = baseURL
			}

			// Check if --json is provided
			jsonStr, _ := cmd.Flags().GetString("json")
			if jsonStr != "" {
				var payload interface{}
				if err := json.Unmarshal([]byte(jsonStr), &payload); err != nil {
					return fmt.Errorf("invalid JSON: %w", err)
				}
				client := NewClient(base, apiKey)
				out, err := client.do("POST", "/completions", payload)
				if err != nil {
					return err
				}
				fmt.Println(string(out))
				return nil
			}

			// Build FIM request from individual flags
			req, err := buildFIMRequest(cmd)
			if err != nil {
				return err
			}

			// Validate request
			if err := ValidateFIMRequest(req); err != nil {
				return err
			}

			client := NewClient(base, apiKey)

			// Handle streaming vs non-streaming
			if req.Stream {
				return client.streamFIMCompletion(req)
			}

			out, err := client.do("POST", "/completions", req)
			if err != nil {
				return err
			}
			return formatFIMResponse(out)
		},
	}
	fimCmd.Flags().String("json", "", "FIM completion request JSON (prompt, suffix, model, etc.) - bypasses individual flags")
	fimCmd.Flags().String("prompt", "", "Code prefix before cursor")
	fimCmd.Flags().String("suffix", "", "Code suffix after cursor")
	fimCmd.Flags().String("base-url", "", "Override base URL for this request")
	fimCmd.Flags().String("model", config.FIM.Model, "Model to use for FIM")
	fimCmd.Flags().Int("max-tokens", config.FIM.MaxTokens, "Maximum tokens to generate (max 4096 for FIM)")
	fimCmd.Flags().Float64("temperature", config.FIM.Temperature, "Sampling temperature (lower = more focused)")
	fimCmd.Flags().Float64("top-p", config.FIM.TopP, "Nucleus sampling threshold (0.0 to 1.0)")
	fimCmd.Flags().Float64("frequency-penalty", config.FIM.FrequencyPenalty, "Frequency penalty (-2.0 to 2.0)")
	fimCmd.Flags().Float64("presence-penalty", config.FIM.PresencePenalty, "Presence penalty (-2.0 to 2.0)")
	fimCmd.Flags().StringSlice("stop", []string{}, "Stop sequences (up to 16 strings)")
	fimCmd.Flags().Bool("stream", config.FIM.Stream, "Enable streaming responses")
	fimCmd.Flags().Bool("include-usage", config.FIM.IncludeUsage, "Include usage info in streaming responses")
	fimCmd.Flags().Bool("echo", config.FIM.Echo, "Echo back the prompt with completion")
	fimCmd.Flags().Int("logprobs", 0, "Return log probabilities (0-20)")
	fimCmd.Flags().Bool("beta", config.FIM.Beta, "Use beta endpoint (default true for FIM)")
	root.AddCommand(fimCmd)

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// buildChatRequest constructs a ChatRequest from CLI flags
func buildChatRequest(cmd *cobra.Command) (*ChatRequest, error) {
	req := &ChatRequest{
		Model:           mustGetString(cmd, "model"),
		Thinking:        &ThinkingConfig{Type: mustGetString(cmd, "thinking")},
		ReasoningEffort: mustGetString(cmd, "reasoning-effort"),
		Stream:          mustGetBool(cmd, "stream"),
	}

	// Build messages from flags
	messages := []Message{}
	
	// System message
	if system := mustGetString(cmd, "system"); system != "" {
		messages = append(messages, Message{Role: "system", Content: system})
	}
	
	// User message
	if user := mustGetString(cmd, "user"); user != "" {
		messages = append(messages, Message{Role: "user", Content: user})
	}
	
	// Assistant message (for conversation history)
	if assistant := mustGetString(cmd, "assistant"); assistant != "" {
		prefix := mustGetBool(cmd, "prefix-completion")
		messages = append(messages, Message{
			Role:   "assistant",
			Content: assistant,
			Prefix: &prefix,
		})
	}
	
	// If no messages provided, error
	if len(messages) == 0 {
		return nil, fmt.Errorf("at least one message is required (use --system, --user, or --assistant)")
	}
	req.Messages = messages

	// Temperature (only set if not default)
	temp := mustGetFloat64(cmd, "temperature")
	if temp != 1.0 {
		req.Temperature = &temp
	}

	// Top P (only set if not default)
	topP := mustGetFloat64(cmd, "top-p")
	if topP != 1.0 {
		req.TopP = &topP
	}

	// Max tokens (only set if > 0)
	maxTokens := mustGetInt(cmd, "max-tokens")
	if maxTokens > 0 {
		req.MaxTokens = &maxTokens
	}

	// Frequency penalty (only set if not default)
	freqPenalty := mustGetFloat64(cmd, "frequency-penalty")
	if freqPenalty != 0.0 {
		req.FrequencyPenalty = &freqPenalty
	}

	// Presence penalty (only set if not default)
	presPenalty := mustGetFloat64(cmd, "presence-penalty")
	if presPenalty != 0.0 {
		req.PresencePenalty = &presPenalty
	}

	// JSON mode
	if mustGetBool(cmd, "json-mode") {
		req.ResponseFormat = &ResponseFormat{Type: "json_object"}
	}

	// Stop sequences
	stopSeqs := mustGetStringSlice(cmd, "stop")
	if len(stopSeqs) > 0 {
		if len(stopSeqs) == 1 {
			req.Stop = stopSeqs[0]
		} else {
			req.Stop = stopSeqs
		}
	}

	// Stream options
	if req.Stream && mustGetBool(cmd, "include-usage") {
		req.StreamOptions = &StreamOptions{IncludeUsage: true}
	}

	// Tools
	if toolsJSON := mustGetString(cmd, "tools"); toolsJSON != "" {
		var tools []Tool
		if err := json.Unmarshal([]byte(toolsJSON), &tools); err != nil {
			return nil, fmt.Errorf("invalid tools JSON: %w", err)
		}
		req.Tools = tools
	}

	// Tool choice
	toolChoice := mustGetString(cmd, "tool-choice")
	if toolChoice != "auto" {
		// Check if it's a JSON object for function-specific choice
		if strings.HasPrefix(toolChoice, "{") {
			var tcFunc map[string]interface{}
			if err := json.Unmarshal([]byte(toolChoice), &tcFunc); err != nil {
				return nil, fmt.Errorf("invalid tool-choice JSON: %w", err)
			}
			req.ToolChoice = tcFunc
		} else {
			req.ToolChoice = toolChoice
		}
	}

	// Logprobs
	if mustGetBool(cmd, "logprobs") {
		trueVal := true
		req.Logprobs = &trueVal
	}

	// Top logprobs
	topLogprobs := mustGetInt(cmd, "top-logprobs")
	if topLogprobs > 0 {
		req.TopLogprobs = &topLogprobs
	}

	return req, nil
}

// buildFIMRequest constructs a FIMRequest from CLI flags
func buildFIMRequest(cmd *cobra.Command) (*FIMRequest, error) {
	prompt := mustGetString(cmd, "prompt")
	if prompt == "" {
		return nil, fmt.Errorf("--prompt or --json required")
	}

	req := &FIMRequest{
		Model:       mustGetString(cmd, "model"),
		Prompt:      prompt,
		Stream:      mustGetBool(cmd, "stream"),
		Echo:        mustGetBool(cmd, "echo"),
	}

	// Suffix
	if suffix := mustGetString(cmd, "suffix"); suffix != "" {
		req.Suffix = &suffix
	}

	// Max tokens (only set if > 0)
	maxTokens := mustGetInt(cmd, "max-tokens")
	if maxTokens > 0 {
		req.MaxTokens = &maxTokens
	}

	// Temperature (only set if not default)
	temp := mustGetFloat64(cmd, "temperature")
	if temp != 0.2 {
		req.Temperature = &temp
	}

	// Top P (only set if not default)
	topP := mustGetFloat64(cmd, "top-p")
	if topP != 1.0 {
		req.TopP = &topP
	}

	// Frequency penalty (only set if not default)
	freqPenalty := mustGetFloat64(cmd, "frequency-penalty")
	if freqPenalty != 0.0 {
		req.FrequencyPenalty = &freqPenalty
	}

	// Presence penalty (only set if not default)
	presPenalty := mustGetFloat64(cmd, "presence-penalty")
	if presPenalty != 0.0 {
		req.PresencePenalty = &presPenalty
	}

	// Stop sequences
	stopSeqs := mustGetStringSlice(cmd, "stop")
	if len(stopSeqs) > 0 {
		if len(stopSeqs) == 1 {
			req.Stop = stopSeqs[0]
		} else {
			req.Stop = stopSeqs
		}
	}

	// Stream options
	if req.Stream && mustGetBool(cmd, "include-usage") {
		req.StreamOptions = &StreamOptions{IncludeUsage: true}
	}

	// Logprobs
	logprobs := mustGetInt(cmd, "logprobs")
	if logprobs > 0 {
		req.Logprobs = &logprobs
	}

	return req, nil
}

// Helper functions for flag access
func mustGetString(cmd *cobra.Command, name string) string {
	val, _ := cmd.Flags().GetString(name)
	return val
}

func mustGetBool(cmd *cobra.Command, name string) bool {
	val, _ := cmd.Flags().GetBool(name)
	return val
}

func mustGetInt(cmd *cobra.Command, name string) int {
	val, _ := cmd.Flags().GetInt(name)
	return val
}

func mustGetFloat64(cmd *cobra.Command, name string) float64 {
	val, _ := cmd.Flags().GetFloat64(name)
	return val
}

func mustGetStringSlice(cmd *cobra.Command, name string) []string {
	val, _ := cmd.Flags().GetStringSlice(name)
	return val
}

func getVersion() string {
	if version != "" {
		return version
	}
	return "1.0.0"
}
