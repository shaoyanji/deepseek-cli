// Package hooks provides hook functions for extending agent behavior.
package hooks

import (
	"context"

	"deepseek-cli/internal/lsp"
)

// LSPHook is called after a file is written to trigger diagnostics.
// This is a placeholder for future extensibility.
type LSPHook struct {
	client *lsp.Client
}

// NewLSPHook creates a new LSP hook with the given client.
func NewLSPHook(client *lsp.Client) *LSPHook {
	return &LSPHook{client: client}
}

// OnFileWrite is called after a file is written.
// It triggers LSP diagnostics if the client is configured.
func (h *LSPHook) OnFileWrite(ctx context.Context, filePath string) ([]lsp.Diagnostic, error) {
	if h.client == nil {
		return []lsp.Diagnostic{}, nil
	}

	diags, err := h.client.RunDiagnostics(ctx, filePath)
	if err != nil {
		return nil, err
	}

	return diags, nil
}
