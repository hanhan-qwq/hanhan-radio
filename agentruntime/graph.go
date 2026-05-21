package agentruntime

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

// NewGraph builds the main radio pipeline:
//
//	string → build_messages → agent → *schema.Message
//
// The agent drives the full flow: select_song → duckduckgo_search → write segue → synthesize_audio.
// The returned Message contains JSON that the caller parses for episode metadata.
func NewGraph(ctx context.Context) (compose.Runnable[string, *schema.Message], error) {
	reactAgent, err := NewAgentWithDefaults(ctx)
	if err != nil {
		return nil, fmt.Errorf("create agent: %w", err)
	}

	agentLambda, err := compose.AnyLambda(reactAgent.Generate, reactAgent.Stream, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("wrap agent as lambda: %w", err)
	}

	g := compose.NewGraph[string, *schema.Message]()

	g.AddLambdaNode("build_messages", compose.InvokableLambda(func(ctx context.Context, input string) ([]*schema.Message, error) {
		return []*schema.Message{schema.UserMessage(input)}, nil
	}))

	g.AddLambdaNode("agent", agentLambda)

	g.AddEdge(compose.START, "build_messages")
	g.AddEdge("build_messages", "agent")
	g.AddEdge("agent", compose.END)

	return g.Compile(ctx)
}
