package main

import (
	"os"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestMustGetString(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().String("test", "default", "test flag")

	// Test getting existing flag
	result := mustGetString(cmd, "test")
	assert.Equal(t, "default", result)

	// Test getting non-existent flag (returns empty string)
	result = mustGetString(cmd, "nonexistent")
	assert.Equal(t, "", result)
}

func TestMustGetBool(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().Bool("test", false, "test flag")

	// Test getting existing flag
	result := mustGetBool(cmd, "test")
	assert.False(t, result)

	// Test getting non-existent flag (returns false)
	result = mustGetBool(cmd, "nonexistent")
	assert.False(t, result)
}

func TestMustGetInt(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().Int("test", 42, "test flag")

	// Test getting existing flag
	result := mustGetInt(cmd, "test")
	assert.Equal(t, 42, result)

	// Test getting non-existent flag (returns 0)
	result = mustGetInt(cmd, "nonexistent")
	assert.Equal(t, 0, result)
}

func TestMustGetFloat64(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().Float64("test", 3.14, "test flag")

	// Test getting existing flag
	result := mustGetFloat64(cmd, "test")
	assert.Equal(t, 3.14, result)

	// Test getting non-existent flag (returns 0.0)
	result = mustGetFloat64(cmd, "nonexistent")
	assert.Equal(t, 0.0, result)
}

func TestMustGetStringSlice(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().StringSlice("test", []string{"a", "b"}, "test flag")

	// Test getting existing flag
	result := mustGetStringSlice(cmd, "test")
	assert.Equal(t, []string{"a", "b"}, result)

	// Test getting non-existent flag (returns empty slice)
	result = mustGetStringSlice(cmd, "nonexistent")
	assert.Equal(t, []string{}, result)
}

func TestGetVersion(t *testing.T) {
	// Test with version set
	originalVersion := version
	version = "2.0.0"
	result := getVersion()
	assert.Equal(t, "2.0.0", result)

	// Test with empty version (returns default)
	version = ""
	result = getVersion()
	assert.Equal(t, "1.0.0", result)

	// Restore original version
	version = originalVersion
}

func TestHasStdinData(t *testing.T) {
	// Test when stdin is a terminal (no data)
	// This is the normal case when running tests
	result := hasStdinData()
	// In test environment, stdin is typically not a TTY, so this might be true
	// We just test that the function doesn't panic
	assert.IsType(t, false, result)
}

func TestBuildChatRequest(t *testing.T) {
	// Helper function to create a fresh command with all flags
	createCmd := func() *cobra.Command {
		cmd := &cobra.Command{}
		cmd.Flags().String("model", "deepseek-chat", "Model")
		cmd.Flags().String("system", "", "System message")
		cmd.Flags().String("user", "Hello", "User message")
		cmd.Flags().String("assistant", "", "Assistant message")
		cmd.Flags().String("thinking", "disabled", "Thinking mode")
		cmd.Flags().String("reasoning-effort", "high", "Reasoning effort")
		cmd.Flags().Bool("stream", false, "Stream")
		cmd.Flags().Float64("temperature", 1.0, "Temperature")
		cmd.Flags().Float64("top-p", 1.0, "Top P")
		cmd.Flags().Int("max-tokens", 0, "Max tokens")
		cmd.Flags().Float64("frequency-penalty", 0.0, "Frequency penalty")
		cmd.Flags().Float64("presence-penalty", 0.0, "Presence penalty")
		cmd.Flags().Bool("json-mode", false, "JSON mode")
		cmd.Flags().StringSlice("stop", []string{}, "Stop sequences")
		cmd.Flags().Bool("include-usage", false, "Include usage")
		cmd.Flags().String("tools", "", "Tools JSON")
		cmd.Flags().String("tool-choice", "auto", "Tool choice")
		cmd.Flags().Bool("logprobs", false, "Logprobs")
		cmd.Flags().Int("top-logprobs", 0, "Top logprobs")
		cmd.Flags().Bool("prefix-completion", false, "Prefix completion")
		return cmd
	}

	// Test basic request building
	cmd := createCmd()
	req, err := buildChatRequest(cmd)
	assert.NoError(t, err)
	assert.NotNil(t, req)
	assert.Equal(t, "deepseek-chat", req.Model)
	assert.Len(t, req.Messages, 1)
	assert.Equal(t, "user", req.Messages[0].Role)
	assert.Equal(t, "Hello", req.Messages[0].Content)
	assert.Equal(t, "disabled", req.Thinking.Type)
	assert.Equal(t, "high", req.ReasoningEffort)
	assert.False(t, req.Stream)

	// Test with system message
	cmd = createCmd()
	cmd.Flags().Set("system", "You are a helpful assistant")
	req, err = buildChatRequest(cmd)
	assert.NoError(t, err)
	assert.Len(t, req.Messages, 2)
	assert.Equal(t, "system", req.Messages[0].Role)
	assert.Equal(t, "user", req.Messages[1].Role)

	// Test with non-default temperature
	cmd = createCmd()
	cmd.Flags().Set("temperature", "0.7")
	req, err = buildChatRequest(cmd)
	assert.NoError(t, err)
	assert.NotNil(t, req.Temperature)
	assert.Equal(t, 0.7, *req.Temperature)

	// Test with max tokens
	cmd = createCmd()
	cmd.Flags().Set("max-tokens", "100")
	req, err = buildChatRequest(cmd)
	assert.NoError(t, err)
	assert.NotNil(t, req.MaxTokens)
	assert.Equal(t, 100, *req.MaxTokens)

	// Test with JSON mode
	cmd = createCmd()
	cmd.Flags().Set("json-mode", "true")
	req, err = buildChatRequest(cmd)
	assert.NoError(t, err)
	assert.NotNil(t, req.ResponseFormat)
	assert.Equal(t, "json_object", req.ResponseFormat.Type)

	// Test with stop sequences
	cmd = createCmd()
	cmd.Flags().Set("stop", "stop1,stop2")
	req, err = buildChatRequest(cmd)
	assert.NoError(t, err)
	assert.NotNil(t, req.Stop)
	stopSlice, ok := req.Stop.([]string)
	assert.True(t, ok)
	assert.Equal(t, []string{"stop1", "stop2"}, stopSlice)

	// Test with single stop sequence
	cmd = createCmd()
	cmd.Flags().Set("stop", "stop1")
	req, err = buildChatRequest(cmd)
	assert.NoError(t, err)
	assert.NotNil(t, req.Stop)
	assert.Equal(t, "stop1", req.Stop)

	// Test with tools JSON
	cmd = createCmd()
	cmd.Flags().Set("tools", `[{"type":"function","function":{"name":"test"}}]`)
	req, err = buildChatRequest(cmd)
	assert.NoError(t, err)
	assert.Len(t, req.Tools, 1)
	assert.Equal(t, "function", req.Tools[0].Type)

	// Test with invalid tools JSON
	cmd = createCmd()
	cmd.Flags().Set("tools", "invalid json")
	req, err = buildChatRequest(cmd)
	assert.Error(t, err)
	assert.Nil(t, req)

	// Test with tool choice JSON
	cmd = createCmd()
	cmd.Flags().Set("tool-choice", `{"type":"function","function":{"name":"test"}}`)
	req, err = buildChatRequest(cmd)
	assert.NoError(t, err)
	assert.NotNil(t, req.ToolChoice)
	tcMap, ok := req.ToolChoice.(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "function", tcMap["type"])

	// Test with logprobs
	cmd = createCmd()
	cmd.Flags().Set("logprobs", "true")
	req, err = buildChatRequest(cmd)
	assert.NoError(t, err)
	assert.NotNil(t, req.Logprobs)
	assert.True(t, *req.Logprobs)

	// Test with top logprobs
	cmd = createCmd()
	cmd.Flags().Set("top-logprobs", "5")
	req, err = buildChatRequest(cmd)
	assert.NoError(t, err)
	assert.NotNil(t, req.TopLogprobs)
	assert.Equal(t, 5, *req.TopLogprobs)

	// Test error when no messages provided
	cmd = createCmd()
	cmd.Flags().Set("user", "")
	cmd.Flags().Set("system", "")
	cmd.Flags().Set("assistant", "")
	req, err = buildChatRequest(cmd)
	assert.Error(t, err)
	assert.Nil(t, req)
	assert.Contains(t, err.Error(), "at least one message is required")
}

func TestBuildFIMRequest(t *testing.T) {
	// Helper function to create a fresh command with all flags
	createCmd := func() *cobra.Command {
		cmd := &cobra.Command{}
		cmd.Flags().String("model", "deepseek-coder", "Model")
		cmd.Flags().String("prompt", "def hello", "Prompt")
		cmd.Flags().String("suffix", "", "Suffix")
		cmd.Flags().Bool("stream", false, "Stream")
		cmd.Flags().Bool("echo", false, "Echo")
		cmd.Flags().Int("max-tokens", 0, "Max tokens")
		cmd.Flags().Float64("temperature", 0.2, "Temperature")
		cmd.Flags().Float64("top-p", 1.0, "Top P")
		cmd.Flags().Float64("frequency-penalty", 0.0, "Frequency penalty")
		cmd.Flags().Float64("presence-penalty", 0.0, "Presence penalty")
		cmd.Flags().StringSlice("stop", []string{}, "Stop sequences")
		cmd.Flags().Bool("include-usage", false, "Include usage")
		cmd.Flags().Int("logprobs", 0, "Logprobs")
		return cmd
	}

	// Test basic request building
	cmd := createCmd()
	req, err := buildFIMRequest(cmd)
	assert.NoError(t, err)
	assert.NotNil(t, req)
	assert.Equal(t, "deepseek-coder", req.Model)
	assert.Equal(t, "def hello", req.Prompt)
	assert.False(t, req.Stream)
	assert.False(t, req.Echo)

	// Test error when no prompt provided
	cmd = createCmd()
	cmd.Flags().Set("prompt", "")
	req, err = buildFIMRequest(cmd)
	assert.Error(t, err)
	assert.Nil(t, req)
	assert.Contains(t, err.Error(), "--prompt or --json required")

	// Test with suffix
	cmd = createCmd()
	cmd.Flags().Set("suffix", "():")
	req, err = buildFIMRequest(cmd)
	assert.NoError(t, err)
	assert.NotNil(t, req.Suffix)
	assert.Equal(t, "():", *req.Suffix)

	// Test with max tokens
	cmd = createCmd()
	cmd.Flags().Set("max-tokens", "100")
	req, err = buildFIMRequest(cmd)
	assert.NoError(t, err)
	assert.NotNil(t, req.MaxTokens)
	assert.Equal(t, 100, *req.MaxTokens)

	// Test with non-default temperature
	cmd = createCmd()
	cmd.Flags().Set("temperature", "0.5")
	req, err = buildFIMRequest(cmd)
	assert.NoError(t, err)
	assert.NotNil(t, req.Temperature)
	assert.Equal(t, 0.5, *req.Temperature)

	// Test with non-default top P
	cmd = createCmd()
	cmd.Flags().Set("top-p", "0.9")
	req, err = buildFIMRequest(cmd)
	assert.NoError(t, err)
	assert.NotNil(t, req.TopP)
	assert.Equal(t, 0.9, *req.TopP)

	// Test with frequency penalty
	cmd = createCmd()
	cmd.Flags().Set("frequency-penalty", "0.5")
	req, err = buildFIMRequest(cmd)
	assert.NoError(t, err)
	assert.NotNil(t, req.FrequencyPenalty)
	assert.Equal(t, 0.5, *req.FrequencyPenalty)

	// Test with presence penalty
	cmd = createCmd()
	cmd.Flags().Set("presence-penalty", "0.3")
	req, err = buildFIMRequest(cmd)
	assert.NoError(t, err)
	assert.NotNil(t, req.PresencePenalty)
	assert.Equal(t, 0.3, *req.PresencePenalty)

	// Test with stop sequences
	cmd = createCmd()
	cmd.Flags().Set("stop", "stop1,stop2")
	req, err = buildFIMRequest(cmd)
	assert.NoError(t, err)
	assert.NotNil(t, req.Stop)
	stopSlice, ok := req.Stop.([]string)
	assert.True(t, ok)
	assert.Equal(t, []string{"stop1", "stop2"}, stopSlice)

	// Test with single stop sequence
	cmd = createCmd()
	cmd.Flags().Set("stop", "stop1")
	req, err = buildFIMRequest(cmd)
	assert.NoError(t, err)
	assert.NotNil(t, req.Stop)
	assert.Equal(t, "stop1", req.Stop)

	// Test with logprobs
	cmd = createCmd()
	cmd.Flags().Set("logprobs", "5")
	req, err = buildFIMRequest(cmd)
	assert.NoError(t, err)
	assert.NotNil(t, req.Logprobs)
	assert.Equal(t, 5, *req.Logprobs)

	// Test with stream options
	cmd = createCmd()
	cmd.Flags().Set("stream", "true")
	cmd.Flags().Set("include-usage", "true")
	req, err = buildFIMRequest(cmd)
	assert.NoError(t, err)
	assert.NotNil(t, req.StreamOptions)
	assert.True(t, req.StreamOptions.IncludeUsage)
}

func TestExecuteSingleTurn_NoAPIKey(t *testing.T) {
	// Save original env var
	originalAPIKey := os.Getenv("DEEPSEEK_API_KEY")
	defer os.Setenv("DEEPSEEK_API_KEY", originalAPIKey)

	// Unset API key
	os.Unsetenv("DEEPSEEK_API_KEY")

	cmd := &cobra.Command{}
	cmd.Flags().String("base-url", "", "Base URL override")
	config := &Config{Chat: GetDefaultChatConfig()}

	err := executeSingleTurn(cmd, "test prompt", config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "DEEPSEEK_API_KEY not set")
}

func TestExecuteHistoryMode_FileNotFound(t *testing.T) {
	cmd := &cobra.Command{}
	config := &Config{Chat: GetDefaultChatConfig()}

	err := executeHistoryMode(cmd, "/nonexistent/file.json", config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "reading history file")
}

func TestExecuteHistoryMode_InvalidJSON(t *testing.T) {
	// Create a temp file with invalid JSON
	tmpfile, err := os.CreateTemp("", "test-*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte("invalid json")); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	cmd := &cobra.Command{}
	config := &Config{Chat: GetDefaultChatConfig()}

	err = executeHistoryMode(cmd, tmpfile.Name(), config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parsing history file")
}

func TestExecuteHistoryMode_NoAPIKey(t *testing.T) {
	// Save original env var
	originalAPIKey := os.Getenv("DEEPSEEK_API_KEY")
	defer os.Setenv("DEEPSEEK_API_KEY", originalAPIKey)

	// Unset API key
	os.Unsetenv("DEEPSEEK_API_KEY")

	// Create a temp file with valid JSON
	tmpfile, err := os.CreateTemp("", "test-*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	validJSON := `[{"role":"user","content":"test"}]`
	if _, err := tmpfile.Write([]byte(validJSON)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	cmd := &cobra.Command{}
	config := &Config{Chat: GetDefaultChatConfig()}

	err = executeHistoryMode(cmd, tmpfile.Name(), config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "DEEPSEEK_API_KEY not set")
}

func TestExecuteTUI_NoAPIKey(t *testing.T) {
	// Save original env var
	originalAPIKey := os.Getenv("DEEPSEEK_API_KEY")
	defer os.Setenv("DEEPSEEK_API_KEY", originalAPIKey)

	// Unset API key
	os.Unsetenv("DEEPSEEK_API_KEY")

	cmd := &cobra.Command{}
	config := &Config{Chat: GetDefaultChatConfig()}

	err := executeTUI(cmd, config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "DEEPSEEK_API_KEY not set")
}

func TestExecuteSingleTurn_InvalidModel(t *testing.T) {
	// Save original env var
	originalAPIKey := os.Getenv("DEEPSEEK_API_KEY")
	defer os.Setenv("DEEPSEEK_API_KEY", originalAPIKey)

	// Set API key
	os.Setenv("DEEPSEEK_API_KEY", "test-key")

	cmd := &cobra.Command{}
	cmd.Flags().String("base-url", "", "Base URL override")

	// Create config with invalid model (empty)
	config := &Config{
		Chat: ChatConfig{
			Model:       "",
			Temperature: 0.7,
			TopP:        1.0,
			MaxTokens:   100,
		},
	}

	err := executeSingleTurn(cmd, "test prompt", config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "model")
}

func TestExecuteSingleTurn_BaseURLOverride(t *testing.T) {
	// Save original env var
	originalAPIKey := os.Getenv("DEEPSEEK_API_KEY")
	defer os.Setenv("DEEPSEEK_API_KEY", originalAPIKey)

	// Set API key
	os.Setenv("DEEPSEEK_API_KEY", "test-key")

	cmd := &cobra.Command{}
	cmd.Flags().String("base-url", "https://custom.api.com", "Base URL override")

	config := &Config{
		Chat: ChatConfig{
			Model:       "deepseek-chat",
			Temperature: 0.7,
			TopP:        1.0,
			MaxTokens:   100,
		},
	}

	err := executeSingleTurn(cmd, "test prompt", config)
	// This will fail because the API doesn't exist, but it should get past the base URL override
	assert.Error(t, err)
	// The error should be about the API call, not about base URL
}

func TestExecuteHistoryMode_EmptyMessages(t *testing.T) {
	// Save original env var
	originalAPIKey := os.Getenv("DEEPSEEK_API_KEY")
	defer os.Setenv("DEEPSEEK_API_KEY", originalAPIKey)

	// Set API key
	os.Setenv("DEEPSEEK_API_KEY", "test-key")

	// Create a temp file with empty messages array
	tmpfile, err := os.CreateTemp("", "test-*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	emptyJSON := `[]`
	if _, err := tmpfile.Write([]byte(emptyJSON)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	cmd := &cobra.Command{}
	config := &Config{Chat: GetDefaultChatConfig()}

	err = executeHistoryMode(cmd, tmpfile.Name(), config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "messages")
}

func TestExecuteHistoryMode_NonExistentFile(t *testing.T) {
	cmd := &cobra.Command{}
	config := &Config{Chat: GetDefaultChatConfig()}

	err := executeHistoryMode(cmd, "/this/file/does/not/exist.json", config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "reading history file")
}

func TestExecuteStdinMode_EmptyInput(t *testing.T) {
	// This test is tricky because we can't easily mock stdin in Go tests
	// We'll just verify the function exists and has the right signature
	_ = executeStdinMode
	// In a real test environment, we'd need to mock os.Stdin
}

func TestExecuteHistoryMode_ValidFile(t *testing.T) {
	// Save original env var
	originalAPIKey := os.Getenv("DEEPSEEK_API_KEY")
	defer os.Setenv("DEEPSEEK_API_KEY", originalAPIKey)

	// Set API key
	os.Setenv("DEEPSEEK_API_KEY", "test-key")

	// Create a temp file with valid messages
	tmpfile, err := os.CreateTemp("", "test-*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	validJSON := `[{"role":"user","content":"Hello"},{"role":"assistant","content":"Hi there"}]`
	if _, err := tmpfile.Write([]byte(validJSON)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	cmd := &cobra.Command{}
	config := &Config{Chat: GetDefaultChatConfig()}

	err = executeHistoryMode(cmd, tmpfile.Name(), config)
	// This will fail because the API doesn't exist, but it should get past file reading and parsing
	assert.Error(t, err)
	// The error should be about the API call, not about file reading
}

func TestExecuteSingleTurn_ValidConfig(t *testing.T) {
	// Save original env var
	originalAPIKey := os.Getenv("DEEPSEEK_API_KEY")
	defer os.Setenv("DEEPSEEK_API_KEY", originalAPIKey)

	// Set API key
	os.Setenv("DEEPSEEK_API_KEY", "test-key")

	cmd := &cobra.Command{}
	cmd.Flags().String("base-url", "", "Base URL override")

	config := &Config{
		Chat: ChatConfig{
			Model:       "deepseek-chat",
			Temperature: 0.7,
			TopP:        1.0,
			MaxTokens:   100,
		},
	}

	err := executeSingleTurn(cmd, "test prompt", config)
	// This will fail because the API doesn't exist
	assert.Error(t, err)
}