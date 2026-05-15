package tools

import (
	"context"
	"encoding/json"
	"os"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

type WeatherResult struct {
	City        string `json:"city"`
	Condition   string `json:"condition"`
	Temperature string `json:"temperature"`
}

func Weather() tool.BaseTool {
	type input struct {
		City string `json:"city" jsonschema:"description=城市名，默认从环境变量读取"`
	}
	t, _ := utils.InferTool(
		"get_weather",
		"获取当前城市的天气情况，当你想在串场词里聊天气或根据天气推荐歌曲时调用",
		func(ctx context.Context, in input) (string, error) {
			city := in.City
			if city == "" {
				city = os.Getenv("RADIO_CITY")
			}
			if city == "" {
				city = "北京"
			}

			result := WeatherResult{
				City:        city,
				Condition:   "小雨",
				Temperature: "12°C",
			}
			b, _ := json.Marshal(result)
			return string(b), nil
		},
	)
	return t
}
