package synthesizeaudio

import "hanhan-radio/agentruntime/selectsong"

// SynthesizeInput is the input for the synthesize_audio tool.
type SynthesizeInput struct {
	Song       selectsong.SelectSongOutput `json:"song" jsonschema_description:"要合成的歌曲信息"`
	Greeting   string                      `json:"greeting" jsonschema_description:"开场问候语"`
	TTSContent string                      `json:"tts_content" jsonschema_description:"TTS 朗读的播报内容"`
	Outro      string                      `json:"outro,omitempty" jsonschema_description:"结束语"`
}

// SynthesizeOutput is the output of the synthesize_audio tool.
type SynthesizeOutput struct {
	AudioFile string `json:"audio_file"`
	Duration  int    `json:"duration"`
}
