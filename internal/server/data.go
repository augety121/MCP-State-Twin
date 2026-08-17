package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/augety121/mcp-state-twin/internal/engine"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const Version = "0.1.0-dev"

type DataPlane struct {
	runtime  *engine.Runtime
	handlers sync.Map
}

func NewDataPlane(runtime *engine.Runtime) *DataPlane {
	return &DataPlane{runtime: runtime}
}

func (d *DataPlane) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	branch, ok := branchFromPath(r.URL.Path)
	if !ok {
		http.NotFound(w, r)
		return
	}
	handler, _ := d.handlers.LoadOrStore(branch, d.branchHandler(branch))
	handler.(http.Handler).ServeHTTP(w, r)
}

func branchFromPath(path string) (string, bool) {
	const prefix = "/mcp/"
	if !strings.HasPrefix(path, prefix) {
		return "", false
	}
	branch := strings.Trim(strings.TrimPrefix(path, prefix), "/")
	if branch == "" || strings.Contains(branch, "/") || len(branch) > 128 {
		return "", false
	}
	return branch, true
}

func (d *DataPlane) branchHandler(branch string) http.Handler {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "SIMULATED/" + d.runtime.Spec().Metadata.Name,
		Version: Version,
	}, nil)
	closedWorld := false
	for _, toolSpec := range d.runtime.Spec().Tools {
		tool := toolSpec
		readOnly := len(tool.Effects) == 0
		destructive := false
		for _, effect := range tool.Effects {
			if effect.Op == "update" || effect.Op == "delete" {
				destructive = true
			}
		}
		server.AddTool(&mcp.Tool{
			Name:         tool.Name,
			Description:  tool.Description,
			InputSchema:  tool.InputSchema,
			OutputSchema: tool.OutputSchema,
			Annotations: &mcp.ToolAnnotations{
				ReadOnlyHint:    readOnly,
				OpenWorldHint:   &closedWorld,
				DestructiveHint: &destructive,
			},
		}, func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var input map[string]any
			if request.Params != nil && len(request.Params.Arguments) > 0 {
				if err := json.Unmarshal(request.Params.Arguments, &input); err != nil {
					return toolError("INVALID_INPUT", "arguments are not a JSON object: "+err.Error()), nil
				}
			}
			if input == nil {
				input = make(map[string]any)
			}
			result, err := d.runtime.Call(ctx, branch, tool.Name, input)
			if err != nil {
				return nil, fmt.Errorf("execute twin tool: %w", err)
			}
			data, err := json.Marshal(result.Result)
			if err != nil {
				return nil, fmt.Errorf("marshal twin result: %w", err)
			}
			return &mcp.CallToolResult{
				Content:           []mcp.Content{&mcp.TextContent{Text: string(data)}},
				StructuredContent: result.Result,
				IsError:           engine.IsDomainError(result),
			}, nil
		})
	}
	return mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return server
	}, &mcp.StreamableHTTPOptions{Stateless: true, JSONResponse: true})
}

func toolError(code, message string) *mcp.CallToolResult {
	value := map[string]any{"error": map[string]any{"code": code, "message": message}}
	data, _ := json.Marshal(value)
	return &mcp.CallToolResult{
		Content:           []mcp.Content{&mcp.TextContent{Text: string(data)}},
		StructuredContent: value,
		IsError:           true,
	}
}
