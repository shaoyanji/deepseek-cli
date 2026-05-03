package execpolicy

import (
	"testing"
)

func TestAcmePolicy(t *testing.T) {
	policy := NewAcmePolicy()
	
	if policy.Name() != "Acme (Plan)" {
		t.Errorf("Expected name 'Acme (Plan)', got '%s'", policy.Name())
	}
	
	if !policy.IsReadOnly() {
		t.Error("Expected Acme policy to be read-only")
	}
	
	// Test read-only tools
	readOnlyTools := []string{"view", "ls", "grep", "fetch", "web_search", "lsp"}
	for _, tool := range readOnlyTools {
		if !policy.CanExecute(tool, nil) {
			t.Errorf("Expected %s to be executable in Acme mode", tool)
		}
	}
	
	// Test write tools
	writeTools := []string{"edit", "bash", "git"}
	for _, tool := range writeTools {
		if policy.CanExecute(tool, nil) {
			t.Errorf("Expected %s to NOT be executable in Acme mode", tool)
		}
	}
}

func TestAgentPolicy(t *testing.T) {
	policy := NewAgentPolicy()
	
	if policy.Name() != "Agent" {
		t.Errorf("Expected name 'Agent', got '%s'", policy.Name())
	}
	
	if policy.IsReadOnly() {
		t.Error("Expected Agent policy to NOT be read-only")
	}
	
	// In Agent mode, nothing executes without approval
	if policy.CanExecute("edit", nil) {
		t.Error("Expected edit to require approval in Agent mode")
	}
	
	if !policy.RequiresApproval("edit", nil) {
		t.Error("Expected edit to require approval in Agent mode")
	}
}

func TestYOLOPolicy(t *testing.T) {
	policy := NewYOLOPolicy()
	
	if policy.Name() != "YOLO" {
		t.Errorf("Expected name 'YOLO', got '%s'", policy.Name())
	}
	
	if policy.IsReadOnly() {
		t.Error("Expected YOLO policy to NOT be read-only")
	}
	
	// In YOLO mode, everything executes automatically
	if !policy.CanExecute("edit", nil) {
		t.Error("Expected edit to be executable in YOLO mode")
	}
	
	if policy.RequiresApproval("edit", nil) {
		t.Error("Expected no approval required in YOLO mode")
	}
}

func TestPolicyFactory(t *testing.T) {
	factory := NewPolicyFactory()
	
	// Test mode parsing
	tests := []struct {
		input    string
		expected ExecutionMode
	}{
		{"acme", ModeAcme},
		{"plan", ModeAcme},
		{"read-only", ModeAcme},
		{"agent", ModeAgent},
		{"interactive", ModeAgent},
		{"yolo", ModeYOLO},
		{"auto", ModeYOLO},
		{"automatic", ModeYOLO},
	}
	
	for _, test := range tests {
		mode, err := factory.ParseMode(test.input)
		if err != nil {
			t.Errorf("ParseMode(%q) returned error: %v", test.input, err)
		}
		if mode != test.expected {
			t.Errorf("ParseMode(%q) = %v, expected %v", test.input, mode, test.expected)
		}
	}
	
	// Test invalid mode
	_, err := factory.ParseMode("invalid")
	if err == nil {
		t.Error("Expected error for invalid mode")
	}
	
	// Test policy creation
	modes := []ExecutionMode{ModeAcme, ModeAgent, ModeYOLO}
	for _, mode := range modes {
		policy, err := factory.Create(mode)
		if err != nil {
			t.Errorf("Create(%v) returned error: %v", mode, err)
		}
		if policy == nil {
			t.Errorf("Create(%v) returned nil policy", mode)
		}
	}
	
	// Test invalid policy creation
	_, err = factory.Create("invalid")
	if err == nil {
		t.Error("Expected error for invalid mode creation")
	}
}

func TestToolApproval(t *testing.T) {
	acmePolicy := NewAcmePolicy()
	yoloPolicy := NewYOLOPolicy()
	
	// Test Acme approval for read-only tool
	approval, err := acmePolicy.ApproveTool("view", map[string]interface{}{"path": "test.go"}, "View file contents")
	if err != nil {
		t.Fatalf("ApproveTool failed: %v", err)
	}
	if !approval.Approved {
		t.Error("Expected view tool to be approved in Acme mode")
	}
	
	// Test Acme approval for write tool
	approval, err = acmePolicy.ApproveTool("edit", map[string]interface{}{"path": "test.go", "content": "test"}, "Edit file")
	if err != nil {
		t.Fatalf("ApproveTool failed: %v", err)
	}
	if approval.Approved {
		t.Error("Expected edit tool to be denied in Acme mode")
	}
	
	// Test YOLO approval
	approval, err = yoloPolicy.ApproveTool("edit", map[string]interface{}{"path": "test.go", "content": "test"}, "Edit file")
	if err != nil {
		t.Fatalf("ApproveTool failed: %v", err)
	}
	if !approval.Approved {
		t.Error("Expected edit tool to be auto-approved in YOLO mode")
	}
	if approval.Reason != "Auto-approved in YOLO mode" {
		t.Errorf("Expected auto-approve reason, got: %s", approval.Reason)
	}
}
