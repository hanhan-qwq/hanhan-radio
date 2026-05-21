package memory

import (
	"context"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

// RecallInput is the input for the recall_memory tool.
type RecallInput struct {
	Keyword string `json:"keyword" jsonschema_description:"搜索关键词（歌名或歌手）"`
	Days    int    `json:"days,omitempty" jsonschema_description:"搜索最近多少天的记录，默认30天"`
}

// RecallOutput is the output for the recall_memory tool.
type RecallOutput struct {
	Records []PlayRecord `json:"records"`
	Count   int          `json:"count"`
}

// NewRecallMemoryTool creates an Agent tool for querying play history.
func NewRecallMemoryTool(store *Store) tool.InvokableTool {
	t, _ := utils.InferTool("recall_memory", "查询历史播放记录。当用户问上次放了什么歌、上次那个歌手是谁等问题时调用。支持按关键词歌名或歌手和时间范围默认30天搜索。",
		func(ctx context.Context, input *RecallInput) (*RecallOutput, error) {
			days := input.Days
			if days <= 0 {
				days = 30
			}
			records, err := store.SearchPlays(input.Keyword, days)
			if err != nil {
				return nil, err
			}
			return &RecallOutput{
				Records: records,
				Count:   len(records),
			}, nil
		})
	return t
}
