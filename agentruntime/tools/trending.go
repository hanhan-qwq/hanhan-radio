package tools

import (
	"context"
	"encoding/json"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

var mockTrending = []string{
	"周杰伦新专辑预告",
	"今日全国大范围降温",
	"世界杯决赛今夜开打",
}

func Trending() tool.BaseTool {
	return &trendingTool{}
}

type trendingTool struct{}

func (t *trendingTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "get_trending",
		Desc: "获取当前热搜榜前几条，当你觉得有必要在串场词里提到今天发生的事时调用",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{}),
	}, nil
}

func (t *trendingTool) InvokableRun(_ context.Context, _ string, _ ...tool.Option) (string, error) {
	b, _ := json.Marshal(mockTrending)
	return string(b), nil
}
