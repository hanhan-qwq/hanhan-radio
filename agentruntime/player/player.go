package player

import "github.com/hanhan-qwq/hanhan-radio/agentruntime/playlist"

type Player interface {
	Play(track playlist.Track) error
	Stop() error
}

type NopPlayer struct{}

func (n *NopPlayer) Play(track playlist.Track) error { return nil }
func (n *NopPlayer) Stop() error                      { return nil }
