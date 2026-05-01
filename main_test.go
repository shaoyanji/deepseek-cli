package main

import (
	"os"
	"testing"

	"github.com/spf13/cobra"
)

func TestMustGetString(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().String("test-flag", "default", "test flag")

	// Test default value
	cmd.ParseFlags([]string{})
	result := mustGetString(cmd, "test-flag")
	if result != "default" {
		t.Errorf("mustGetString() = %q, want %q", result, "default")
	}

	// Test set value
	cmd.ParseFlags([]string{"--test-flag", "custom"})
	result = mustGetString(cmd, "test-flag")
	if result != "custom" {
		t.Errorf("mustGetString() = %q, want %q", result, "custom")
	}
}

func TestMustGetBool(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().Bool("test-bool", false, "test bool flag")

	// Test default value
	cmd.ParseFlags([]string{})
	result := mustGetBool(cmd, "test-bool")
	if result != false {
		t.Errorf("mustGetBool() = %v, want false", result)
	}

	// Test set to true
	cmd.ParseFlags([]string{"--test-bool"})
	result = mustGetBool(cmd, "test-bool")
	if result != true {
		t.Errorf("mustGetBool() = %v, want true", result)
	}
}

func TestMustGetInt(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().Int("test-int", 0, "test int flag")

	// Test default value
	cmd.ParseFlags([]string{})
	result := mustGetInt(cmd, "test-int")
	if result != 0 {
		t.Errorf("mustGetInt() = %d, want 0", result)
	}

	// Test set value
	cmd.ParseFlags([]string{"--test-int", "42"})
	result = mustGetInt(cmd, "test-int")
	if result != 42 {
		t.Errorf("mustGetInt() = %d, want 42", result)
	}
}

func TestMustGetFloat64(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().Float64("test-float", 1.0, "test float flag")

	// Test default value
	cmd.ParseFlags([]string{})
	result := mustGetFloat64(cmd, "test-float")
	if result != 1.0 {
		t.Errorf("mustGetFloat64() = %f, want 1.0", result)
	}

	// Test set value
	cmd.ParseFlags([]string{"--test-float", "0.5"})
	result = mustGetFloat64(cmd, "test-float")
	if result != 0.5 {
		t.Errorf("mustGetFloat64() = %f, want 0.5", result)
	}
}

func TestMustGetStringSlice(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().StringSlice("test-slice", []string{}, "test slice flag")

	// Test default value
	cmd.ParseFlags([]string{})
	result := mustGetStringSlice(cmd, "test-slice")
	if len(result) != 0 {
		t.Errorf("mustGetStringSlice() length = %d, want 0", len(result))
	}

	// Test set value
	cmd.ParseFlags([]string{"--test-slice", "a,b,c"})
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
	
	cmd1.ParseFlags([]string{"--user", "Hello"})
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
	
	cmd2.ParseFlags([]string{"--system", "You are helpful", "--user", "Hi"})
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
	
	cmd3.ParseFlags([]string{"--user", "test", "--temperature", "0.5"})
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
	
	cmd4.ParseFlags([]string{"--user", "test", "--json-mode"})
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
	
	cmd5.ParseFlags([]string{})
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

	cmd1.ParseFlags([]string{"--prompt", "func main() {"})
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

	cmd2.ParseFlags([]string{"--prompt", "func main() {", "--suffix", "}"})
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

	cmd3.ParseFlags([]string{})
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
		w.WriteString("test input")
		w.Close()
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
	defer r.Close()
	defer w.Close()
	
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
	cmd.ParseFlags([]string{"--user", "weather", "--tools", toolsJSON})
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
	cmd.ParseFlags([]string{"--user", "test", "--tool-choice", "none"})
	req, err := buildChatRequest(cmd)
	if err != nil {
		t.Fatalf("buildChatRequest() error = %v", err)
	}
	if req.ToolChoice != "none" {
		t.Errorf("buildChatRequest() tool-choice not set correctly")
	}

	// Test JSON tool choice
	cmd.ParseFlags([]string{"--user", "test", "--tool-choice", `{"type":"function","function":{"name":"get_weather"}}`})
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

	cmd.ParseFlags([]string{"--user", "test", "--logprobs", "--top-logprobs", "5"})
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
	cmd.ParseFlags([]string{"--user", "test", "--stop", "STOP"})
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

	cmd.ParseFlags([]string{"--prompt", "test", "--logprobs", "5"})
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

	cmd.ParseFlags([]string{"--prompt", "test", "--stream", "--include-usage"})
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

	cmd.ParseFlags([]string{"--prompt", "test", "--stop", "STOP"})
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

	cmd.ParseFlags([]string{"--prompt", "test", "--frequency-penalty", "1.0", "--presence-penalty", "-0.5"})
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
	cmd.ParseFlags([]string{"--prompt", "test", "--temperature", "0.5"})
	req, err := buildFIMRequest(cmd)
	if err != nil {
		t.Fatalf("buildFIMRequest() error = %v", err)
	}
	if req.Temperature == nil || *req.Temperature != 0.5 {
		t.Errorf("buildFIMRequest() temperature not set correctly")
	}

	// Test with non-default top-p
	cmd.ParseFlags([]string{"--prompt", "test", "--top-p", "0.8"})
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

	cmd.ParseFlags([]string{"--assistant", "previous", "--prefix-completion"})
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

	cmd.ParseFlags([]string{"--user", "test", "--stream", "--include-usage"})
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

	cmd.ParseFlags([]string{"--user", "test", "--frequency-penalty", "1.5", "--presence-penalty", "-0.5"})
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
	defer os.Setenv("DEEPSEEK_API_KEY", oldKey)
	os.Unsetenv("DEEPSEEK_API_KEY")

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
	w.Close()
	defer r.Close()

	cmd := &cobra.Command{}
	config := &Config{
		Chat: GetDefaultChatConfig(),
	}


func TestExecuteStdinMode_EmptyInput(t *testing.T) {
	// Save original stdin
	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()

	r, w, _ := os.Pipe()
	os.Stdin = r
	w.Close()

	cmd := &cobra.Command{}
	config := &Config{
		Chat: GetDefaultChatConfig(),
	}

	err := executeStdinMode(cmd, config)
	if err == nil {
		t.Error("executeStdinMode() should error with empty stdin")
	}
}

func TestExecuteHistoryMode_FileNotExists(t *testing.T) {
	cmd := &cobra.Command{}
	config := &Config{
		Chat: GetDefaultChatConfig(),
	}

	err := executeHistoryMode(cmd, "/nonexistent/file.json", config)
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
	defer os.Remove(tmpfile.Name())

	tmpfile.WriteString("invalid json")
	tmpfile.Close()

	cmd := &cobra.Command{}
	config := &Config{
		Chat: GetDefaultChatConfig(),
	}

	err = executeHistoryMode(cmd, tmpfile.Name(), config)
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

	cmd.ParseFlags([]string{"--user", "test", "--tools", "invalid"})
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
	cmd.ParseFlags([]string{"--user", "test", "--tool-choice", `{"type":"function"`})
	_, err := buildChatRequest(cmd)
	if err == nil {
		t.Error("buildChatRequest() should error with invalid tool-choice JSON")
	}
}
