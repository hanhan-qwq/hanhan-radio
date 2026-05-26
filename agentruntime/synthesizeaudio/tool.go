package synthesizeaudio

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"

	"hanhan-radio/agentruntime/log"
	"hanhan-radio/agentruntime/tts"
)

// NewTool creates an InvokableTool that synthesizes audio from segments.
// Synthesis runs asynchronously — the tool returns immediately and TTS+ffmpeg runs in background.
func NewTool(ttsClient *tts.Client, defaultOutputDir string) (tool.InvokableTool, error) {
	return utils.InferTool[*SynthesizeInput, *SynthesizeOutput](
		"synthesize_audio",
		"将电台串词合成为语音，并与对应的音乐文件拼接成完整的电台音频。调用后返回音频文件路径和时长。",
		func(ctx context.Context, in *SynthesizeInput) (*SynthesizeOutput, error) {
			if len(in.Segments) == 0 {
				return nil, fmt.Errorf("no segments to synthesize")
			}

			seg := in.Segments[0]

			dir := OutputDirFromCtx(ctx)
			if dir == "" {
				dir = defaultOutputDir
			}
			_ = os.MkdirAll(dir, 0755)

			outputPath := filepath.Join(dir, "final.mp3")

			go func() {
				bgCtx := context.Background()
				log.L().Infow("tts_start", "segue_len", len(seg.Segue), "title", seg.Title)

				voicePath := filepath.Join(dir, "voice_0.wav")
				audioBytes, err := ttsClient.Synthesize(bgCtx, seg.Segue)
				if err != nil {
					log.L().Errorw("tts_synthesize_failed", "err", err)
					return
				}
				if err := os.WriteFile(voicePath, audioBytes, 0644); err != nil {
					log.L().Errorw("write_voice_failed", "err", err)
					return
				}

				log.L().Infow("tts_done", "voice_bytes", len(audioBytes), "voice_path", voicePath)
				log.L().Infow("concat_start", "voice", voicePath, "music", seg.FilePath)

				if err := Concat(bgCtx, voicePath, seg.FilePath, outputPath); err != nil {
					log.L().Errorw("concat_failed", "err", err)
					return
				}

				dur, err := ProbeDuration(bgCtx, outputPath)
				if err != nil {
					log.L().Warnw("probe_duration_failed", "err", err)
				}

				log.L().Infow("synthesize_done", "output", outputPath, "duration", dur)
			}()

			return &SynthesizeOutput{
				AudioFile: outputPath,
				Segments:  in.Segments,
			}, nil
		})
}
