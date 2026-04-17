package ws

import (
	"encoding/json"
	"log"
	"sync"
)

// Client is a WebSocket client identified by user ID
type Client struct {
	UserID int
	Send   chan []byte
}

// Hub holds registered clients and broadcasts messages
type Hub struct {
	clients    map[int]map[*Client]struct{} // user ID -> active connections
	register   chan *Client
	unregister chan *Client
	broadcast  chan *BroadcastMessage
	mu         sync.RWMutex
}

// BroadcastMessage is sent to a specific user (for private messages)
type BroadcastMessage struct {
	TargetUserID int
	Payload      []byte
}

// WSMessage is the JSON structure for WebSocket messages
type WSMessage struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload,omitempty"`
}

// NewHub creates a new Hub
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[int]map[*Client]struct{}),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan *BroadcastMessage, 256),
	}
}

// Run runs the hub loop
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			var becameOnline bool
			h.mu.Lock()
			if h.clients[client.UserID] == nil {
				h.clients[client.UserID] = make(map[*Client]struct{})
			}
			if len(h.clients[client.UserID]) == 0 {
				becameOnline = true
			}
			h.clients[client.UserID][client] = struct{}{}
			h.mu.Unlock()
			if becameOnline {
				h.broadcastPresence(client.UserID, true)
			}

		case client := <-h.unregister:
			if client == nil {
				continue
			}
			var becameOffline bool
			h.mu.Lock()
			if byUser, ok := h.clients[client.UserID]; ok {
				if _, exists := byUser[client]; exists {
					delete(byUser, client)
					close(client.Send)
				}
				if len(byUser) == 0 {
					delete(h.clients, client.UserID)
					becameOffline = true
				}
			}
			h.mu.Unlock()
			if becameOffline {
				h.broadcastPresence(client.UserID, false)
			}

		case msg := <-h.broadcast:
			h.mu.RLock()
			if clients, ok := h.clients[msg.TargetUserID]; ok {
				for c := range clients {
					select {
					case c.Send <- msg.Payload:
					default:
						// client buffer full, skip
					}
				}
			}
			h.mu.RUnlock()
		}
	}
}

// BroadcastToUser sends a message to a specific user
func (h *Hub) BroadcastToUser(userID int, msg *WSMessage) {
	payload, err := json.Marshal(msg)
	if err != nil {
		log.Printf("ws hub: marshal error: %v", err)
		return
	}
	h.broadcast <- &BroadcastMessage{TargetUserID: userID, Payload: payload}
}

// OnlineUserIDs returns IDs of currently online users
func (h *Hub) OnlineUserIDs() []int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	ids := make([]int, 0, len(h.clients))
	for id, conns := range h.clients {
		if len(conns) > 0 {
			ids = append(ids, id)
		}
	}
	return ids
}

// IsOnline returns whether the user is connected via WebSocket
func (h *Hub) IsOnline(userID int) bool {
	h.mu.RLock()
	conns, ok := h.clients[userID]
	h.mu.RUnlock()
	return ok && len(conns) > 0
}

func (h *Hub) broadcastPresence(userID int, online bool) {
	msg := &WSMessage{
		Type: "user_presence",
		Payload: map[string]interface{}{
			"user_id": userID,
			"online":  online,
		},
	}
	payload, err := json.Marshal(msg)
	if err != nil {
		log.Printf("ws hub: presence marshal error: %v", err)
		return
	}
	h.mu.RLock()
	for _, byUser := range h.clients {
		for c := range byUser {
			select {
			case c.Send <- payload:
			default:
				// skip overloaded client
			}
		}
	}
	h.mu.RUnlock()
}
