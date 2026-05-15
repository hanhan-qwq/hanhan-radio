package radio

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"

	"github.com/hanhan-qwq/hanhan-radio/agentruntime/host"
	"github.com/hanhan-qwq/hanhan-radio/agentruntime/player"
	"github.com/hanhan-qwq/hanhan-radio/agentruntime/playlist"
	"github.com/hanhan-qwq/hanhan-radio/agentruntime/selector"
)

type Config struct {
	ChatModel  model.ToolCallingChatModel
	TracksJSON string // path to tracks.json (pre-tagged)
	PromptsDir string
	Interval   time.Duration
	Player     player.Player
	Tools      []tool.BaseTool
}

type Radio struct {
	cm       model.ToolCallingChatModel
	tracks   []playlist.Track
	host     *host.Host
	player   player.Player
	interval time.Duration

	subs   []chan string
	reqCh  chan request
	ctx    context.Context
	cancel context.CancelFunc
}

type request struct {
	track playlist.Track
	state string
}

func New(cfg Config) (*Radio, error) {
	tracks, err := playlist.Load(cfg.TracksJSON)
	if err != nil {
		return nil, fmt.Errorf("load tracks: %w", err)
	}

	h, err := host.New(cfg.ChatModel, cfg.PromptsDir, cfg.Tools)
	if err != nil {
		return nil, fmt.Errorf("host: %w", err)
	}

	if cfg.Player == nil {
		cfg.Player = &player.NopPlayer{}
	}

	ctx, cancel := context.WithCancel(context.Background())
	return &Radio{
		cm:       cfg.ChatModel,
		tracks:   tracks,
		host:     h,
		player:   cfg.Player,
		interval: cfg.Interval,
		reqCh:    make(chan request, 16),
		ctx:      ctx,
		cancel:   cancel,
	}, nil
}

func (r *Radio) Subscribe() <-chan string {
	ch := make(chan string, 64)
	r.subs = append(r.subs, ch)
	return ch
}

func (r *Radio) Request(track playlist.Track, state string) {
	select {
	case r.reqCh <- request{track: track, state: state}:
	default:
	}
}

func (r *Radio) Start() error {
	fmt.Println("🎙️  憨憨电台 自动播放中...")
	fmt.Println()

	var lastPlayed *host.LastPlayed
	listenerState := ""

	for {
		// check for request
		var sel *selector.Result
		select {
		case req := <-r.reqCh:
			sel = &selector.Result{Track: req.track, Reason: "听众点歌"}
			listenerState = req.state
		default:
		}

		// auto-select if no request
		if sel == nil {
			var lastTrack *playlist.Track
			if lastPlayed != nil {
				lastTrack = &lastPlayed.Track
			}
			var err error
			sel, err = selector.Next(r.ctx, r.cm, r.tracks, lastTrack, listenerState)
			if err != nil {
				return fmt.Errorf("selector: %w", err)
			}
		}

		// generate script
		var state string
		if listenerState != "" {
			state = listenerState
			listenerState = ""
		}

		info := BuildContext(state)
		stream, err := r.host.Generate(r.ctx, sel.Track, sel, lastPlayed, info.State, info.Time, info.Festival)
		if err != nil {
			return fmt.Errorf("host: %w", err)
		}

		var content strings.Builder
		for {
			chunk, err := stream.Recv()
			if err != nil {
				break
			}
			if chunk != nil && chunk.Content != "" {
				content.WriteString(chunk.Content)
				r.broadcast(chunk.Content)
			}
		}
		stream.Close()

		fmt.Println()
		r.broadcast("\n---\n")

		brief := host.Summary(content.String(), 80)
		lastPlayed = &host.LastPlayed{Track: sel.Track, Brief: brief}

		// play + wait
		r.player.Play(sel.Track)
		r.waitInterval()
	}
}

func (r *Radio) broadcast(text string) {
	for _, ch := range r.subs {
		select {
		case ch <- text:
		default:
		}
	}
}

func (r *Radio) waitInterval() {
	timer := time.NewTimer(r.interval)
	defer timer.Stop()
	select {
	case <-r.ctx.Done():
	case <-timer.C:
	}
}
