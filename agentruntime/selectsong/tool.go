package selectsong

import (
	"context"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

// NewTool creates an InvokableTool from the selectsong chain.
func NewTool(ctx context.Context, cm model.BaseChatModel) (tool.InvokableTool, error) {
	chain, err := NewChain(ctx, cm)
	if err != nil {
		return nil, err
	}

	return utils.InferTool("select_song", "根据用户的心情、风格偏好选择合适的歌曲。调用后返回歌曲的名称、歌手、音频路径等信息。",
		func(ctx context.Context, input *SelectSongInput) (*SelectSongOutput, error) {
			return chain.Invoke(ctx, input)
		})
}
