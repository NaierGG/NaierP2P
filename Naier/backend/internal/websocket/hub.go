package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type Hub struct {
	clients    map[uuid.UUID]*Client
	channels   map[uuid.UUID]map[uuid.UUID]*Client
	userConns  map[uuid.UUID][]*Client
	register   chan *Client
	unregister chan *Client
	broadcast  chan *ChannelMessage
	redis      *redis.Client
	router     *Router
	delivery   DeliveryTracker
	instanceID string
	mu         sync.RWMutex
}

type DeliveryTracker interface {
	MarkDeliveredToDevice(ctx context.Context, deviceID, messageID uuid.UUID) error
}

type ChannelMessage struct {
	ChannelID  uuid.UUID
	Event      []byte
	ExcludeID  uuid.UUID
	RemoteOnly bool
}

type redisEnvelope struct {
	ChannelID  string          `json:"channel_id"`
	Event      json.RawMessage `json:"event"`
	ExcludeID  string          `json:"exclude_id,omitempty"`
	InstanceID string          `json:"instance_id"`
}

func NewHub(redisClient *redis.Client) *Hub {
	return &Hub{
		clients:    make(map[uuid.UUID]*Client),
		channels:   make(map[uuid.UUID]map[uuid.UUID]*Client),
		userConns:  make(map[uuid.UUID][]*Client),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan *ChannelMessage, 256),
		redis:      redisClient,
		instanceID: uuid.NewString(),
	}
}

func (h *Hub) SetRouter(router *Router) {
	h.router = router
}

func (h *Hub) SetDeliveryTracker(delivery DeliveryTracker) {
	h.delivery = delivery
}

func (h *Hub) Run(ctx context.Context) {
	go h.subscribeRedis(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case client := <-h.register:
			h.registerClient(client)
		case client := <-h.unregister:
			h.unregisterClient(client)
		case message := <-h.broadcast:
			h.broadcastLocal(message.ChannelID, message.Event, message.ExcludeID)
			if !message.RemoteOnly {
				h.publishToRedis(ctx, message)
			}
		}
	}
}

func (h *Hub) BroadcastToChannel(channelID uuid.UUID, event []byte, excludeClientID uuid.UUID) {
	h.broadcast <- &ChannelMessage{
		ChannelID: channelID,
		Event:     event,
		ExcludeID: excludeClientID,
	}
}

func (h *Hub) BroadcastToUser(userID uuid.UUID, event []byte) {
	h.mu.RLock()
	connections := append([]*Client(nil), h.userConns[userID]...)
	h.mu.RUnlock()

	for _, client := range connections {
		select {
		case client.Send <- event:
			h.markDeliveredIfMessage(client.DeviceID, event)
		default:
			go func(cl *Client) {
				h.unregister <- cl
			}(client)
		}
	}
}

func (h *Hub) markDeliveredIfMessage(deviceID uuid.UUID, event []byte) {
	if h.delivery == nil {
		return
	}

	var wsEvent WSEvent
	if err := json.Unmarshal(event, &wsEvent); err != nil {
		return
	}
	if wsEvent.Type != EventMessageNew {
		return
	}

	var payload struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(wsEvent.Payload, &payload); err != nil {
		return
	}
	messageID, err := uuid.Parse(payload.ID)
	if err != nil {
		return
	}

	_ = h.delivery.MarkDeliveredToDevice(context.Background(), deviceID, messageID)
}

func (h *Hub) GetOnlineUsersInChannel(channelID uuid.UUID) []uuid.UUID {
	h.mu.RLock()
	defer h.mu.RUnlock()

	channelClients := h.channels[channelID]
	seen := make(map[uuid.UUID]struct{}, len(channelClients))
	users := make([]uuid.UUID, 0, len(channelClients))
	for _, client := range channelClients {
		if _, exists := seen[client.UserID]; exists {
			continue
		}
		seen[client.UserID] = struct{}{}
		users = append(users, client.UserID)
	}

	return users
}

func (h *Hub) JoinChannel(client *Client, channelID uuid.UUID) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, exists := h.channels[channelID]; !exists {
		h.channels[channelID] = make(map[uuid.UUID]*Client)
	}

	h.channels[channelID][client.ID] = client
	client.subscribe(channelID)
}

func (h *Hub) LeaveChannel(client *Client, channelID uuid.UUID) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if channelClients, exists := h.channels[channelID]; exists {
		delete(channelClients, client.ID)
		if len(channelClients) == 0 {
			delete(h.channels, channelID)
		}
	}

	client.unsubscribe(channelID)
}

func (h *Hub) registerClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.clients[client.ID] = client
	h.userConns[client.UserID] = append(h.userConns[client.UserID], client)
}

func (h *Hub) unregisterClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, exists := h.clients[client.ID]; !exists {
		return
	}

	delete(h.clients, client.ID)
	for channelID, clients := range h.channels {
		delete(clients, client.ID)
		if len(clients) == 0 {
			delete(h.channels, channelID)
		}
	}

	connections := h.userConns[client.UserID]
	filtered := connections[:0]
	for _, conn := range connections {
		if conn.ID != client.ID {
			filtered = append(filtered, conn)
		}
	}
	if len(filtered) == 0 {
		delete(h.userConns, client.UserID)
	} else {
		h.userConns[client.UserID] = filtered
	}

	close(client.Send)
}

func (h *Hub) broadcastLocal(channelID uuid.UUID, event []byte, excludeClientID uuid.UUID) {
	h.mu.RLock()
	channelClients := h.channels[channelID]
	clients := make([]*Client, 0, len(channelClients))
	for _, client := range channelClients {
		clients = append(clients, client)
	}
	h.mu.RUnlock()

	for _, client := range clients {
		if excludeClientID != uuid.Nil && client.ID == excludeClientID {
			continue
		}

		select {
		case client.Send <- event:
		default:
			go func(cl *Client) {
				h.unregister <- cl
			}(client)
		}
	}
}

func (h *Hub) publishToRedis(ctx context.Context, message *ChannelMessage) {
	redisEvent := redisEnvelope{
		ChannelID:  message.ChannelID.String(),
		Event:      json.RawMessage(message.Event),
		ExcludeID:  message.ExcludeID.String(),
		InstanceID: h.instanceID,
	}

	payload, err := json.Marshal(redisEvent)
	if err != nil {
		return
	}

	_ = h.redis.Publish(ctx, h.redisChannel(message.ChannelID), payload).Err()
}

func (h *Hub) subscribeRedis(ctx context.Context) {
	pubsub := h.redis.PSubscribe(ctx, "ws:channel:*")
	defer func() {
		_ = pubsub.Close()
	}()

	ch := pubsub.Channel()
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}

			var envelope redisEnvelope
			if err := json.Unmarshal([]byte(msg.Payload), &envelope); err != nil {
				continue
			}

			if envelope.InstanceID == h.instanceID {
				continue
			}

			channelID, err := uuid.Parse(envelope.ChannelID)
			if err != nil {
				continue
			}

			excludeID := uuid.Nil
			if envelope.ExcludeID != "" {
				parsed, err := uuid.Parse(envelope.ExcludeID)
				if err == nil {
					excludeID = parsed
				}
			}

			h.broadcast <- &ChannelMessage{
				ChannelID:  channelID,
				Event:      []byte(envelope.Event),
				ExcludeID:  excludeID,
				RemoteOnly: true,
			}
		}
	}
}

func (h *Hub) redisChannel(channelID uuid.UUID) string {
	return fmt.Sprintf("ws:channel:%s", channelID.String())
}
