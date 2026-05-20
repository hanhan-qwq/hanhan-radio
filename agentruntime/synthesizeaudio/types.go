package synthesizeaudio

// SongSegment is a single song with its segue (串词) in the episode.
type SongSegment struct {
	Title  string `json:"title"`
	Artist string `json:"artist"`
	Segue  string `json:"segue"`
}

// SynthesizeInput is the input for the synthesize_audio node.
type SynthesizeInput struct {
	Segments []SongSegment `json:"segments"`
}

// SynthesizeOutput is the output of the synthesize_audio node.
type SynthesizeOutput struct {
	AudioFile string `json:"audio_file"`
	Duration  int    `json:"duration"`
}
