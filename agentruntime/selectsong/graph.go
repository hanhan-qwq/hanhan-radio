package selectsong

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

const systemPrompt = `你是一个音乐推荐助手。根据用户的心情和偏好，从提供的歌曲列表中选出最合适的一首歌。

你必须严格按以下 JSON 格式输出，不要包含其他内容：
{"title":"歌名","artist":"歌手","album":"","audio_url":"/music/歌手/歌名.mp3","duration":240,"genre":""}

选歌规则：
- 优先匹配用户的情绪、风格、语言、歌手偏好
- 如果用户没有明确偏好，选择经典且广受欢迎的歌
- 从提供的歌曲列表中选取，不要编造不存在的歌`

// NewChain builds a 3-step select_song chain:
//
//	load_songs → build_prompt → llm_select
func NewChain(ctx context.Context, cm model.BaseChatModel) (compose.Runnable[*SelectSongInput, *SelectSongOutput], error) {
	c := compose.NewChain[*SelectSongInput, *SelectSongOutput]()

	c.AppendLambda(
		compose.InvokableLambda(func(ctx context.Context, in *SelectSongInput) (*songsLoaded, error) {
			songs, err := loadSongs("data/songs.txt")
			if err != nil {
				return nil, fmt.Errorf("load songs: %w", err)
			}
			return &songsLoaded{Input: in, Songs: songs}, nil
		}),
	)

	c.AppendLambda(
		compose.InvokableLambda(func(ctx context.Context, in *songsLoaded) (*promptReady, error) {
			messages := buildMessages(in.Input, in.Songs)
			return &promptReady{Input: in.Input, Songs: in.Songs, Messages: messages}, nil
		}),
	)

	c.AppendLambda(
		compose.InvokableLambda(func(ctx context.Context, in *promptReady) (*SelectSongOutput, error) {
			msg, err := cm.Generate(ctx, in.Messages)
			if err != nil {
				return nil, fmt.Errorf("llm generate: %w", err)
			}
			var out SelectSongOutput
			if err := json.Unmarshal([]byte(msg.Content), &out); err != nil {
				return nil, fmt.Errorf("parse llm response: %w\ncontent: %s", err, msg.Content)
			}
			// audio_url might be a placeholder; ensure it's set
			if out.AudioURL == "" {
				out.AudioURL = fmt.Sprintf("/music/%s/%s.mp3", out.Artist, out.Title)
			}
			return &out, nil
		}),
	)

	return c.Compile(ctx)
}

func loadSongs(path string) ([]Song, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(data), "\n")
	var songs []Song
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		artist := fields[len(fields)-1]
		title := strings.Join(fields[:len(fields)-1], "")
		songs = append(songs, Song{Title: title, Artist: artist})
	}
	if len(songs) == 0 {
		return nil, fmt.Errorf("no songs found in %s", path)
	}
	return songs, nil
}

func buildMessages(input *SelectSongInput, songs []Song) []*schema.Message {
	var sb strings.Builder
	sb.WriteString("歌曲列表：\n")
	for i, s := range songs {
		sb.WriteString(fmt.Sprintf("%d. %s - %s\n", i+1, s.Title, s.Artist))
	}

	sb.WriteString("\n用户偏好：\n")
	if input.Mood != "" {
		sb.WriteString(fmt.Sprintf("- 心情: %s\n", input.Mood))
	}
	if input.Genre != "" {
		sb.WriteString(fmt.Sprintf("- 风格: %s\n", input.Genre))
	}
	if input.Artist != "" {
		sb.WriteString(fmt.Sprintf("- 歌手: %s\n", input.Artist))
	}
	if input.Language != "" {
		sb.WriteString(fmt.Sprintf("- 语言: %s\n", input.Language))
	}
	if input.Mood == "" && input.Genre == "" && input.Artist == "" && input.Language == "" {
		sb.WriteString("（无特殊偏好，请自由推荐一首经典好歌）\n")
	}

	return []*schema.Message{
		schema.SystemMessage(systemPrompt),
		schema.UserMessage(sb.String()),
	}
}
