package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	digitaltwin "github.com/aosanya/mwanachama-backend-digitaltwin"
	"github.com/aosanya/mwanachama-backend-digitaltwin/routes"
)

const InstanceArgument = "instance_id"

type ManagerResolver func(ctx context.Context, instanceID string) (digitaltwin.TwinManager, error)

type schemaOnly struct{ digitaltwin.TwinManager }

func RegisterTools(server *mcp.Server, resolve ManagerResolver, m routes.Mount) error {
	if resolve == nil {
		return errors.New("digitaltwin mcp: a manager resolver is required")
	}
	declared, err := routes.BuildTools(schemaOnly{}, m)
	if err != nil {
		return err
	}
	for _, t := range declared {
		schema, err := withInstance(t.InputSchema)
		if err != nil {
			return fmt.Errorf("digitaltwin mcp: %s input schema: %w", t.Name, err)
		}
		server.AddTool(&mcp.Tool{
			Name:        t.Name,
			Title:       t.Title,
			Description: t.Description,
			InputSchema: schema,
		}, handlerFor(t.Name, resolve, m))
	}
	registerBatch(server, resolve, m)
	return nil
}

func withInstance(raw json.RawMessage) (map[string]any, error) {
	schema := map[string]any{}
	if err := json.Unmarshal(raw, &schema); err != nil {
		return nil, err
	}
	props, _ := schema["properties"].(map[string]any)
	if props == nil {
		props = map[string]any{}
	}
	props[InstanceArgument] = map[string]any{
		"type":        "string",
		"description": "The instance whose twin to act on.",
	}
	schema["properties"] = props
	required, _ := schema["required"].([]any)
	schema["required"] = append(required, InstanceArgument)
	schema["type"] = "object"
	return schema, nil
}

func handlerFor(name string, resolve ManagerResolver, m routes.Mount) mcp.ToolHandler {
	return func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		out, err := call(ctx, name, req.Params.Arguments, resolve, m)
		if err != nil {
			return &mcp.CallToolResult{
				IsError: true,
				Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
			}, nil
		}
		structured, text := envelope(name, out)
		return &mcp.CallToolResult{
			Content:           []mcp.Content{&mcp.TextContent{Text: text}},
			StructuredContent: structured,
		}, nil
	}
}

func call(ctx context.Context, name string, raw json.RawMessage, resolve ManagerResolver, m routes.Mount) (any, error) {
	args := map[string]json.RawMessage{}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &args); err != nil {
			return nil, fmt.Errorf("arguments must be an object: %w", err)
		}
	}
	var instance string
	if err := json.Unmarshal(args[InstanceArgument], &instance); err != nil || instance == "" {
		return nil, fmt.Errorf("%s is required", InstanceArgument)
	}
	delete(args, InstanceArgument)
	tm, err := resolve(ctx, instance)
	if err != nil {
		return nil, err
	}
	tools, err := routes.BuildTools(tm, m)
	if err != nil {
		return nil, err
	}
	rest, err := json.Marshal(args)
	if err != nil {
		return nil, err
	}
	for _, t := range tools {
		if t.Name == name {
			return t.Invoke(ctx, rest)
		}
	}
	return nil, fmt.Errorf("no tool named %s", name)
}

func envelope(name string, out any) (any, string) {
	rv := reflect.ValueOf(out)
	if rv.IsValid() && (rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array) {
		return map[string]any{"items": out}, fmt.Sprintf("Found %d item(s)", rv.Len())
	}
	if !rv.IsValid() {
		return map[string]any{}, fmt.Sprintf("%s completed", name)
	}
	return out, fmt.Sprintf("%s completed", name)
}
