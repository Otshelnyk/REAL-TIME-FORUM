package ws

import "github.com/ndanbaev/forum/internal/app/ports"

// Publisher adapts Hub to the application RealtimePublisher port.
// It preserves existing WS message shapes.
type Publisher struct {
	Hub *Hub
}

func NewPublisher(hub *Hub) *Publisher {
	return &Publisher{Hub: hub}
}

var _ ports.RealtimePublisher = (*Publisher)(nil)

func (p *Publisher) BroadcastToUser(userID int, msgType string, payload any) {
	if p == nil || p.Hub == nil {
		return
	}
	p.Hub.BroadcastToUser(userID, &WSMessage{Type: msgType, Payload: payload})
}

func (p *Publisher) OnlineUserIDs() []int {
	if p == nil || p.Hub == nil {
		return nil
	}
	return p.Hub.OnlineUserIDs()
}

