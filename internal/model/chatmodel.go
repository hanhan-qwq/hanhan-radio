package model

import (
	"context"
	"log"
	"os"

	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino/components/model"
	arkModel "github.com/volcengine/volcengine-go-sdk/service/arkruntime/model"
)

func NewArkModel() model.ToolCallingChatModel {
	apiKey := os.Getenv("ARK_API_KEY")
	modelName := os.Getenv("ARK_MODEL")

	if apiKey == "" {
		log.Fatal("ARK_API_KEY is required")
	}
	if modelName == "" {
		log.Fatal("ARK_MODEL is required")
	}

	cm, err := ark.NewChatModel(context.Background(), &ark.ChatModelConfig{
		APIKey: apiKey,
		Model:  modelName,
		Thinking: &arkModel.Thinking{
			Type: arkModel.ThinkingTypeDisabled,
		},
	})
	if err != nil {
		log.Fatalf("ark.NewChatModel failed: %v", err)
	}
	return cm
}
