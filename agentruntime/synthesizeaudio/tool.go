package synthesizeaudio

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	ttool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"

	"hanhan-radio/agentruntime/log"
	"hanhan-radio/agentruntime/tts"
)

// NewTool creates an InvokableTool that synthesizes audio from segments.
// Before synthesis it interrupts for user confirmation (HITL).
// On resume: "confirm" → proceed, "skip" → prompt Agent to re-select, "cancel" → abort.
func NewTool(ttsClient *tts.Client, defaultOutputDir string) (ttool.InvokableTool, error) {
	return utils.InferTool[*SynthesizeInput, *SynthesizeOutput](
		"synthesize_audio",
		"将电台串词合成为语音，并与对应的音乐文件拼接成完整的电台音频。调用后返回音频文件路径和时长。",
		func(ctx context.Context, in *SynthesizeInput) (*SynthesizeOutput, error) {
			wasInterrupted, _, _ := ttool.GetInterruptState[any](ctx)
			if wasInterrupted {
				isTarget, hasData, action := ttool.GetResumeContext[string](ctx)
				if !isTarget {
					return nil, ttool.Interrupt(ctx, nil)
				}
				if !hasData || action == "" {
					action = "confirm"
				}
				switch action {
				case "confirm":
					// proceed to synthesis
				case "skip":
					return nil, fmt.Errorf("SKIP_SONG: 用户想换一首歌，请调用 select_song 重新选歌")
				case "cancel":
					return nil, fmt.Errorf("CANCEL: 用户取消了电台合成")
				}
			} else {
				if len(in.Segments) == 0 {
					return nil, fmt.Errorf("no segments to synthesize")
				}
				return nil, ttool.Interrupt(ctx, ConfirmInfo{Segments: in.Segments})
			}

			if len(in.Segments) == 0 {
				return nil, fmt.Errorf("no segments to synthesize")
			}

			seg := in.Segments[0]

			dir := OutputDirFromCtx(ctx)
			if dir == "" {
				dir = defaultOutputDir
			}
			if err := os.MkdirAll(dir, 0755); err != nil {
				return nil, fmt.Errorf("create output dir %s: %w", dir, err)
			}

			log.L().Infow("tts_start", "segue_len", len(seg.Segue), "title", seg.Title)

			voicePath := filepath.Join(dir, "voice_0.wav")
			audioBytes, err := ttsClient.Synthesize(ctx, seg.Segue)
			if err != nil {
				return nil, fmt.Errorf("tts synthesize: %w", err)
			}
			if err := os.WriteFile(voicePath, audioBytes, 0644); err != nil {
				return nil, fmt.Errorf("write voice file: %w", err)
			}

			log.L().Infow("tts_done", "voice_bytes", len(audioBytes), "voice_path", voicePath)

			outputPath := filepath.Join(dir, "final.mp3")

			log.L().Infow("concat_start", "voice", voicePath, "music", seg.FilePath)

			if err := Concat(ctx, voicePath, seg.FilePath, outputPath); err != nil {
				return nil, fmt.Errorf("concat audio: %w", err)
			}

			dur, err := ProbeDuration(ctx, outputPath)
			if err != nil {
				return nil, fmt.Errorf("probe duration: %w", err)
			}

			log.L().Infow("synthesize_done", "output", outputPath, "duration", dur)

			return &SynthesizeOutput{
				AudioFile: outputPath,
				Duration:  dur,
				Segments:  in.Segments,
			}, nil
		})
}
