package log

import (
	"context"

	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/schema"
)

// InitCallbacks registers a global Eino callback handler for structured logging.
// Call once at startup, before any graph execution.
func InitCallbacks() {
	h := callbacks.NewHandlerBuilder().
		OnStartFn(onStart).
		OnEndFn(onEnd).
		OnErrorFn(onError).
		Build()
	callbacks.AppendGlobalHandlers(h)
}

func onStart(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
	if info == nil {
		return ctx
	}

	switch info.Name {
	case "build_messages":
		s, _ := input.(string)
		sugared.Infow("start", "node", info.Name, "input_len", len(s))

	case "agent":
		msgs, _ := input.([]*schema.Message)
		sugared.Infow("start", "node", info.Name, "msg_count", len(msgs))

	case "load_songs":
		sugared.Infow("start", "node", info.Name)

	case "llm_select":
		sugared.Debugw("start", "node", info.Name)

	case "tts":
		sugared.Infow("start", "node", info.Name)

	case "concat":
		sugared.Infow("start", "node", info.Name)
	}

	return ctx
}

func onEnd(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
	if info == nil {
		return ctx
	}

	switch info.Name {
	case "build_messages":
		msgs, _ := output.([]*schema.Message)
		sugared.Infow("done", "node", info.Name, "msg_count", len(msgs))

	case "agent":
		sugared.Infow("done", "node", info.Name)

	case "parse_result":
		sugared.Infow("done", "node", info.Name)

	case "load_songs":
		sugared.Infow("done", "node", info.Name)

	case "llm_select":
		sugared.Infow("done", "node", info.Name)

	case "tts":
		sugared.Infow("done", "node", info.Name)

	case "concat":
		sugared.Infow("done", "node", info.Name)
	}

	return ctx
}

func onError(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
	if info == nil {
		sugared.Errorw("error", "err", err)
		return ctx
	}
	sugared.Errorw("error", "node", info.Name, "err", err)
	return ctx
}
