package synthesizeaudio

// SongSegment is a single song with its segue (串词) and file path in the episode.
type SongSegment struct {
	Title    string `json:"title"`
	Artist   string `json:"artist"`
	Segue    string `json:"segue"`
	FilePath string `json:"file_path"` // local audio file path, from select_song
}

// SynthesizeInput is the input for the synthesize_audio node.
type SynthesizeInput struct {
	Segments []SongSegment `json:"segments"`
}

// synthesizeAudioReady is the intermediate state between tts and concat nodes.
type synthesizeAudioReady struct {
	VoicePath  string // TTS generated voice file
	MusicPath  string // original music file path (from SongSegment.FilePath)
	OutputPath string // final output path after concat
}

// SynthesizeOutput is the output of the synthesize_audio node.
type SynthesizeOutput struct {
	AudioFile string `json:"audio_file"`
	Duration  int    `json:"duration"`
}
