package agentruntime

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	"hanhan-radio/agentruntime/log"
	"hanhan-radio/agentruntime/synthesizeaudio"
	"hanhan-radio/agentruntime/tts"
)

const outputDir = "output"

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

	ttsClient := tts.NewClient(tts.Config{
		APIKey: os.Getenv("MIMO_API_KEY"),
	})

	audioGraph, err := synthesizeaudio.NewGraph(ctx, ttsClient, outputDir)
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
		log.L().Debugw("agent response", "content", msg.Content)

		var segments []synthesizeaudio.SongSegment
		if err := json.Unmarshal([]byte(msg.Content), &segments); err != nil {
			return nil, fmt.Errorf("parse agent JSON output: %w\nraw: %s", err, msg.Content)
		}

		log.L().Infow("parsed", "node", "parse_result", "segments", len(segments),
			"first_title", segments[0].Title, "first_artist", segments[0].Artist)

		return &synthesizeaudio.SynthesizeInput{Segments: segments}, nil
	}))

	// Step 4: synthesize audio (TTS + concat)
	g.AddLambdaNode("synthesize_audio", compose.InvokableLambda(func(ctx context.Context, in *synthesizeaudio.SynthesizeInput) (*synthesizeaudio.SynthesizeOutput, error) {
		log.L().Debugw("synthesize_input", "segments", len(in.Segments))
		out, err := audioGraph.Invoke(ctx, in)
		if err != nil {
			return nil, err
		}
		log.L().Infow("done", "node", "synthesize_audio", "output", out.AudioFile, "duration", out.Duration)
		return out, nil
	}))

	g.AddEdge(compose.START, "build_messages")
	g.AddEdge("build_messages", "agent")
	g.AddEdge("agent", "parse_result")
	g.AddEdge("parse_result", "synthesize_audio")
	g.AddEdge("synthesize_audio", compose.END)

	return g.Compile(ctx)
}
