package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	digitaltwin "github.com/aosanya/mwanachama-backend-digitaltwin"
	"github.com/aosanya/mwanachama-backend-digitaltwin/routes"
)

const (
	batchTool   = "digitaltwin_observation_record_batch"
	batchAction = "digitaltwin.observation.record_batch"
)

func registerBatch(server *mcp.Server, resolve ManagerResolver, m routes.Mount) {
	server.AddTool(&mcp.Tool{
		Name:        batchTool,
		Title:       "Record observations",
		Description: "Store many measurements in one call, which is the path a source that samples often should take. All or nothing: if one is refused, none are stored.",
		InputSchema: map[string]any{
			"type":     "object",
			"required": []any{InstanceArgument, "items"},
			"properties": map[string]any{
				InstanceArgument: map[string]any{"type": "string", "description": "The instance whose twin to act on."},
				"items": map[string]any{
					"type":        "array",
					"description": "The measurements, each with node_id, metric_name, value and recorded_at.",
					"items":       map[string]any{"type": "object"},
				},
			},
		},
	}, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		out, err := recordBatch(ctx, req.Params.Arguments, resolve, m)
		if err != nil {
			return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}}}, nil
		}
		structured, text := envelope(batchTool, out)
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: text}}, StructuredContent: structured}, nil
	})
}

func recordBatch(ctx context.Context, raw json.RawMessage, resolve ManagerResolver, m routes.Mount) ([]digitaltwin.Observation, error) {
	var args struct {
		InstanceID string                    `json:"instance_id"`
		Items      []digitaltwin.Observation `json:"items"`
	}
	if err := json.Unmarshal(raw, &args); err != nil {
		return nil, fmt.Errorf("arguments: %w", err)
	}
	if args.InstanceID == "" {
		return nil, fmt.Errorf("%s is required", InstanceArgument)
	}
	if m.Authorize != nil {
		if err := m.Authorize(ctx, batchAction); err != nil {
			return nil, err
		}
	}
	tm, err := resolve(ctx, args.InstanceID)
	if err != nil {
		return nil, err
	}
	return tm.RecordObservations(ctx, args.Items)
}
