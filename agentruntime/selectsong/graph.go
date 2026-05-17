package selectsong

import (
	"context"

	"github.com/cloudwego/eino/compose"
)

// NewGraph builds the select_song sub-graph.
// Phase 1: mock data, returns a fixed song.
func NewGraph(ctx context.Context) (compose.Runnable[*SelectSongInput, *SelectSongOutput], error) {
	g := compose.NewGraph[*SelectSongInput, *SelectSongOutput]()

	g.AddLambdaNode("mock_select", compose.InvokableLambda(func(ctx context.Context, in *SelectSongInput) (*SelectSongOutput, error) {
		return &SelectSongOutput{
			Title:    "夜曲",
			Artist:   "周杰伦",
			Album:    "十一月的肖邦",
			AudioURL: "/music/周杰伦/夜曲.mp3",
			Duration: 231,
			Genre:    "流行",
		}, nil
	}))

	g.AddEdge(compose.START, "mock_select")
	g.AddEdge("mock_select", compose.END)

	return g.Compile(ctx)
}
