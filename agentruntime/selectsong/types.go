package selectsong

// SelectSongInput is the input for the select_song tool.
type SelectSongInput struct {
	Mood     string `json:"mood,omitempty" jsonschema_description:"用户心情，如轻松、愉快、伤感、兴奋"`
	Genre    string `json:"genre,omitempty" jsonschema_description:"音乐风格，如流行、摇滚、爵士、电子、古典"`
	Artist   string `json:"artist,omitempty" jsonschema_description:"指定歌手名称"`
	Language string `json:"language,omitempty" jsonschema_description:"语言偏好，如中文、英文、日文、韩文"`
}

// SelectSongOutput is the output of the select_song tool.
type SelectSongOutput struct {
	Title    string `json:"title"`
	Artist   string `json:"artist"`
	Album    string `json:"album,omitempty"`
	AudioURL string `json:"audio_url"`
	Duration int    `json:"duration"`
	Genre    string `json:"genre,omitempty"`
}
