package agenteval

import (
	"github.com/augety121/mcp-state-twin/internal/agenthost"
	"github.com/augety121/mcp-state-twin/internal/bundle"
	"github.com/augety121/mcp-state-twin/internal/spec"
	"github.com/augety121/mcp-state-twin/internal/task"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Preflight is static admission only: no model availability checks or world
// execution. RunMock additionally discovers tools through the actual MCP path.
func Preflight(t *task.Task, b *bundle.Artifact, c *RunConfig) error {
	if err := c.Validate(); err != nil {
		return err
	}
	if err := Admit(t, b); err != nil {
		return err
	}
	twin, err := spec.Decode(b.Files[b.Manifest.Spec])
	if err != nil {
		return err
	}
	var tools []*mcp.Tool
	for _, s := range twin.Tools {
		tools = append(tools, &mcp.Tool{Name: s.Name, Description: s.Description, InputSchema: s.InputSchema})
	}
	host, err := agenthost.New(c.Model, c.MaxOutputTokens, t, tools)
	if err != nil {
		return err
	}
	host.Stop()
	return nil
}
