package synthesizeaudio

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/cloudwego/eino/compose"

	"hanhan-radio/agentruntime/log"
	"hanhan-radio/agentruntime/tts"
)

// NewGraph builds the synthesize_audio sub-graph:
//
//	SynthesizeInput → tts → concat → SynthesizeOutput
func NewGraph(ctx context.Context, ttsClient *tts.Client, outputDir string) (compose.Runnable[*SynthesizeInput, *SynthesizeOutput], error) {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("create output dir %s: %w", outputDir, err)
	}

	g := compose.NewGraph[*SynthesizeInput, *SynthesizeOutput]()

	g.AddLambdaNode("tts", compose.InvokableLambda(func(ctx context.Context, in *SynthesizeInput) (*synthesizeAudioReady, error) {
		if len(in.Segments) == 0 {
			return nil, fmt.Errorf("no segments to synthesize")
		}

		seg := in.Segments[0]

		log.L().Infow("tts_start", "segue_len", len(seg.Segue), "title", seg.Title)

		voicePath := filepath.Join(outputDir, "voice_0.wav")
		audioBytes, err := ttsClient.Synthesize(ctx, seg.Segue)
		if err != nil {
			return nil, fmt.Errorf("tts synthesize: %w", err)
		}
		if err := os.WriteFile(voicePath, audioBytes, 0644); err != nil {
			return nil, fmt.Errorf("write voice file: %w", err)
		}

		log.L().Infow("tts_done", "voice_bytes", len(audioBytes), "voice_path", voicePath)

		outputPath := filepath.Join(outputDir, fmt.Sprintf("ep_%s.mp3", seg.Title))

		return &synthesizeAudioReady{
			VoicePath:  voicePath,
			MusicPath:  seg.FilePath,
			OutputPath: outputPath,
		}, nil
	}))

	g.AddLambdaNode("concat", compose.InvokableLambda(func(ctx context.Context, in *synthesizeAudioReady) (*SynthesizeOutput, error) {
		log.L().Infow("concat_start", "voice", in.VoicePath, "music", in.MusicPath)

		if err := Concat(ctx, in.VoicePath, in.MusicPath, in.OutputPath); err != nil {
			return nil, fmt.Errorf("concat audio: %w", err)
		}

		dur, err := ProbeDuration(ctx, in.OutputPath)
		if err != nil {
			return nil, fmt.Errorf("probe duration: %w", err)
		}

		return &SynthesizeOutput{
			AudioFile: in.OutputPath,
			Duration:  dur,
		}, nil
	}))

	g.AddEdge(compose.START, "tts")
	g.AddEdge("tts", "concat")
	g.AddEdge("concat", compose.END)

	return g.Compile(ctx)
}
