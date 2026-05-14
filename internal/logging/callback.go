package logging

import (
	"context"
	"fmt"
	"log"

	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
)

func BuildLogHandler() callbacks.Handler {
	return callbacks.NewHandlerBuilder().
		OnStartFn(func(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
			if mi := model.ConvCallbackInput(input); mi != nil {
				log.Printf("[%s] ChatModel start, %d messages", info.Name, len(mi.Messages))
				return ctx
			}
			if ti := tool.ConvCallbackInput(input); ti != nil {
				log.Printf("[%s] Tool call start, args: %s", info.Name, ti.ArgumentsInJSON)
				return ctx
			}
			return ctx
		}).
		OnEndFn(func(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
			if mo := model.ConvCallbackOutput(output); mo != nil {
				tokens := ""
				if mo.TokenUsage != nil {
					tokens = fmt.Sprintf(", tokens: in=%d out=%d", mo.TokenUsage.PromptTokens, mo.TokenUsage.CompletionTokens)
				}
				log.Printf("[%s] ChatModel end%s", info.Name, tokens)
				return ctx
			}
			if to := tool.ConvCallbackOutput(output); to != nil {
				log.Printf("[%s] Tool call end, result: %s", info.Name, truncate(to.Response, 200))
				return ctx
			}
			return ctx
		}).
		OnErrorFn(func(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
			log.Printf("[%s] ERROR: %v", info.Name, err)
			return ctx
		}).
		Build()
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
