package synthesizeaudio

import (
	"context"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

// NewTool creates an InvokableTool from the synthesizeaudio graph.
func NewTool(ctx context.Context) (tool.InvokableTool, error) {
	g, err := NewGraph(ctx)
	if err != nil {
		return nil, err
	}

	return utils.InferTool("synthesize_audio", "将歌曲信息和播报文本合成为完整音频。需要提供歌曲信息、开场问候语、TTS播报内容和结束语。",
		func(ctx context.Context, input *SynthesizeInput) (*SynthesizeOutput, error) {
			return g.Invoke(ctx, input)
		})
}
