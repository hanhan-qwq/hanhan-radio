package selectsong

import (
	"context"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

// NewTool creates an InvokableTool from the selectsong chain.
// rctx is optional — pass nil if no profile / recent-play context is available.
func NewTool(ctx context.Context, cm model.BaseChatModel, searcher *Searcher, rctx *RerankContext) (tool.InvokableTool, error) {
	chain, err := NewChain(ctx, cm, searcher, rctx)
	if err != nil {
		return nil, err
	}

	return utils.InferTool("select_song", "根据用户的心情、风格偏好从曲库中搜索并选择合适的歌曲。支持语义搜索，调用后返回歌曲的名称、歌手、音频路径等信息。",
		func(ctx context.Context, input *SelectSongInput) (*SelectSongOutput, error) {
			return chain.Invoke(ctx, input)
		})
}
