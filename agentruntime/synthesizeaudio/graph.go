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
		return &SynthesizeOutput{
			AudioFile: fmt.Sprintf("/output/episode_%s.mp3", in.Song.Title),
			Duration:  in.Song.Duration + 60,
		}, nil
	}))

	g.AddEdge(compose.START, "mock_synthesize")
	g.AddEdge("mock_synthesize", compose.END)

	return g.Compile(ctx)
}
