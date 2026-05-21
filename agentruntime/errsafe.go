package agentruntime

import (
	"context"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// errSafeTool wraps an InvokableTool so errors are returned as result strings
// instead of propagating up and crashing the agent.
type errSafeTool struct {
	inner tool.InvokableTool
}

func (t *errSafeTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return t.inner.Info(ctx)
}

func (t *errSafeTool) InvokableRun(ctx context.Context, jsonArgs string, opts ...tool.Option) (string, error) {
	result, err := t.inner.InvokableRun(ctx, jsonArgs, opts...)
	if err != nil {
		info, _ := t.inner.Info(ctx)
		name := "tool"
		if info != nil {
			name = info.Name
		}
		return name + " 出错: " + err.Error(), nil
	}
	return result, nil
}

// safeTool wraps a tool to prevent errors from crashing the agent.
// Tool errors are converted to result strings so the agent can adapt.
func safeTool(inner tool.InvokableTool) tool.InvokableTool {
	return &errSafeTool{inner: inner}
}
