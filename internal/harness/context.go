package harness

import (
	"fmt"
	"sync"
)

// Context is the shared runtime context that all plugins contribute to and consume from.
// Following the Cordis-inspired design, there's no privileged core - everything is a plugin.
type Context struct {
	mu sync.RWMutex
	
	// Core services
	sessions       SessionService
	systemPrompt   SystemPromptService
	tools          ToolRegistry
	agents         AgentRegistry
	llm            LLMService
	sessionProjections []SessionProjection
	
	// Capability seams
	fs             FSService
	shell          ShellService
	subagent       SubagentService
	sandbox        SandboxService
	
	// Extension points
	commands       CommandRegistry
	jobs           JobService
	webhookRuntime WebhookRuntime
	
	// Plugin state
	plugins map[string]*PluginHandle
	scopes  map[string]*Scope
}

// NewContext creates a new empty runtime context
func NewContext() *Context {
	return &Context{
		plugins: make(map[string]*PluginHandle),
		scopes:  make(map[string]*Scope),
	}
}

// Plugin defines the interface that all plugins must implement
type Plugin interface {
	// Name returns the unique identifier for this plugin
	Name() string
	
	// Install registers services, events, and effects into the context
	Install(ctx *Context) error
	
	// Uninstall reverses all registrations made during Install
	Uninstall(ctx *Context) error
}

// PluginHandle tracks a mounted plugin and its reversible effects
type PluginHandle struct {
	plugin    Plugin
	effects   []Effect
	unloaded  bool
}

// Effect represents a reversible registration or state change
type Effect func() // undo function

// Scope provides scoped registration that isolates one agent's contributions
type Scope struct {
	id       string
	parent   *Context
	effects  []Effect
}

// Mount installs a plugin into the context, making its services available
func (c *Context) Mount(plugin Plugin) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	name := plugin.Name()
	if _, exists := c.plugins[name]; exists {
		return fmt.Errorf("plugin already mounted: %s", name)
	}
	
	handle := &PluginHandle{
		plugin:   plugin,
		effects:  make([]Effect, 0),
		unloaded: false,
	}
	
	// Create a temporary context for installation to track effects
	if err := plugin.Install(c); err != nil {
		return fmt.Errorf("plugin install failed: %w", err)
	}
	
	c.plugins[name] = handle
	return nil
}

// Unmount removes a plugin and reverses all its effects
func (c *Context) Unmount(name string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	handle, exists := c.plugins[name]
	if !exists {
		return fmt.Errorf("plugin not found: %s", name)
	}
	
	if handle.unloaded {
		return fmt.Errorf("plugin already unloaded: %s", name)
	}
	
	// Reverse effects in LIFO order
	for i := len(handle.effects) - 1; i >= 0; i-- {
		handle.effects[i]() // undo effect
	}
	
	if err := handle.plugin.Uninstall(c); err != nil {
		return fmt.Errorf("plugin uninstall failed: %w", err)
	}
	
	handle.unloaded = true
	delete(c.plugins, name)
	return nil
}

// CreateScope creates a new scoped registration context for an agent
func (c *Context) CreateScope(id string, parent *Context) *Scope {
	return &Scope{
		id:      id,
		parent:  parent,
		effects: make([]Effect, 0),
	}
}

// Register adds a reversible effect to the scope
func (s *Scope) Register(effect Effect) {
	s.effects = append(s.effects, effect)
}

// Dispose reverses all effects in the scope
func (s *Scope) Dispose() {
	for i := len(s.effects) - 1; i >= 0; i-- {
		s.effects[i]() // undo effect
	}
}

// WithScope executes a function within a scoped context
func (c *Context) WithScope(id string, fn func(*Scope) error) error {
	scope := c.CreateScope(id, c)
	defer scope.Dispose()
	return fn(scope)
}

// Inject adds a service to the context with an optional cleanup effect
func (c *Context) Inject(setter func(), cleanup Effect) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	setter()
	if cleanup != nil {
		// Track cleanup for unmounting
		// In practice, this would be associated with the mounting plugin
	}
}
