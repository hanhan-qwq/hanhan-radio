package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"

	"github.com/hanhan-qwq/hanhan-radio/internal/agent"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type wsMessage struct {
	Type string `json:"type"`
	Data string `json:"data"`
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("ws upgrade: %v", err)
		return
	}
	defer conn.Close()

	sess, _ := s.store.GetOrCreate("ws-" + randomID())
	host := agent.NewRadioHost(s.runner, sess)
	defer host.Close()

	events := host.Start()

	// forward agent events to ws
	done := make(chan struct{})
	go func() {
		defer close(done)
		for evt := range events {
			writeWS(conn, evt.Type, evt.Data)
		}
	}()

	// read ws messages, forward to radio host
	go func() {
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}
			var m wsMessage
			if json.Unmarshal(msg, &m) != nil {
				continue
			}
			switch m.Type {
			case "speak":
				if m.Data != "" {
					host.Send(m.Data)
				}
			case "pause":
				host.Pause()
			case "resume":
				host.Resume()
			}
		}
	}()

	<-done
}

func writeWS(conn *websocket.Conn, typ, data string) {
	b, _ := json.Marshal(wsMessage{Type: typ, Data: data})
	conn.WriteMessage(websocket.TextMessage, b)
}

func randomID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano()%1000000)
}
