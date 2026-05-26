package selectsong

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/components/model"
	ttool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

// NewTool creates an InvokableTool from the selectsong chain.
// After selecting a song it interrupts for user confirmation (HITL).
// On resume: "confirm" → return selected song, "skip" → prompt Agent to re-select.
// rctx is optional — pass nil if no profile / recent-play context is available.
func NewTool(ctx context.Context, cm model.BaseChatModel, searcher *Searcher, rctx *RerankContext) (ttool.InvokableTool, error) {
	chain, err := NewChain(ctx, cm, searcher, rctx)
	if err != nil {
		return nil, err
	}

	// Closure captures the last selected song across interrupt/resume cycles.
	// The tool instance survives the full agent lifecycle.
	var savedOutput *SelectSongOutput

	return utils.InferTool("select_song", "根据用户的心情、风格偏好从曲库中搜索并选择合适的歌曲。支持语义搜索，调用后返回歌曲的名称、歌手、音频路径等信息。",
		func(ctx context.Context, input *SelectSongInput) (*SelectSongOutput, error) {
			wasInterrupted, _, _ := ttool.GetInterruptState[any](ctx)
			if wasInterrupted {
				isTarget, hasData, action := ttool.GetResumeContext[string](ctx)
				if !isTarget {
					// Another component is the resume target — re-interrupt to preserve our state.
					return nil, ttool.Interrupt(ctx, nil)
				}
				if !hasData || action == "" {
					action = "confirm"
				}
				if action == "skip" {
					savedOutput = nil
					return nil, fmt.Errorf("SKIP_SONG: 用户想换一首歌，请重新搜索推荐")
				}
				return savedOutput, nil
			}

			output, err := chain.Invoke(ctx, input)
			if err != nil {
				return nil, err
			}
			savedOutput = output

			return nil, ttool.Interrupt(ctx,
				ConfirmSongInfo{
					Title:  output.Title,
					Artist: output.Artist,
					Genre:  output.Genre,
				},
			)
		})
}
