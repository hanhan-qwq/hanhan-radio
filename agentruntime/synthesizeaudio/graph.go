package synthesizeaudio

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/compose"
)

// NewGraph builds the synthesize_audio sub-graph.
// Phase 1: mock — generates a fake audio file path.
func NewGraph(ctx context.Context) (compose.Runnable[*SynthesizeInput, *SynthesizeOutput], error) {
	g := compose.NewGraph[*SynthesizeInput, *SynthesizeOutput]()

	g.AddLambdaNode("mock_synthesize", compose.InvokableLambda(func(ctx context.Context, in *SynthesizeInput) (*SynthesizeOutput, error) {
		totalDuration := 0
		for _, seg := range in.Segments {
			totalDuration += len(seg.Segue) / 4 // rough estimate: ~4 chars/sec
		}
		if len(in.Segments) > 0 {
			return &SynthesizeOutput{
				AudioFile: fmt.Sprintf("/output/episode_%s.mp3", in.Segments[0].Title),
				Duration:  totalDuration,
			}, nil
		}
		return &SynthesizeOutput{
			AudioFile: "/output/episode_empty.mp3",
			Duration:  0,
		}, nil
	}))

	g.AddEdge(compose.START, "mock_synthesize")
	g.AddEdge("mock_synthesize", compose.END)

	return g.Compile(ctx)
}
