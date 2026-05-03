package main

import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// mockClient implements APIClientIface for testing
type mockClient struct {
	doFunc              func(method, path string, body interface{}) ([]byte, error)
	streamChatFunc      func(req *ChatRequest) error
	streamFIMFunc       func(req *FIMRequest) error
}

func (m *mockClient) do(method, path string, body interface{}) ([]byte, error) {
	if m.doFunc != nil {
		return m.doFunc(method, path, body)
	}
	return nil, nil
}

func (m *mockClient) streamChatCompletion(req *ChatRequest) error {
	if m.streamChatFunc != nil {
		return m.streamChatFunc(req)
	}
	return nil
}

func (m *mockClient) streamFIMCompletion(req *FIMRequest) error {
	if m.streamFIMFunc != nil {
		return m.streamFIMFunc(req)
	}
	return nil
}

func TestMustGetString(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().String("test-flag", "default", "test flag")

	// Test default value
	_ = cmd.ParseFlags([]string{})
	result := mustGetString(cmd, "test-flag")
	if result != "default" {
		t.Errorf("mustGetString() = %q, want %q", result, "default")
	}

	// Test set value
	_ = cmd.ParseFlags([]string{"--test-flag", "custom"})
	result = mustGetString(cmd, "test-flag")
	if result != "custom" {
		t.Errorf("mustGetString() = %q, want %q", result, "custom")
	}
}

func TestMustGetBool(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().Bool("test-bool", false, "test bool flag")

	// Test default value
	_ = cmd.ParseFlags([]string{})
	result := mustGetBool(cmd, "test-bool")
	if result != false {
		t.Errorf("mustGetBool() = %v, want false", result)
	}

	// Test set to true
	_ = cmd.ParseFlags([]string{"--test-bool"})
	result = mustGetBool(cmd, "test-bool")
	if result != true {
		t.Errorf("mustGetBool() = %v, want true", result)
	}
}

func TestMustGetInt(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().Int("test-int", 0, "test int flag")

	// Test default value
	_ = cmd.ParseFlags([]string{})
	result := mustGetInt(cmd, "test-int")
	if result != 0 {
		t.Errorf("mustGetInt() = %d, want 0", result)
	}

	// Test set value
	_ = cmd.ParseFlags([]string{"--test-int", "42"})
	result = mustGetInt(cmd, "test-int")
	if result != 42 {
		t.Errorf("mustGetInt() = %d, want 42", result)
	}
}

func TestMustGetFloat64(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().Float64("test-float", 1.0, "test float flag")

	// Test default value
	_ = cmd.ParseFlags([]string{})
	result := mustGetFloat64(cmd, "test-float")
	if result != 1.0 {
		t.Errorf("mustGetFloat64() = %f, want 1.0", result)
	}

	// Test set value
	_ = cmd.ParseFlags([]string{"--test-float", "0.5"})
	result = mustGetFloat64(cmd, "test-float")
	if result != 0.5 {
		t.Errorf("mustGetFloat64() = %f, want 0.5", result)
	}
}

func TestMustGetStringSlice(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().StringSlice("test-slice", []string{}, "test slice flag")

	// Test default value
	_ = cmd.ParseFlags([]string{})
	result := mustGetStringSlice(cmd, "test-slice")
	if len(result) != 0 {
		t.Errorf("mustGetStringSlice() length = %d, want 0", len(result))
	}

	// Test set value
	_ = cmd.ParseFlags([]string{"--test-slice", "a,b,c"})
	result = mustGetStringSlice(cmd, "test-slice")
	if len(result) != 3 || result[0] != "a" || result[2] != "c" {
		t.Errorf("mustGetStringSlice() = %v, want [a b c]", result)
	}
}

func TestGetVersion_WithVersion(t *testing.T) {
	// Save old version
	oldVersion := version
	defer func() { version = oldVersion }()
	
	version = "2.0.0"
	result := getVersion()
	if result != "2.0.0" {
		t.Errorf("getVersion() = %q, want %q", result, "2.0.0")
	}
}

func TestBuildChatRequest(t *testing.T) {
	// Test basic request
	cmd1 := &cobra.Command{}
	cmd1.Flags().String("model", "deepseek-v4-pro", "")
	cmd1.Flags().String("system", "", "")
	cmd1.Flags().String("user", "", "")
	cmd1.Flags().String("assistant", "", "")
	cmd1.Flags().String("thinking", "enabled", "")
	cmd1.Flags().String("reasoning-effort", "high", "")
	cmd1.Flags().Bool("stream", false, "")
	cmd1.Flags().Float64("temperature", 1.0, "")
	cmd1.Flags().Float64("top-p", 1.0, "")
	cmd1.Flags().Int("max-tokens", 0, "")
	cmd1.Flags().Float64("frequency-penalty", 0.0, "")
	cmd1.Flags().Float64("presence-penalty", 0.0, "")
	cmd1.Flags().Bool("json-mode", false, "")
	cmd1.Flags().StringSlice("stop", []string{}, "")
	cmd1.Flags().Bool("include-usage", false, "")
	cmd1.Flags().String("tools", "", "")
	cmd1.Flags().String("tool-choice", "auto", "")
	cmd1.Flags().Bool("logprobs", false, "")
	cmd1.Flags().Int("top-logprobs", 0, "")
	cmd1.Flags().Bool("prefix-completion", false, "")
	
	_ = cmd1.ParseFlags([]string{"--user", "Hello"})
	req, err := buildChatRequest(cmd1)
	if err != nil {
		t.Fatalf("buildChatRequest() error = %v", err)
	}
	if len(req.Messages) != 1 || req.Messages[0].Content != "Hello" {
		t.Errorf("buildChatRequest() messages incorrect")
	}

	// Test with system message
	cmd2 := &cobra.Command{}
	cmd2.Flags().String("model", "deepseek-v4-pro", "")
	cmd2.Flags().String("system", "", "")
	cmd2.Flags().String("user", "", "")
	cmd2.Flags().String("assistant", "", "")
	cmd2.Flags().String("thinking", "enabled", "")
	cmd2.Flags().String("reasoning-effort", "high", "")
	cmd2.Flags().Bool("stream", false, "")
	cmd2.Flags().Float64("temperature", 1.0, "")
	cmd2.Flags().Float64("top-p", 1.0, "")
	cmd2.Flags().Int("max-tokens", 0, "")
	cmd2.Flags().Float64("frequency-penalty", 0.0, "")
	cmd2.Flags().Float64("presence-penalty", 0.0, "")
	cmd2.Flags().Bool("json-mode", false, "")
	cmd2.Flags().StringSlice("stop", []string{}, "")
	cmd2.Flags().Bool("include-usage", false, "")
	cmd2.Flags().String("tools", "", "")
	cmd2.Flags().String("tool-choice", "auto", "")
	cmd2.Flags().Bool("logprobs", false, "")
	cmd2.Flags().Int("top-logprobs", 0, "")
	cmd2.Flags().Bool("prefix-completion", false, "")
	
	_ = cmd2.ParseFlags([]string{"--system", "You are helpful", "--user", "Hi"})
	req, err = buildChatRequest(cmd2)
	if err != nil {
		t.Fatalf("buildChatRequest() error = %v", err)
	}
	if len(req.Messages) != 2 || req.Messages[0].Role != "system" {
		t.Errorf("buildChatRequest() with system message incorrect")
	}

	// Test with temperature
	cmd3 := &cobra.Command{}
	cmd3.Flags().String("model", "deepseek-v4-pro", "")
	cmd3.Flags().String("system", "", "")
	cmd3.Flags().String("user", "", "")
	cmd3.Flags().String("assistant", "", "")
	cmd3.Flags().String("thinking", "enabled", "")
	cmd3.Flags().String("reasoning-effort", "high", "")
	cmd3.Flags().Bool("stream", false, "")
	cmd3.Flags().Float64("temperature", 1.0, "")
	cmd3.Flags().Float64("top-p", 1.0, "")
	cmd3.Flags().Int("max-tokens", 0, "")
	cmd3.Flags().Float64("frequency-penalty", 0.0, "")
	cmd3.Flags().Float64("presence-penalty", 0.0, "")
	cmd3.Flags().Bool("json-mode", false, "")
	cmd3.Flags().StringSlice("stop", []string{}, "")
	cmd3.Flags().Bool("include-usage", false, "")
	cmd3.Flags().String("tools", "", "")
	cmd3.Flags().String("tool-choice", "auto", "")
	cmd3.Flags().Bool("logprobs", false, "")
	cmd3.Flags().Int("top-logprobs", 0, "")
	cmd3.Flags().Bool("prefix-completion", false, "")
	
	_ = cmd3.ParseFlags([]string{"--user", "test", "--temperature", "0.5"})
	req, err = buildChatRequest(cmd3)
	if err != nil {
		t.Fatalf("buildChatRequest() error = %v", err)
	}
	if req.Temperature == nil || *req.Temperature != 0.5 {
		t.Errorf("buildChatRequest() temperature not set correctly")
	}

	// Test with JSON mode
	cmd4 := &cobra.Command{}
	cmd4.Flags().String("model", "deepseek-v4-pro", "")
	cmd4.Flags().String("system", "", "")
	cmd4.Flags().String("user", "", "")
	cmd4.Flags().String("assistant", "", "")
	cmd4.Flags().String("thinking", "enabled", "")
	cmd4.Flags().String("reasoning-effort", "high", "")
	cmd4.Flags().Bool("stream", false, "")
	cmd4.Flags().Float64("temperature", 1.0, "")
	cmd4.Flags().Float64("top-p", 1.0, "")
	cmd4.Flags().Int("max-tokens", 0, "")
	cmd4.Flags().Float64("frequency-penalty", 0.0, "")
	cmd4.Flags().Float64("presence-penalty", 0.0, "")
	cmd4.Flags().Bool("json-mode", false, "")
	cmd4.Flags().StringSlice("stop", []string{}, "")
	cmd4.Flags().Bool("include-usage", false, "")
	cmd4.Flags().String("tools", "", "")
	cmd4.Flags().String("tool-choice", "auto", "")
	cmd4.Flags().Bool("logprobs", false, "")
	cmd4.Flags().Int("top-logprobs", 0, "")
	cmd4.Flags().Bool("prefix-completion", false, "")
	
	_ = cmd4.ParseFlags([]string{"--user", "test", "--json-mode"})
	req, err = buildChatRequest(cmd4)
	if err != nil {
		t.Fatalf("buildChatRequest() error = %v", err)
	}
	if req.ResponseFormat == nil || req.ResponseFormat.Type != "json_object" {
		t.Errorf("buildChatRequest() json mode not set correctly")
	}

	// Test no messages error - use fresh command
	cmd5 := &cobra.Command{}
	cmd5.Flags().String("model", "deepseek-v4-pro", "")
	cmd5.Flags().String("system", "", "")
	cmd5.Flags().String("user", "", "")
	cmd5.Flags().String("assistant", "", "")
	cmd5.Flags().String("thinking", "enabled", "")
	cmd5.Flags().String("reasoning-effort", "high", "")
	cmd5.Flags().Bool("stream", false, "")
	cmd5.Flags().Float64("temperature", 1.0, "")
	cmd5.Flags().Float64("top-p", 1.0, "")
	cmd5.Flags().Int("max-tokens", 0, "")
	cmd5.Flags().Float64("frequency-penalty", 0.0, "")
	cmd5.Flags().Float64("presence-penalty", 0.0, "")
	cmd5.Flags().Bool("json-mode", false, "")
	cmd5.Flags().StringSlice("stop", []string{}, "")
	cmd5.Flags().Bool("include-usage", false, "")
	cmd5.Flags().String("tools", "", "")
	cmd5.Flags().String("tool-choice", "auto", "")
	cmd5.Flags().Bool("logprobs", false, "")
	cmd5.Flags().Int("top-logprobs", 0, "")
	cmd5.Flags().Bool("prefix-completion", false, "")
	
	_ = cmd5.ParseFlags([]string{})
	_, err = buildChatRequest(cmd5)
	if err == nil {
		t.Error("buildChatRequest() should error with no messages")
	}
}

func TestBuildFIMRequest(t *testing.T) {
	// Test basic request
	cmd1 := &cobra.Command{}
	cmd1.Flags().String("model", "deepseek-v4-pro", "")
	cmd1.Flags().String("prompt", "", "")
	cmd1.Flags().String("suffix", "", "")
	cmd1.Flags().Bool("stream", false, "")
	cmd1.Flags().Bool("echo", false, "")
	cmd1.Flags().Int("max-tokens", 0, "")
	cmd1.Flags().Float64("temperature", 0.2, "")
	cmd1.Flags().Float64("top-p", 1.0, "")
	cmd1.Flags().Float64("frequency-penalty", 0.0, "")
	cmd1.Flags().Float64("presence-penalty", 0.0, "")
	cmd1.Flags().StringSlice("stop", []string{}, "")
	cmd1.Flags().Bool("include-usage", false, "")
	cmd1.Flags().Int("logprobs", 0, "")

	_ = cmd1.ParseFlags([]string{"--prompt", "func main() {"})
	req, err := buildFIMRequest(cmd1)
	if err != nil {
		t.Fatalf("buildFIMRequest() error = %v", err)
	}
	if req.Prompt != "func main() {" {
		t.Errorf("buildFIMRequest() prompt incorrect")
	}

	// Test with suffix
	cmd2 := &cobra.Command{}
	cmd2.Flags().String("model", "deepseek-v4-pro", "")
	cmd2.Flags().String("prompt", "", "")
	cmd2.Flags().String("suffix", "", "")
	cmd2.Flags().Bool("stream", false, "")
	cmd2.Flags().Bool("echo", false, "")
	cmd2.Flags().Int("max-tokens", 0, "")
	cmd2.Flags().Float64("temperature", 0.2, "")
	cmd2.Flags().Float64("top-p", 1.0, "")
	cmd2.Flags().Float64("frequency-penalty", 0.0, "")
	cmd2.Flags().Float64("presence-penalty", 0.0, "")
	cmd2.Flags().StringSlice("stop", []string{}, "")
	cmd2.Flags().Bool("include-usage", false, "")
	cmd2.Flags().Int("logprobs", 0, "")

	_ = cmd2.ParseFlags([]string{"--prompt", "func main() {", "--suffix", "}"})
	req, err = buildFIMRequest(cmd2)
	if err != nil {
		t.Fatalf("buildFIMRequest() error = %v", err)
	}
	if req.Suffix == nil || *req.Suffix != "}" {
		t.Errorf("buildFIMRequest() suffix not set correctly")
	}

	// Test no prompt error - fresh command
	cmd3 := &cobra.Command{}
	cmd3.Flags().String("model", "deepseek-v4-pro", "")
	cmd3.Flags().String("prompt", "", "")
	cmd3.Flags().String("suffix", "", "")
	cmd3.Flags().Bool("stream", false, "")
	cmd3.Flags().Bool("echo", false, "")
	cmd3.Flags().Int("max-tokens", 0, "")
	cmd3.Flags().Float64("temperature", 0.2, "")
	cmd3.Flags().Float64("top-p", 1.0, "")
	cmd3.Flags().Float64("frequency-penalty", 0.0, "")
	cmd3.Flags().Float64("presence-penalty", 0.0, "")
	cmd3.Flags().StringSlice("stop", []string{}, "")
	cmd3.Flags().Bool("include-usage", false, "")
	cmd3.Flags().Int("logprobs", 0, "")

	_ = cmd3.ParseFlags([]string{})
	_, err = buildFIMRequest(cmd3)
	if err == nil {
		t.Error("buildFIMRequest() should error with no prompt")
	}
}

func TestHasStdinData(t *testing.T) {
	// Save original stdin
	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()

	// Create a pipe to simulate stdin with data
	r, w, _ := os.Pipe()
	os.Stdin = r
	
	go func() {
		_, _ = w.WriteString("test input")
		_ = w.Close()
	}()

	result := hasStdinData()
	if !result {
		t.Error("hasStdinData() should return true when stdin has data")
	}
}

func TestHasStdinData_NoData(t *testing.T) {
	// Save original stdin
	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()

	// Create a pipe - even without writing, it's still a pipe (not a terminal)
	// So hasStdinData will return true
	// We can't easily test the "false" case without restoring stdin to a terminal
	r, w, _ := os.Pipe()
	os.Stdin = r
	
	// Close the write end so we don't leak file descriptors
	// But keep the read end open
	defer func() { _ = r.Close() }()
	defer func() { _ = w.Close() }()
	
	// The function should not panic
	result := hasStdinData()
	// We can't assert the value since it depends on the pipe state
	_ = result
}

func TestBuildChatRequest_WithTools(t *testing.T) {
	cmd := &cobra.Command{}
	
	// Add flags
	cmd.Flags().String("model", "deepseek-v4-pro", "")
	cmd.Flags().String("system", "", "")
	cmd.Flags().String("user", "test", "")
	cmd.Flags().String("assistant", "", "")
	cmd.Flags().String("thinking", "enabled", "")
	cmd.Flags().String("reasoning-effort", "high", "")
	cmd.Flags().Bool("stream", false, "")
	cmd.Flags().Float64("temperature", 1.0, "")
	cmd.Flags().Float64("top-p", 1.0, "")
	cmd.Flags().Int("max-tokens", 0, "")
	cmd.Flags().Float64("frequency-penalty", 0.0, "")
	cmd.Flags().Float64("presence-penalty", 0.0, "")
	cmd.Flags().Bool("json-mode", false, "")
	cmd.Flags().StringSlice("stop", []string{}, "")
	cmd.Flags().Bool("include-usage", false, "")
	cmd.Flags().String("tools", "", "")
	cmd.Flags().String("tool-choice", "auto", "")
	cmd.Flags().Bool("logprobs", false, "")
	cmd.Flags().Int("top-logprobs", 0, "")
	cmd.Flags().Bool("prefix-completion", false, "")

	toolsJSON := `[{"type":"function","function":{"name":"get_weather","parameters":{"type":"object"}}}]`
	_ = cmd.ParseFlags([]string{"--user", "weather", "--tools", toolsJSON})
	req, err := buildChatRequest(cmd)
	if err != nil {
		t.Fatalf("buildChatRequest() with tools error = %v", err)
	}
	if len(req.Tools) != 1 || req.Tools[0].Function.Name != "get_weather" {
		t.Errorf("buildChatRequest() tools not set correctly")
	}
}

func TestBuildChatRequest_WithToolChoice(t *testing.T) {
	cmd := &cobra.Command{}
	
	// Add flags
	cmd.Flags().String("model", "deepseek-v4-pro", "")
	cmd.Flags().String("system", "", "")
	cmd.Flags().String("user", "test", "")
	cmd.Flags().String("assistant", "", "")
	cmd.Flags().String("thinking", "enabled", "")
	cmd.Flags().String("reasoning-effort", "high", "")
	cmd.Flags().Bool("stream", false, "")
	cmd.Flags().Float64("temperature", 1.0, "")
	cmd.Flags().Float64("top-p", 1.0, "")
	cmd.Flags().Int("max-tokens", 0, "")
	cmd.Flags().Float64("frequency-penalty", 0.0, "")
	cmd.Flags().Float64("presence-penalty", 0.0, "")
	cmd.Flags().Bool("json-mode", false, "")
	cmd.Flags().StringSlice("stop", []string{}, "")
	cmd.Flags().Bool("include-usage", false, "")
	cmd.Flags().String("tools", "", "")
	cmd.Flags().String("tool-choice", "auto", "")
	cmd.Flags().Bool("logprobs", false, "")
	cmd.Flags().Int("top-logprobs", 0, "")
	cmd.Flags().Bool("prefix-completion", false, "")

	// Test "none" tool choice
	_ = cmd.ParseFlags([]string{"--user", "test", "--tool-choice", "none"})
	req, err := buildChatRequest(cmd)
	if err != nil {
		t.Fatalf("buildChatRequest() error = %v", err)
	}
	if req.ToolChoice != "none" {
		t.Errorf("buildChatRequest() tool-choice not set correctly")
	}

	// Test JSON tool choice
	_ = cmd.ParseFlags([]string{"--user", "test", "--tool-choice", `{"type":"function","function":{"name":"get_weather"}}`})
	req, err = buildChatRequest(cmd)
	if err != nil {
		t.Fatalf("buildChatRequest() with JSON tool-choice error = %v", err)
	}
	if req.ToolChoice == nil {
		t.Error("buildChatRequest() tool-choice JSON not set")
	}
}

func TestBuildChatRequest_WithLogprobs(t *testing.T) {
	cmd := &cobra.Command{}
	
	// Add flags
	cmd.Flags().String("model", "deepseek-v4-pro", "")
	cmd.Flags().String("system", "", "")
	cmd.Flags().String("user", "test", "")
	cmd.Flags().String("assistant", "", "")
	cmd.Flags().String("thinking", "enabled", "")
	cmd.Flags().String("reasoning-effort", "high", "")
	cmd.Flags().Bool("stream", false, "")
	cmd.Flags().Float64("temperature", 1.0, "")
	cmd.Flags().Float64("top-p", 1.0, "")
	cmd.Flags().Int("max-tokens", 0, "")
	cmd.Flags().Float64("frequency-penalty", 0.0, "")
	cmd.Flags().Float64("presence-penalty", 0.0, "")
	cmd.Flags().Bool("json-mode", false, "")
	cmd.Flags().StringSlice("stop", []string{}, "")
	cmd.Flags().Bool("include-usage", false, "")
	cmd.Flags().String("tools", "", "")
	cmd.Flags().String("tool-choice", "auto", "")
	cmd.Flags().Bool("logprobs", false, "")
	cmd.Flags().Int("top-logprobs", 0, "")
	cmd.Flags().Bool("prefix-completion", false, "")

	_ = cmd.ParseFlags([]string{"--user", "test", "--logprobs", "--top-logprobs", "5"})
	req, err := buildChatRequest(cmd)
	if err != nil {
		t.Fatalf("buildChatRequest() error = %v", err)
	}
	if req.Logprobs == nil || !*req.Logprobs {
		t.Errorf("buildChatRequest() logprobs not set correctly")
	}
	if req.TopLogprobs == nil || *req.TopLogprobs != 5 {
		t.Errorf("buildChatRequest() top-logprobs not set correctly")
	}
}

func TestBuildChatRequest_WithStopSequences(t *testing.T) {
	cmd := &cobra.Command{}
	
	// Add flags
	cmd.Flags().String("model", "deepseek-v4-pro", "")
	cmd.Flags().String("system", "", "")
	cmd.Flags().String("user", "test", "")
	cmd.Flags().String("assistant", "", "")
	cmd.Flags().String("thinking", "enabled", "")
	cmd.Flags().String("reasoning-effort", "high", "")
	cmd.Flags().Bool("stream", false, "")
	cmd.Flags().Float64("temperature", 1.0, "")
	cmd.Flags().Float64("top-p", 1.0, "")
	cmd.Flags().Int("max-tokens", 0, "")
	cmd.Flags().Float64("frequency-penalty", 0.0, "")
	cmd.Flags().Float64("presence-penalty", 0.0, "")
	cmd.Flags().Bool("json-mode", false, "")
	cmd.Flags().StringSlice("stop", []string{}, "")
	cmd.Flags().Bool("include-usage", false, "")
	cmd.Flags().String("tools", "", "")
	cmd.Flags().String("tool-choice", "auto", "")
	cmd.Flags().Bool("logprobs", false, "")
	cmd.Flags().Int("top-logprobs", 0, "")
	cmd.Flags().Bool("prefix-completion", false, "")

	// Test single stop sequence
	_ = cmd.ParseFlags([]string{"--user", "test", "--stop", "STOP"})
	req, err := buildChatRequest(cmd)
	if err != nil {
		t.Fatalf("buildChatRequest() error = %v", err)
	}
	if req.Stop != "STOP" {
		t.Errorf("buildChatRequest() single stop not set correctly")
	}
}

func TestBuildFIMRequest_WithLogprobs(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().String("model", "deepseek-v4-pro", "")
	cmd.Flags().String("prompt", "test", "")
	cmd.Flags().String("suffix", "", "")
	cmd.Flags().Bool("stream", false, "")
	cmd.Flags().Bool("echo", false, "")
	cmd.Flags().Int("max-tokens", 0, "")
	cmd.Flags().Float64("temperature", 0.2, "")
	cmd.Flags().Float64("top-p", 1.0, "")
	cmd.Flags().Float64("frequency-penalty", 0.0, "")
	cmd.Flags().Float64("presence-penalty", 0.0, "")
	cmd.Flags().StringSlice("stop", []string{}, "")
	cmd.Flags().Bool("include-usage", false, "")
	cmd.Flags().Int("logprobs", 0, "")

	_ = cmd.ParseFlags([]string{"--prompt", "test", "--logprobs", "5"})
	req, err := buildFIMRequest(cmd)
	if err != nil {
		t.Fatalf("buildFIMRequest() error = %v", err)
	}
	if req.Logprobs == nil || *req.Logprobs != 5 {
		t.Errorf("buildFIMRequest() logprobs not set correctly")
	}
}

func TestBuildFIMRequest_WithStreamAndUsage(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().String("model", "deepseek-v4-pro", "")
	cmd.Flags().String("prompt", "test", "")
	cmd.Flags().String("suffix", "", "")
	cmd.Flags().Bool("stream", false, "")
	cmd.Flags().Bool("echo", false, "")
	cmd.Flags().Int("max-tokens", 0, "")
	cmd.Flags().Float64("temperature", 0.2, "")
	cmd.Flags().Float64("top-p", 1.0, "")
	cmd.Flags().Float64("frequency-penalty", 0.0, "")
	cmd.Flags().Float64("presence-penalty", 0.0, "")
	cmd.Flags().StringSlice("stop", []string{}, "")
	cmd.Flags().Bool("include-usage", false, "")
	cmd.Flags().Int("logprobs", 0, "")

	_ = cmd.ParseFlags([]string{"--prompt", "test", "--stream", "--include-usage"})
	req, err := buildFIMRequest(cmd)
	if err != nil {
		t.Fatalf("buildFIMRequest() error = %v", err)
	}
	if !req.Stream {
		t.Error("buildFIMRequest() stream not set")
	}
	if req.StreamOptions == nil || !req.StreamOptions.IncludeUsage {
		t.Errorf("buildFIMRequest() include-usage not set correctly")
	}
}

func TestBuildFIMRequest_WithStopSequences(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().String("model", "deepseek-v4-pro", "")
	cmd.Flags().String("prompt", "test", "")
	cmd.Flags().String("suffix", "", "")
	cmd.Flags().Bool("stream", false, "")
	cmd.Flags().Bool("echo", false, "")
	cmd.Flags().Int("max-tokens", 0, "")
	cmd.Flags().Float64("temperature", 0.2, "")
	cmd.Flags().Float64("top-p", 1.0, "")
	cmd.Flags().Float64("frequency-penalty", 0.0, "")
	cmd.Flags().Float64("presence-penalty", 0.0, "")
	cmd.Flags().StringSlice("stop", []string{}, "")
	cmd.Flags().Bool("include-usage", false, "")
	cmd.Flags().Int("logprobs", 0, "")

	_ = cmd.ParseFlags([]string{"--prompt", "test", "--stop", "STOP"})
	req, err := buildFIMRequest(cmd)
	if err != nil {
		t.Fatalf("buildFIMRequest() error = %v", err)
	}
	if req.Stop != "STOP" {
		t.Errorf("buildFIMRequest() stop not set correctly")
	}
}

func TestBuildFIMRequest_WithPenalties(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().String("model", "deepseek-v4-pro", "")
	cmd.Flags().String("prompt", "test", "")
	cmd.Flags().String("suffix", "", "")
	cmd.Flags().Bool("stream", false, "")
	cmd.Flags().Bool("echo", false, "")
	cmd.Flags().Int("max-tokens", 0, "")
	cmd.Flags().Float64("temperature", 0.2, "")
	cmd.Flags().Float64("top-p", 1.0, "")
	cmd.Flags().Float64("frequency-penalty", 0.0, "")
	cmd.Flags().Float64("presence-penalty", 0.0, "")
	cmd.Flags().StringSlice("stop", []string{}, "")
	cmd.Flags().Bool("include-usage", false, "")
	cmd.Flags().Int("logprobs", 0, "")

	_ = cmd.ParseFlags([]string{"--prompt", "test", "--frequency-penalty", "1.0", "--presence-penalty", "-0.5"})
	req, err := buildFIMRequest(cmd)
	if err != nil {
		t.Fatalf("buildFIMRequest() error = %v", err)
	}
	if req.FrequencyPenalty == nil || *req.FrequencyPenalty != 1.0 {
		t.Errorf("buildFIMRequest() frequency-penalty not set correctly")
	}
	if req.PresencePenalty == nil || *req.PresencePenalty != -0.5 {
		t.Errorf("buildFIMRequest() presence-penalty not set correctly")
	}
}

func TestBuildFIMRequest_WithTemperatureAndTopP(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().String("model", "deepseek-v4-pro", "")
	cmd.Flags().String("prompt", "test", "")
	cmd.Flags().String("suffix", "", "")
	cmd.Flags().Bool("stream", false, "")
	cmd.Flags().Bool("echo", false, "")
	cmd.Flags().Int("max-tokens", 0, "")
	cmd.Flags().Float64("temperature", 0.2, "")
	cmd.Flags().Float64("top-p", 1.0, "")
	cmd.Flags().Float64("frequency-penalty", 0.0, "")
	cmd.Flags().Float64("presence-penalty", 0.0, "")
	cmd.Flags().StringSlice("stop", []string{}, "")
	cmd.Flags().Bool("include-usage", false, "")
	cmd.Flags().Int("logprobs", 0, "")

	// Test with non-default temperature
	_ = cmd.ParseFlags([]string{"--prompt", "test", "--temperature", "0.5"})
	req, err := buildFIMRequest(cmd)
	if err != nil {
		t.Fatalf("buildFIMRequest() error = %v", err)
	}
	if req.Temperature == nil || *req.Temperature != 0.5 {
		t.Errorf("buildFIMRequest() temperature not set correctly")
	}

	// Test with non-default top-p
	_ = cmd.ParseFlags([]string{"--prompt", "test", "--top-p", "0.8"})
	req, err = buildFIMRequest(cmd)
	if err != nil {
		t.Fatalf("buildFIMRequest() error = %v", err)
	}
	if req.TopP == nil || *req.TopP != 0.8 {
		t.Errorf("buildFIMRequest() top-p not set correctly")
	}
	}

func TestBuildChatRequest_WithAssistantMessage(t *testing.T) {
	cmd := &cobra.Command{}
	
	// Add flags
	cmd.Flags().String("model", "deepseek-v4-pro", "")
	cmd.Flags().String("system", "", "")
	cmd.Flags().String("user", "", "")
	cmd.Flags().String("assistant", "previous response", "")
	cmd.Flags().String("thinking", "enabled", "")
	cmd.Flags().String("reasoning-effort", "high", "")
	cmd.Flags().Bool("stream", false, "")
	cmd.Flags().Float64("temperature", 1.0, "")
	cmd.Flags().Float64("top-p", 1.0, "")
	cmd.Flags().Int("max-tokens", 0, "")
	cmd.Flags().Float64("frequency-penalty", 0.0, "")
	cmd.Flags().Float64("presence-penalty", 0.0, "")
	cmd.Flags().Bool("json-mode", false, "")
	cmd.Flags().StringSlice("stop", []string{}, "")
	cmd.Flags().Bool("include-usage", false, "")
	cmd.Flags().String("tools", "", "")
	cmd.Flags().String("tool-choice", "auto", "")
	cmd.Flags().Bool("logprobs", false, "")
	cmd.Flags().Int("top-logprobs", 0, "")
	cmd.Flags().Bool("prefix-completion", false, "")

	_ = cmd.ParseFlags([]string{"--assistant", "previous", "--prefix-completion"})
	req, err := buildChatRequest(cmd)
	if err != nil {
		t.Fatalf("buildChatRequest() error = %v", err)
	}
	if len(req.Messages) != 1 || req.Messages[0].Role != "assistant" {
		t.Errorf("buildChatRequest() assistant message not set correctly")
	}
	if req.Messages[0].Prefix == nil || !*req.Messages[0].Prefix {
		t.Errorf("buildChatRequest() prefix not set correctly")
	}
}

func TestBuildChatRequest_WithStreamAndUsage(t *testing.T) {
	cmd := &cobra.Command{}
	
	// Add flags
	cmd.Flags().String("model", "deepseek-v4-pro", "")
	cmd.Flags().String("system", "", "")
	cmd.Flags().String("user", "test", "")
	cmd.Flags().String("assistant", "", "")
	cmd.Flags().String("thinking", "enabled", "")
	cmd.Flags().String("reasoning-effort", "high", "")
	cmd.Flags().Bool("stream", false, "")
	cmd.Flags().Float64("temperature", 1.0, "")
	cmd.Flags().Float64("top-p", 1.0, "")
	cmd.Flags().Int("max-tokens", 0, "")
	cmd.Flags().Float64("frequency-penalty", 0.0, "")
	cmd.Flags().Float64("presence-penalty", 0.0, "")
	cmd.Flags().Bool("json-mode", false, "")
	cmd.Flags().StringSlice("stop", []string{}, "")
	cmd.Flags().Bool("include-usage", false, "")
	cmd.Flags().String("tools", "", "")
	cmd.Flags().String("tool-choice", "auto", "")
	cmd.Flags().Bool("logprobs", false, "")
	cmd.Flags().Int("top-logprobs", 0, "")
	cmd.Flags().Bool("prefix-completion", false, "")

	_ = cmd.ParseFlags([]string{"--user", "test", "--stream", "--include-usage"})
	req, err := buildChatRequest(cmd)
	if err != nil {
		t.Fatalf("buildChatRequest() error = %v", err)
	}
	if !req.Stream {
		t.Error("buildChatRequest() stream not set")
	}
	if req.StreamOptions == nil || !req.StreamOptions.IncludeUsage {
		t.Errorf("buildChatRequest() include-usage not set correctly")
	}
}

func TestBuildChatRequest_WithPenalties(t *testing.T) {
	cmd := &cobra.Command{}
	
	// Add flags
	cmd.Flags().String("model", "deepseek-v4-pro", "")
	cmd.Flags().String("system", "", "")
	cmd.Flags().String("user", "test", "")
	cmd.Flags().String("assistant", "", "")
	cmd.Flags().String("thinking", "enabled", "")
	cmd.Flags().String("reasoning-effort", "high", "")
	cmd.Flags().Bool("stream", false, "")
	cmd.Flags().Float64("temperature", 1.0, "")
	cmd.Flags().Float64("top-p", 1.0, "")
	cmd.Flags().Int("max-tokens", 0, "")
	cmd.Flags().Float64("frequency-penalty", 0.0, "")
	cmd.Flags().Float64("presence-penalty", 0.0, "")
	cmd.Flags().Bool("json-mode", false, "")
	cmd.Flags().StringSlice("stop", []string{}, "")
	cmd.Flags().Bool("include-usage", false, "")
	cmd.Flags().String("tools", "", "")
	cmd.Flags().String("tool-choice", "auto", "")
	cmd.Flags().Bool("logprobs", false, "")
	cmd.Flags().Int("top-logprobs", 0, "")
	cmd.Flags().Bool("prefix-completion", false, "")

	_ = cmd.ParseFlags([]string{"--user", "test", "--frequency-penalty", "1.5", "--presence-penalty", "-0.5"})
	req, err := buildChatRequest(cmd)
	if err != nil {
		t.Fatalf("buildChatRequest() error = %v", err)
	}
	if req.FrequencyPenalty == nil || *req.FrequencyPenalty != 1.5 {
		t.Errorf("buildChatRequest() frequency-penalty not set correctly")
	}
	if req.PresencePenalty == nil || *req.PresencePenalty != -0.5 {
		t.Errorf("buildChatRequest() presence-penalty not set correctly")
	}
}

func TestRun(t *testing.T) {
	// Save original os.Args
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// Test with --version flag
	os.Args = []string{"deepseek", "--version"}
	err := run()
	if err != nil {
		t.Errorf("run() with --version error = %v", err)
	}
}

func TestRun_Help(t *testing.T) {
	// Save original os.Args
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// Test with --help flag
	os.Args = []string{"deepseek", "--help"}
	// This will print help and return nil (or an error)
	_ = run()
}

func TestRun_InvalidCommand(t *testing.T) {
	// Save original os.Args
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// Test with invalid command
	os.Args = []string{"deepseek", "invalidcommand"}
	err := run()
	if err == nil {
		t.Error("run() should error with invalid command")
	}
}

func TestLaunchTUI_NoAPIKey(t *testing.T) {
	// Save and restore env
	oldKey := os.Getenv("DEEPSEEK_API_KEY")
	defer func() { _ = os.Setenv("DEEPSEEK_API_KEY", oldKey) }()
	_ = os.Unsetenv("DEEPSEEK_API_KEY")

	cmd := &cobra.Command{}
	config := &Config{
		Chat: GetDefaultChatConfig(),
	}

	// This should return an error about missing API key
	err := launchTUI(cmd, config)
	if err == nil {
		t.Error("launchTUI() should error without API key")
	}
}

func TestHandleSingleTurn_NoAPIKey(t *testing.T) {
	cmd := &cobra.Command{}
	config := &Config{
		Chat: GetDefaultChatConfig(),
	}

	// This should return an error about missing API key
	err := handleSingleTurn(cmd, "test prompt", config)
	if err == nil {
		t.Error("handleSingleTurn() should error without API key")
	}
}

func TestHandleHistoryMode_FileNotExists(t *testing.T) {
	cmd := &cobra.Command{}
	config := &Config{
		Chat: GetDefaultChatConfig(),
	}

	// This should return an error about file not found
	err := handleHistoryMode(cmd, "/nonexistent/file.json", config)
	if err == nil {
		t.Error("handleHistoryMode() should error with nonexistent file")
	}
}

func TestHandleStdinMode_EmptyInput(t *testing.T) {
	// Save original stdin
	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()

	// Create empty pipe for stdin
	r, w, _ := os.Pipe()
	os.Stdin = r
	_ = w.Close()
	defer func() { _ = r.Close() }()

	cmd := &cobra.Command{}
	config := &Config{
		Chat: GetDefaultChatConfig(),
	}

	err := handleStdinMode(cmd, config)
	if err == nil {
		t.Error("handleStdinMode() should error with empty stdin")
	}
}

func TestExecuteStdinMode_EmptyInput(t *testing.T) {
	// Save original stdin
	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()

	r, w, _ := os.Pipe()
	os.Stdin = r
	_ = w.Close()

	cmd := &cobra.Command{}
	config := &Config{
		Chat: GetDefaultChatConfig(),
	}

	err := executeStdinMode(cmd, config, nil)
	if err == nil {
		t.Error("executeStdinMode() should error with empty stdin")
	}
}

func TestExecuteHistoryMode_FileNotExists(t *testing.T) {
	cmd := &cobra.Command{}
	config := &Config{
		Chat: GetDefaultChatConfig(),
	}

	err := executeHistoryMode(cmd, "/nonexistent/file.json", config, nil)
	if err == nil {
		t.Error("executeHistoryMode() should error with nonexistent file")
	}
}

func TestExecuteHistoryMode_InvalidJSON(t *testing.T) {
	// Create a temp file with invalid JSON
	tmpfile, err := os.CreateTemp("", "history*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()

	_, _ = tmpfile.WriteString("invalid json")
	_ = tmpfile.Close()

	cmd := &cobra.Command{}
	config := &Config{
		Chat: GetDefaultChatConfig(),
	}

	err = executeHistoryMode(cmd, tmpfile.Name(), config, nil)
	if err == nil {
		t.Error("executeHistoryMode() should error with invalid JSON")
	}
}

func TestBuildChatRequest_InvalidToolsJSON(t *testing.T) {
	cmd := &cobra.Command{}
	
	// Add flags
	cmd.Flags().String("model", "deepseek-v4-pro", "")
	cmd.Flags().String("system", "", "")
	cmd.Flags().String("user", "test", "")
	cmd.Flags().String("assistant", "", "")
	cmd.Flags().String("thinking", "enabled", "")
	cmd.Flags().String("reasoning-effort", "high", "")
	cmd.Flags().Bool("stream", false, "")
	cmd.Flags().Float64("temperature", 1.0, "")
	cmd.Flags().Float64("top-p", 1.0, "")
	cmd.Flags().Int("max-tokens", 0, "")
	cmd.Flags().Float64("frequency-penalty", 0.0, "")
	cmd.Flags().Float64("presence-penalty", 0.0, "")
	cmd.Flags().Bool("json-mode", false, "")
	cmd.Flags().StringSlice("stop", []string{}, "")
	cmd.Flags().Bool("include-usage", false, "")
	cmd.Flags().String("tools", "invalid json", "")
	cmd.Flags().String("tool-choice", "auto", "")
	cmd.Flags().Bool("logprobs", false, "")
	cmd.Flags().Int("top-logprobs", 0, "")
	cmd.Flags().Bool("prefix-completion", false, "")

	_ = cmd.ParseFlags([]string{"--user", "test", "--tools", "invalid"})
	_, err := buildChatRequest(cmd)
	if err == nil {
		t.Error("buildChatRequest() should error with invalid tools JSON")
	}
}

func TestBuildChatRequest_InvalidToolChoiceJSON(t *testing.T) {
	cmd := &cobra.Command{}
	
	// Add flags
	cmd.Flags().String("model", "deepseek-v4-pro", "")
	cmd.Flags().String("system", "", "")
	cmd.Flags().String("user", "test", "")
	cmd.Flags().String("assistant", "", "")
	cmd.Flags().String("thinking", "enabled", "")
	cmd.Flags().String("reasoning-effort", "high", "")
	cmd.Flags().Bool("stream", false, "")
	cmd.Flags().Float64("temperature", 1.0, "")
	cmd.Flags().Float64("top-p", 1.0, "")
	cmd.Flags().Int("max-tokens", 0, "")
	cmd.Flags().Float64("frequency-penalty", 0.0, "")
	cmd.Flags().Float64("presence-penalty", 0.0, "")
	cmd.Flags().Bool("json-mode", false, "")
	cmd.Flags().StringSlice("stop", []string{}, "")
	cmd.Flags().Bool("include-usage", false, "")
	cmd.Flags().String("tools", "", "")
	cmd.Flags().String("tool-choice", "auto", "")
	cmd.Flags().Bool("logprobs", false, "")
	cmd.Flags().Int("top-logprobs", 0, "")
	cmd.Flags().Bool("prefix-completion", false, "")

	// Use invalid JSON that starts with { but is malformed
	_ = cmd.ParseFlags([]string{"--user", "test", "--tool-choice", `{"type":"function"`})
	_, err := buildChatRequest(cmd)
	if err == nil {
		t.Error("buildChatRequest() should error with invalid tool-choice JSON")
	}
}

func TestExecuteChatCommand_WithMockClient(t *testing.T) {
	// Create mock client
	mock := &mockClient{
		doFunc: func(method, path string, body interface{}) ([]byte, error) {
			// Return a valid chat response
			return []byte(`{"choices":[{"message":{"content":"Hello from mock"}},"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}}`), nil
		},
	}

	cmd := &cobra.Command{}
	cmd.Flags().String("json", "", "JSON input")
	cmd.Flags().Bool("beta", false, "beta flag")
	cmd.Flags().String("base-url", "", "base url")
	cmd.Flags().Bool("cache", false, "cache flag")
	_ = cmd.ParseFlags([]string{"--json", `{"model":"deepseek-v4-pro","messages":[{"role":"user","content":"hi"}]}`})

	err := executeChatCommand(cmd, mock)
	if err != nil {
		t.Errorf("executeChatCommand() with mock client error = %v", err)
	}
}

func TestExecuteChatCommand_NonJSON(t *testing.T) {
	// Create mock client
	mock := &mockClient{
		doFunc: func(method, path string, body interface{}) ([]byte, error) {
			return []byte(`{"choices":[{"message":{"content":"Hello"}},"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}}`), nil
		},
	}

	cmd := &cobra.Command{}
	cmd.Flags().String("model", "deepseek-v4-pro", "model")
	cmd.Flags().String("system", "", "system")
	cmd.Flags().String("user", "", "user")
	cmd.Flags().String("assistant", "", "assistant")
	cmd.Flags().String("thinking", "enabled", "thinking")
	cmd.Flags().String("reasoning-effort", "high", "reasoning")
	cmd.Flags().Bool("stream", false, "stream")
	cmd.Flags().Float64("temperature", 1.0, "temp")
	cmd.Flags().Float64("top-p", 1.0, "top-p")
	cmd.Flags().Int("max-tokens", 0, "max")
	cmd.Flags().Float64("frequency-penalty", 0.0, "fp")
	cmd.Flags().Float64("presence-penalty", 0.0, "pp")
	cmd.Flags().Bool("json-mode", false, "json")
	cmd.Flags().StringSlice("stop", []string{}, "stop")
	cmd.Flags().Bool("include-usage", false, "usage")
	cmd.Flags().String("tools", "", "tools")
	cmd.Flags().String("tool-choice", "auto", "tc")
	cmd.Flags().Bool("logprobs", false, "lp")
	cmd.Flags().Int("top-logprobs", 0, "tlp")
	cmd.Flags().Bool("prefix-completion", false, "prefix")
	cmd.Flags().Bool("beta", false, "beta")
	cmd.Flags().String("base-url", "", "base")
	_ = cmd.ParseFlags([]string{"--user", "hello"})

	err := executeChatCommand(cmd, mock)
	if err != nil {
		t.Errorf("executeChatCommand() non-JSON error = %v", err)
	}
}

func TestExecuteChatCommand_JSONMode(t *testing.T) {
	// Create mock client
	mock := &mockClient{
		doFunc: func(method, path string, body interface{}) ([]byte, error) {
			return []byte(`{"choices":[{"message":{"content":"{\"key\":\"value\"}"}},"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}}`), nil
		},
	}

	cmd := &cobra.Command{}
	cmd.Flags().String("model", "deepseek-v4-pro", "model")
	cmd.Flags().String("system", "", "system")
	cmd.Flags().String("user", "", "user")
	cmd.Flags().String("assistant", "", "assistant")
	cmd.Flags().String("thinking", "enabled", "thinking")
	cmd.Flags().String("reasoning-effort", "high", "reasoning")
	cmd.Flags().Bool("stream", false, "stream")
	cmd.Flags().Float64("temperature", 1.0, "temp")
	cmd.Flags().Float64("top-p", 1.0, "top-p")
	cmd.Flags().Int("max-tokens", 0, "max")
	cmd.Flags().Float64("frequency-penalty", 0.0, "fp")
	cmd.Flags().Float64("presence-penalty", 0.0, "pp")
	cmd.Flags().Bool("json-mode", true, "json")
	cmd.Flags().StringSlice("stop", []string{}, "stop")
	cmd.Flags().Bool("include-usage", false, "usage")
	cmd.Flags().String("tools", "", "tools")
	cmd.Flags().String("tool-choice", "auto", "tc")
	cmd.Flags().Bool("logprobs", false, "lp")
	cmd.Flags().Int("top-logprobs", 0, "tlp")
	cmd.Flags().Bool("prefix-completion", false, "prefix")
	cmd.Flags().Bool("beta", false, "beta")
	cmd.Flags().String("base-url", "", "base")
	_ = cmd.ParseFlags([]string{"--user", "hello", "--json-mode"})

	err := executeChatCommand(cmd, mock)
	if err != nil {
		t.Errorf("executeChatCommand() JSON mode error = %v", err)
	}
}

func TestExecuteFIMCommand_WithMockClient(t *testing.T) {
	// Create mock client
	mock := &mockClient{
		doFunc: func(method, path string, body interface{}) ([]byte, error) {
			// Return a valid FIM response
			return []byte(`{"choices":[{"text":"func main() {}"}],"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}}`), nil
		},
	}

	cmd := &cobra.Command{}
	cmd.Flags().String("json", "", "JSON input")
	cmd.Flags().Bool("beta", false, "beta flag")
	cmd.Flags().String("base-url", "", "base url")
	_ = cmd.ParseFlags([]string{"--json", `{"model":"deepseek-v4-pro","prompt":"func main() {"}`})

	err := executeFIMCommand(cmd, mock)
	if err != nil {
		t.Errorf("executeFIMCommand() with mock client error = %v", err)
	}
}

func TestExecuteFIMCommand_NonJSON(t *testing.T) {
	// Create mock client
	mock := &mockClient{
		doFunc: func(method, path string, body interface{}) ([]byte, error) {
			return []byte(`{"choices":[{"text":"func main() {}"}],"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}}`), nil
		},
	}

	cmd := &cobra.Command{}
	cmd.Flags().String("model", "deepseek-v4-pro", "model")
	cmd.Flags().String("prompt", "", "prompt")
	cmd.Flags().String("suffix", "", "suffix")
	cmd.Flags().Bool("stream", false, "stream")
	cmd.Flags().Bool("echo", false, "echo")
	cmd.Flags().Int("max-tokens", 0, "max")
	cmd.Flags().Float64("temperature", 0.2, "temp")
	cmd.Flags().Float64("top-p", 1.0, "top-p")
	cmd.Flags().Float64("frequency-penalty", 0.0, "fp")
	cmd.Flags().Float64("presence-penalty", 0.0, "pp")
	cmd.Flags().StringSlice("stop", []string{}, "stop")
	cmd.Flags().Bool("include-usage", false, "usage")
	cmd.Flags().Int("logprobs", 0, "lp")
	cmd.Flags().Bool("beta", false, "beta")
	cmd.Flags().String("base-url", "", "base")
	_ = cmd.ParseFlags([]string{"--prompt", "test prompt"})

	err := executeFIMCommand(cmd, mock)
	if err != nil {
		t.Errorf("executeFIMCommand() non-JSON error = %v", err)
	}
}


func TestExecuteChatCommand_StreamingWithMockClient(t *testing.T) {
	// Create mock client with streaming
	mock := &mockClient{
		streamChatFunc: func(req *ChatRequest) error {
			// Simulate streaming output
			return nil
		},
	}

	cmd := &cobra.Command{}
	cmd.Flags().String("json", "", "JSON input")
	cmd.Flags().Bool("beta", false, "beta flag")
	cmd.Flags().String("base-url", "", "base url")
	cmd.Flags().Bool("cache", false, "cache flag")
	_ = cmd.ParseFlags([]string{"--json", `{"model":"deepseek-v4-pro","messages":[{"role":"user","content":"hi"}],"stream":true}`})

	err := executeChatCommand(cmd, mock)
	if err != nil {
		t.Errorf("executeChatCommand() streaming with mock client error = %v", err)
	}
}

func TestExecuteFIMCommand_StreamingWithMockClient(t *testing.T) {
	// Create mock client with streaming
	mock := &mockClient{
		streamFIMFunc: func(req *FIMRequest) error {
			// Simulate streaming output
			return nil
		},
	}

	cmd := &cobra.Command{}
	cmd.Flags().String("json", "", "JSON input")
	cmd.Flags().Bool("beta", false, "beta flag")
	cmd.Flags().String("base-url", "", "base url")
	_ = cmd.ParseFlags([]string{"--json", `{"model":"deepseek-v4-pro","prompt":"func main() {"}`})

	err := executeFIMCommand(cmd, mock)
	if err != nil {
		t.Errorf("executeFIMCommand() streaming with mock client error = %v", err)
	}
}

func TestExecuteModelsCommand_WithMockClient(t *testing.T) {
	// Create mock client
	mock := &mockClient{
		doFunc: func(method, path string, body interface{}) ([]byte, error) {
			return []byte(`{"object":"list","data":[{"id":"deepseek-v4-pro","object":"model","owned_by":"deepseek"}]}`), nil
		},
	}

	cmd := &cobra.Command{}
	err := executeModelsCommand(cmd, mock)
	if err != nil {
		t.Errorf("executeModelsCommand() with mock client error = %v", err)
	}
}

func TestExecuteBalanceCommand_WithMockClient(t *testing.T) {
	// Create mock client
	mock := &mockClient{
		doFunc: func(method, path string, body interface{}) ([]byte, error) {
			return []byte(`{"balance":100.50,"total_balance":150.75,"available_balance":100.50,"granted_balance":50.25}`), nil
		},
	}

	cmd := &cobra.Command{}
	err := executeBalanceCommand(cmd, mock)
	if err != nil {
		t.Errorf("executeBalanceCommand() with mock client error = %v", err)
	}
}

func TestExecuteSingleTurn_WithMockClient(t *testing.T) {
	// Create mock client
	mock := &mockClient{
		doFunc: func(method, path string, body interface{}) ([]byte, error) {
			return []byte(`{"choices":[{"message":{"content":"Mock response"}},"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}}`), nil
		},
	}

	cmd := &cobra.Command{}
	config := &Config{
		Chat: GetDefaultChatConfig(),
	}

	err := executeSingleTurn(cmd, "test prompt", config, mock)
	if err != nil {
		t.Errorf("executeSingleTurn() with mock client error = %v", err)
	}
}

func TestExecuteChatCommand_NoClientNoAPIKey(t *testing.T) {
	// Save and restore env
	oldKey := os.Getenv("DEEPSEEK_API_KEY")
	defer func() { _ = os.Setenv("DEEPSEEK_API_KEY", oldKey) }()
	_ = os.Unsetenv("DEEPSEEK_API_KEY")

	cmd := &cobra.Command{}
	cmd.Flags().String("json", "", "JSON input")
	cmd.Flags().Bool("beta", false, "beta flag")
	cmd.Flags().String("base-url", "", "base url")
	cmd.Flags().Bool("cache", false, "cache flag")
	_ = cmd.ParseFlags([]string{"--json", `{"model":"deepseek-v4-pro","messages":[{"role":"user","content":"hi"}]}`})

	err := executeChatCommand(cmd, nil)
	if err == nil {
		t.Error("executeChatCommand() should error without API key")
	}
}

func TestExecuteFIMCommand_NoClientNoAPIKey(t *testing.T) {
	// Save and restore env
	oldKey := os.Getenv("DEEPSEEK_API_KEY")
	defer func() { _ = os.Setenv("DEEPSEEK_API_KEY", oldKey) }()
	_ = os.Unsetenv("DEEPSEEK_API_KEY")

	cmd := &cobra.Command{}
	cmd.Flags().String("json", "", "JSON input")
	cmd.Flags().Bool("beta", false, "beta flag")
	cmd.Flags().String("base-url", "", "base url")
	_ = cmd.ParseFlags([]string{"--json", `{"model":"deepseek-v4-pro","prompt":"test"}`})

	err := executeFIMCommand(cmd, nil)
	if err == nil {
		t.Error("executeFIMCommand() should error without API key")
	}
}

func TestFormatJSONModeResponse_Additional(t *testing.T) {
	// Test with nil message content
	data := `{"choices":[{"message":{"content":null}}],"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}}`
	err := formatJSONModeResponse([]byte(data), false)
	if err != nil {
		t.Errorf("formatJSONModeResponse() with nil content error = %v", err)
	}

	// Test with invalid JSON in content
	data = `{"choices":[{"message":{"content":"not valid json"}}]}`
	err = formatJSONModeResponse([]byte(data), false)
	if err != nil {
		t.Errorf("formatJSONModeResponse() with invalid JSON in content error = %v", err)
	}

	// Test with no choices
	data = `{"choices":[]}`
	err = formatJSONModeResponse([]byte(data), false)
	if err != nil {
		t.Errorf("formatJSONModeResponse() with no choices error = %v", err)
	}

	// Test with no message
	data = `{"choices":[{}]}`
	err = formatJSONModeResponse([]byte(data), false)
	if err != nil {
		t.Errorf("formatJSONModeResponse() with no message error = %v", err)
	}
}

func TestParseSSEStream_Additional(t *testing.T) {
	// Test with ChatResponse format (not delta)
	input := "data: {\"choices\":[{\"message\":{\"content\":\"Hello\"}}]}\n\ndata: [DONE]\n\n"
	client := &Client{}
	err := client.parseSSEStream(strings.NewReader(input), "chat")
	if err != nil {
		t.Errorf("parseSSEStream() with ChatResponse format error = %v", err)
	}

	// Test with FIMResponse format (not simple chunk)
	input = "data: {\"choices\":[{\"text\":\"code\"}]}\n\ndata: [DONE]\n\n"
	client = &Client{}
	err = client.parseSSEStream(strings.NewReader(input), "fim")
	if err != nil {
		t.Errorf("parseSSEStream() with FIMResponse format error = %v", err)
	}

	// Test with multiple choices
	input = "data: {\"choices\":[{\"delta\":{\"content\":\"a\"}},{\"delta\":{\"content\":\"b\"}}]}\n\ndata: [DONE]\n\n"
	client = &Client{}
	err = client.parseSSEStream(strings.NewReader(input), "chat")
	if err != nil {
		t.Errorf("parseSSEStream() with multiple choices error = %v", err)
	}

	// Test with scanner error
	errorReader := &errorReader{err: io.ErrUnexpectedEOF}
	client = &Client{}
	err = client.parseSSEStream(errorReader, "chat")
	if err == nil {
		t.Error("parseSSEStream() should return error from scanner")
	}
}

func TestExecuteModelsCommand_NoClient(t *testing.T) {
	// Save and restore env
	oldKey := os.Getenv("DEEPSEEK_API_KEY")
	oldBase := os.Getenv("DEEPSEEK_API_BASE")
	defer func() {
		_ = os.Setenv("DEEPSEEK_API_KEY", oldKey)
		_ = os.Setenv("DEEPSEEK_API_BASE", oldBase)
	}()

	// Set env vars
	_ = os.Setenv("DEEPSEEK_API_KEY", "test-key")
	_ = os.Setenv("DEEPSEEK_API_BASE", "https://test.com")

	cmd := &cobra.Command{}
	err := executeModelsCommand(cmd, nil)
	// Will fail with HTTP error since no real server
	if err == nil {
		t.Error("executeModelsCommand() should error without real server")
	}
}

func TestExecuteBalanceCommand_NoClient(t *testing.T) {
	// Save and restore env
	oldKey := os.Getenv("DEEPSEEK_API_KEY")
	oldBase := os.Getenv("DEEPSEEK_API_BASE")
	defer func() {
		_ = os.Setenv("DEEPSEEK_API_KEY", oldKey)
		_ = os.Setenv("DEEPSEEK_API_BASE", oldBase)
	}()

	// Set env vars
	_ = os.Setenv("DEEPSEEK_API_KEY", "test-key")
	_ = os.Setenv("DEEPSEEK_API_BASE", "https://test.com")

	cmd := &cobra.Command{}
	err := executeBalanceCommand(cmd, nil)
	// Will fail with HTTP error since no real server
	if err == nil {
		t.Error("executeBalanceCommand() should error without real server")
	}
}

func TestExecuteHistoryMode_WithMockClient(t *testing.T) {
	// Create mock client
	mock := &mockClient{
		doFunc: func(method, path string, body interface{}) ([]byte, error) {
			return []byte(`{"choices":[{"message":{"content":"Mock response"}},"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}}`), nil
		},
	}

	// Create a temp file with valid history
	tmpfile, err := os.CreateTemp("", "history*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()

	history := `[{"role":"user","content":"test"}]`
	_, _ = tmpfile.WriteString(history)
	_ = tmpfile.Close()

	cmd := &cobra.Command{}
	config := &Config{
		Chat: GetDefaultChatConfig(),
	}

	err = executeHistoryMode(cmd, tmpfile.Name(), config, mock)
	if err != nil {
		t.Errorf("executeHistoryMode() with mock client error = %v", err)
	}
}

func TestExecuteStdinMode_WithMockClient(t *testing.T) {
	// Create mock client
	mock := &mockClient{
		doFunc: func(method, path string, body interface{}) ([]byte, error) {
			return []byte(`{"choices":[{"message":{"content":"Mock response"}},"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}}`), nil
		},
	}

	// Save original stdin
	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()

	// Create a pipe to simulate stdin with data
	r, w, _ := os.Pipe()
	os.Stdin = r

	go func() {
		_, _ = w.WriteString("test input")
		_ = w.Close()
	}()

	cmd := &cobra.Command{}
	config := &Config{
		Chat: GetDefaultChatConfig(),
	}

	err := executeStdinMode(cmd, config, mock)
	if err != nil {
		t.Errorf("executeStdinMode() with mock client error = %v", err)
	}
}

func TestFormatJSONModeResponse_AdditionalTests(t *testing.T) {
	// Test with valid JSON content that needs pretty-printing
	data := `{"choices":[{"message":{"content":"{\"name\":\"test\",\"value\":123}"}}],"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}}`
	err := formatJSONModeResponse([]byte(data), false)
	if err != nil {
		t.Errorf("formatJSONModeResponse() with valid JSON content error = %v", err)
	}

	// Test with showCache
	data = `{"choices":[{"message":{"content":"{\"a\":1}"}}],"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15,"prompt_cache_hit_tokens":3,"prompt_cache_miss_tokens":7}}`
	err = formatJSONModeResponse([]byte(data), true)
	if err != nil {
		t.Errorf("formatJSONModeResponse() with cache error = %v", err)
	}

	// Test with raw JSON (not JSON mode)
	data = `{"choices":[{"message":{"content":"hello"}}]}`
	err = formatJSONModeResponse([]byte(data), false)
	if err != nil {
		t.Errorf("formatJSONModeResponse() with raw JSON error = %v", err)
	}
}

func TestParseSSEStream_MoreTests(t *testing.T) {
	// Test with empty data lines
	input := "data: \n\n" +
		"data: {\"choices\":[{\"delta\":{\"content\":\"test\"}}]}\n\n" +
		"data: [DONE]\n\n"

	client := &Client{}
	err := client.parseSSEStream(strings.NewReader(input), "chat")
	if err != nil {
		t.Errorf("parseSSEStream() with empty data lines error = %v", err)
	}

	// Test with usage-only chunk
	input = "data: {\"usage\":{\"prompt_tokens\":10,\"completion_tokens\":5,\"total_tokens\":15}}\n\n" +
		"data: [DONE]\n\n"

	client = &Client{}
	err = client.parseSSEStream(strings.NewReader(input), "chat")
	if err != nil {
		t.Errorf("parseSSEStream() with usage-only chunk error = %v", err)
	}

	// Test with non-data lines
	input = "event: message\n" +
		"data: {\"choices\":[{\"delta\":{\"content\":\"test\"}}]}\n\n" +
		"data: [DONE]\n\n"

	client = &Client{}
	err = client.parseSSEStream(strings.NewReader(input), "chat")
	if err != nil {
		t.Errorf("parseSSEStream() with non-data lines error = %v", err)
	}
}

func TestRun_WithArgs(t *testing.T) {
	// Save original os.Args
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// Test with --version
	os.Args = []string{"deepseek", "--version"}
	err := run()
	if err != nil {
		t.Errorf("run() with --version error = %v", err)
	}
}

func TestRun_WithHelp(t *testing.T) {
	// Save original os.Args
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// Test with --help
	os.Args = []string{"deepseek", "--help"}
	// This will print help and return nil or error
	_ = run()
}

func TestFormatJSONModeResponse_MoreTests(t *testing.T) {
	// Test with nil choices
	data := `{"choices":null}`
	err := formatJSONModeResponse([]byte(data), false)
	if err != nil {
		t.Errorf("formatJSONModeResponse() with nil choices error = %v", err)
	}

	// Test with empty choices array
	data = `{"choices":[]}`
	err = formatJSONModeResponse([]byte(data), false)
	if err != nil {
		t.Errorf("formatJSONModeResponse() with empty choices error = %v", err)
	}

	// Test with valid JSON content
	data = `{"choices":[{"message":{"content":"{\"a\":1,\"b\":\"test\"}"}}]}`
	err = formatJSONModeResponse([]byte(data), false)
	if err != nil {
		t.Errorf("formatJSONModeResponse() with valid JSON content error = %v", err)
	}

	// Test with invalid JSON content but valid wrapper
	data = `{"choices":[{"message":{"content":"not json"}}]}`
	err = formatJSONModeResponse([]byte(data), false)
	if err != nil {
		t.Errorf("formatJSONModeResponse() with invalid JSON content error = %v", err)
	}
}


func TestRun_WithPrompt(t *testing.T) {
	// Save original os.Args
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// Test with -p flag (calls handleSingleTurn)
	os.Args = []string{"deepseek", "-p", "test prompt"}
	// This will fail with no API key, but should cover the prompt path
	_ = run()
}

func TestRun_WithHistory(t *testing.T) {
	// Save original os.Args
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// Test with --history flag (calls handleHistoryMode)
	os.Args = []string{"deepseek", "--history", "/nonexistent/file.json"}
	// This will fail with file not found, but should cover the history path
	// We expect this to fail, so just run it and ignore the result
	_ = run()
}

func TestRun_NoFlagsWithStdin(t *testing.T) {
	// Save original os.Args
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// Save original stdin
	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()

	// Create a pipe to simulate stdin with data
	r, w, _ := os.Pipe()
	os.Stdin = r

	go func() {
		_, _ = w.WriteString("test input")
		_ = w.Close()
	}()

	// Test with no flags but stdin has data (calls handleStdinMode)
	os.Args = []string{"deepseek"}
	// This will fail with no API key, but should cover the stdin path
	_ = run()
}

func TestFormatJSONModeResponse_Comprehensive(t *testing.T) {
	// Test with nil resp but valid JSON wrapping
	data := `{"choices":null}`
	err := formatJSONModeResponse([]byte(data), false)
	if err != nil {
		t.Errorf("formatJSONModeResponse() with null choices error = %v", err)
	}

	// Test with choices array but no message
	data = `{"choices":[{}]}`
	err = formatJSONModeResponse([]byte(data), false)
	if err != nil {
		t.Errorf("formatJSONModeResponse() with empty message error = %v", err)
	}

	// Test with message but no content
	data = `{"choices":[{"message":{}}]}`
	err = formatJSONModeResponse([]byte(data), false)
	if err != nil {
		t.Errorf("formatJSONModeResponse() with no content error = %v", err)
	}

	// Test with invalid JSON in content that can't be pretty-printed
	data = `{"choices":[{"message":{"content":"not json"}}]}`
	err = formatJSONModeResponse([]byte(data), false)
	if err != nil {
		t.Errorf("formatJSONModeResponse() with non-JSON content error = %v", err)
	}

	// Test with showCache
	data = `{"choices":[{"message":{"content":"{\"a\":1}"}}],"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15,"prompt_cache_hit_tokens":3,"prompt_cache_miss_tokens":7}}`
	err = formatJSONModeResponse([]byte(data), true)
	if err != nil {
		t.Errorf("formatJSONModeResponse() with cache error = %v", err)
	}
}

func TestParseSSEStream_Comprehensive(t *testing.T) {
	// Test with invalid JSON chunk (should skip)
	input := "data: invalid json\n\n" +
		"data: {\"choices\":[{\"delta\":{\"content\":\"valid\"}}]}\n\n" +
		"data: [DONE]\n\n"
	client := &Client{}
	err := client.parseSSEStream(strings.NewReader(input), "chat")
	if err != nil {
		t.Errorf("parseSSEStream() with invalid JSON chunk error = %v", err)
	}

	// Test with non-data lines (should skip)
	input = "event: message\n" +
		"data: {\"choices\":[{\"delta\":{\"content\":\"test\"}}]}\n\n" +
		"data: [DONE]\n\n"
	client = &Client{}
	err = client.parseSSEStream(strings.NewReader(input), "chat")
	if err != nil {
		t.Errorf("parseSSEStream() with non-data lines error = %v", err)
	}

	// Test with only finish_reason
	input = "data: {\"choices\":[{\"finish_reason\":\"stop\"}]}\n\n" +
		"data: [DONE]\n\n"
	client = &Client{}
	err = client.parseSSEStream(strings.NewReader(input), "chat")
	if err != nil {
		t.Errorf("parseSSEStream() with only finish_reason error = %v", err)
	}

	// Test with empty input
	client = &Client{}
	err = client.parseSSEStream(strings.NewReader(""), "chat")
	if err != nil {
		t.Errorf("parseSSEStream() with empty input error = %v", err)
	}

	// Test with only [DONE]
	client = &Client{}
	err = client.parseSSEStream(strings.NewReader("data: [DONE]\n\n"), "chat")
	if err != nil {
		t.Errorf("parseSSEStream() with only DONE error = %v", err)
	}
}
