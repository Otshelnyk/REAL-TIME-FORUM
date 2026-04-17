package ws

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/ndanbaev/forum/internal/app/wsrt"
	"github.com/ndanbaev/forum/internal/adapters/deps"
	"github.com/ndanbaev/forum/internal/infra/notify"
	"github.com/ndanbaev/forum/internal/infra/sqlite"
	"github.com/ndanbaev/forum/internal/middleware"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// IncomingMessage from client
type IncomingMessage struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

// PrivateMessagePayload for sending a private message
type PrivateMessagePayload struct {
	ToID    int    `json:"to_id"`
	Content string `json:"content"`
}

// TypingPayload for typing indicator events
type TypingPayload struct {
	ToID     int  `json:"to_id"`
	IsTyping bool `json:"is_typing"`
}

// HandleWebSocket upgrades HTTP to WebSocket and handles the connection
func HandleWebSocket(hub *Hub, w http.ResponseWriter, r *http.Request) {
	user := middleware.GetCurrentUser(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	d := deps.FromRequest(r)
	if d == nil || d.DB == nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("ws upgrade: %v", err)
		return
	}

	client := &Client{UserID: user.ID, Send: make(chan []byte, 256)}
	hub.register <- client
	defer func() {
		hub.unregister <- client
		conn.Close()
	}()

	go writePump(conn, client)
	readPump(conn, hub, d.DB, client)
}

func readPump(conn *websocket.Conn, hub *Hub, db *sql.DB, client *Client) {
	defer conn.Close()
	svc := &wsrt.Service{
		Messages: sqlite.NewMessageRepo(db),
		Users:    sqlite.NewUserRepo(db),
		Notify:   &notify.Service{Repo: sqlite.NewNotificationRepo(db), Realtime: NewPublisher(hub)},
		Realtime: NewPublisher(hub),
		Now:      time.Now,
	}
	for {
		var msg IncomingMessage
		if err := conn.ReadJSON(&msg); err != nil {
			break
		}
		switch msg.Type {
		case "private_message":
			var payload PrivateMessagePayload
			if err := json.Unmarshal(msg.Payload, &payload); err != nil {
				continue
			}
			if err := svc.SendPrivateMessage(client.UserID, payload.ToID, payload.Content); err != nil {
				log.Printf("ws save pm: %v", err)
			}
		case "typing":
			var payload TypingPayload
			if err := json.Unmarshal(msg.Payload, &payload); err != nil {
				continue
			}
			_ = svc.Typing(client.UserID, payload.ToID, payload.IsTyping)
		}
	}
}

func writePump(conn *websocket.Conn, client *Client) {
	defer conn.Close()
	for data := range client.Send {
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			return
		}
	}
}
