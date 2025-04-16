package toolsets

import (
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func NewServerTool(tool mcp.Tool, handler server.ToolHandlerFunc) server.ServerTool {
	return server.ServerTool{Tool: tool, Handler: handler}
}

type Toolset struct {
	Name          string
	Description   string
	Enabled       bool
	readOnly      bool
	writeTools    []server.ServerTool
	readTools     []server.ServerTool
	disabledTools map[string]bool // Map for efficient lookup
}

func (t *Toolset) GetActiveTools() []server.ServerTool {
	if !t.Enabled {
		return nil
	}
	activeTools := []server.ServerTool{}
	appendIfNotDisabled := func(tools []server.ServerTool) {
		for _, tool := range tools {
			if !t.disabledTools[tool.Tool.Name] {
				activeTools = append(activeTools, tool)
			}
		}
	}

	appendIfNotDisabled(t.readTools)
	if !t.readOnly {
		appendIfNotDisabled(t.writeTools)
	}
	return activeTools
}

func (t *Toolset) GetAvailableTools() []server.ServerTool {
	// This lists *all* potential tools, regardless of disabled status
	if t.readOnly {
		return t.readTools
	}
	return append(t.readTools, t.writeTools...)
}

// RegisterTools registers only the enabled and *not disabled* tools with the server.
func (t *Toolset) RegisterTools(s *server.MCPServer) {
	if !t.Enabled {
		return
	}

	registerIfNotDisabled := func(tools []server.ServerTool) {
		for _, tool := range tools {
			if !t.disabledTools[tool.Tool.Name] {
				s.AddTool(tool.Tool, tool.Handler)
			}
		}
	}

	registerIfNotDisabled(t.readTools)
	if !t.readOnly {
		registerIfNotDisabled(t.writeTools)
	}
}

func (t *Toolset) SetReadOnly() {
	// Set the toolset to read-only
	t.readOnly = true
}

func (t *Toolset) AddWriteTools(tools ...server.ServerTool) *Toolset {
	// Silently ignore if the toolset is read-only to avoid any breach of that contract
	if !t.readOnly {
		t.writeTools = append(t.writeTools, tools...)
	}
	return t
}

func (t *Toolset) AddReadTools(tools ...server.ServerTool) *Toolset {
	t.readTools = append(t.readTools, tools...)
	return t
}

type ToolsetGroup struct {
	Toolsets      map[string]*Toolset
	everythingOn  bool
	readOnly      bool
	disabledTools map[string]bool // Store disabled tools here
}

// NewToolsetGroup creates a new ToolsetGroup, initializing the disabled tools map.
func NewToolsetGroup(readOnly bool, disabledToolsList []string) *ToolsetGroup {
	disabledToolsMap := make(map[string]bool)
	for _, toolName := range disabledToolsList {
		disabledToolsMap[toolName] = true
	}
	return &ToolsetGroup{
		Toolsets:      make(map[string]*Toolset),
		everythingOn:  false,
		readOnly:      readOnly,
		disabledTools: disabledToolsMap,
	}
}

func (tg *ToolsetGroup) AddToolset(ts *Toolset) {
	if tg.readOnly {
		ts.SetReadOnly()
	}
	ts.disabledTools = tg.disabledTools // Pass down the disabled map to the toolset
	tg.Toolsets[ts.Name] = ts
}

func NewToolset(name string, description string) *Toolset {
	return &Toolset{
		Name:          name,
		Description:   description,
		Enabled:       false,
		readOnly:      false,
		disabledTools: make(map[string]bool), // Initialize the map
	}
}

func (tg *ToolsetGroup) IsEnabled(name string) bool {
	// If everythingOn is true, all features are enabled
	if tg.everythingOn {
		return true
	}

	feature, exists := tg.Toolsets[name]
	if !exists {
		return false
	}
	return feature.Enabled
}

func (tg *ToolsetGroup) EnableToolsets(names []string) error {
	// Special case for "all"
	for _, name := range names {
		if name == "all" {
			tg.everythingOn = true
			break
		}
		err := tg.EnableToolset(name)
		if err != nil {
			return err
		}
	}
	// Do this after to ensure all toolsets are enabled if "all" is present anywhere in list
	if tg.everythingOn {
		for name := range tg.Toolsets {
			err := tg.EnableToolset(name)
			if err != nil {
				return err
			}
		}
		return nil
	}
	return nil
}

func (tg *ToolsetGroup) EnableToolset(name string) error {
	toolset, exists := tg.Toolsets[name]
	if !exists {
		return fmt.Errorf("toolset %s does not exist", name)
	}
	toolset.Enabled = true
	tg.Toolsets[name] = toolset
	return nil
}

func (tg *ToolsetGroup) RegisterTools(s *server.MCPServer) {
	for _, toolset := range tg.Toolsets {
		toolset.RegisterTools(s) // Toolset's RegisterTools now handles disabled filtering
	}
}
