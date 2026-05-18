package agentruntime

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	"hanhan-radio/agentruntime/synthesizeaudio"
)

// NewGraph builds the main radio pipeline:
//
//	string → build_messages → agent(AnyLambda) → parse_result → synthesize_audio → *SynthesizeOutput
func NewGraph(ctx context.Context) (compose.Runnable[string, *synthesizeaudio.SynthesizeOutput], error) {
	reactAgent, err := NewAgentWithDefaults(ctx)
	if err != nil {
		return nil, fmt.Errorf("create agent: %w", err)
	}

	agentLambda, err := compose.AnyLambda(reactAgent.Generate, reactAgent.Stream, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("wrap agent as lambda: %w", err)
	}

	audioGraph, err := synthesizeaudio.NewGraph(ctx)
	if err != nil {
		return nil, fmt.Errorf("create synthesize graph: %w", err)
	}

	g := compose.NewGraph[string, *synthesizeaudio.SynthesizeOutput]()

	// Step 1: wrap user input as messages
	g.AddLambdaNode("build_messages", compose.InvokableLambda(func(ctx context.Context, input string) ([]*schema.Message, error) {
		return []*schema.Message{schema.UserMessage(input)}, nil
	}))

	// Step 2: ReAct agent (Generate/Stream wrapped as AnyLambda)
	g.AddLambdaNode("agent", agentLambda)

	// Step 3: parse agent's JSON output into SynthesizeInput
	g.AddLambdaNode("parse_result", compose.InvokableLambda(func(ctx context.Context, msg *schema.Message) (*synthesizeaudio.SynthesizeInput, error) {
		// print agent output for user visibility
		fmt.Println(msg.Content)

		var si synthesizeaudio.SynthesizeInput
		if err := json.Unmarshal([]byte(msg.Content), &si); err != nil {
			return nil, fmt.Errorf("parse agent JSON output: %w\nraw: %s", err, msg.Content)
		}
		return &si, nil
	}))

	// Step 4: synthesize audio
	g.AddLambdaNode("synthesize_audio", compose.InvokableLambda(func(ctx context.Context, in *synthesizeaudio.SynthesizeInput) (*synthesizeaudio.SynthesizeOutput, error) {
		return audioGraph.Invoke(ctx, in)
	}))

	g.AddEdge(compose.START, "build_messages")
	g.AddEdge("build_messages", "agent")
	g.AddEdge("agent", "parse_result")
	g.AddEdge("parse_result", "synthesize_audio")
	g.AddEdge("synthesize_audio", compose.END)

	return g.Compile(ctx)
}
