package synthesizeaudio

import "context"

type ctxKeyOutputDir struct{}

// WithOutputDir injects an output directory into ctx.
// synthesize_audio nodes read this value at runtime.
func WithOutputDir(ctx context.Context, dir string) context.Context {
	return context.WithValue(ctx, ctxKeyOutputDir{}, dir)
}

// OutputDirFromCtx reads the output directory from ctx.
func OutputDirFromCtx(ctx context.Context) string {
	s, _ := ctx.Value(ctxKeyOutputDir{}).(string)
	return s
}

// SongSegment is a single song with its segue (串词) and file path in the episode.
type SongSegment struct {
	Title    string `json:"title"`
	Artist   string `json:"artist"`
	Segue    string `json:"segue"`
	FilePath string `json:"file_path"` // local audio file path, from select_song
}

// SynthesizeInput is the input for the synthesize_audio node.
type SynthesizeInput struct {
	Segments  []SongSegment `json:"segments"`
	OutputDir string        `json:"output_dir"`
}

// synthesizeAudioReady is the intermediate state between tts and concat nodes.
type synthesizeAudioReady struct {
	VoicePath  string        // TTS generated voice file
	MusicPath  string        // original music file path (from SongSegment.FilePath)
	OutputPath string        // final output path after concat
	Segments   []SongSegment // for propagation to output
}

// SynthesizeOutput is the output of the synthesize_audio node.
type SynthesizeOutput struct {
	AudioFile string        `json:"audio_file"`
	Duration  int           `json:"duration"`
	Segments  []SongSegment `json:"-"`
}
