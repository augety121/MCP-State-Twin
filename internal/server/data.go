package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/augety121/mcp-state-twin/internal/engine"
	"github.com/augety121/mcp-state-twin/internal/spec"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const Version = "0.1.0-dev"

type DataPlane struct {
	runtime *engine.Runtime
	handler http.Handler
}

func NewDataPlane(runtime *engine.Runtime) *DataPlane {
	d := &DataPlane{runtime: runtime}
	d.handler = d.buildHandler()
	return d
}

type branchContextKey struct{}

var branchIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)

func (d *DataPlane) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	branch, ok := branchFromPath(r.URL.Path)
	if !ok {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	ctx := context.WithValue(r.Context(), branchContextKey{}, branch)
	d.handler.ServeHTTP(w, r.WithContext(ctx))
}

func branchFromPath(path string) (string, bool) {
	const prefix = "/mcp/"
	if !strings.HasPrefix(path, prefix) {
		return "", false
	}
	branch := strings.Trim(strings.TrimPrefix(path, prefix), "/")
	if !branchIDPattern.MatchString(branch) {
		return "", false
	}
	return branch, true
}

func (d *DataPlane) buildHandler() http.Handler {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "SIMULATED/" + d.runtime.Spec().Metadata.Name,
		Version: Version,
	}, nil)
	closedWorld := false
	for _, toolSpec := range d.runtime.Spec().Tools {
		tool := toolSpec
		annotations := spec.ToolAnnotations(tool)
		server.AddTool(&mcp.Tool{
			Name:         tool.Name,
			Description:  tool.Description,
			InputSchema:  tool.InputSchema,
			OutputSchema: tool.OutputSchema,
			Annotations: &mcp.ToolAnnotations{
				ReadOnlyHint:    annotations.ReadOnly,
				OpenWorldHint:   &closedWorld,
				DestructiveHint: &annotations.Destructive,
			},
		}, func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			branch, ok := ctx.Value(branchContextKey{}).(string)
			if !ok || branch == "" {
				return nil, fmt.Errorf("branch context is missing")
			}
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
