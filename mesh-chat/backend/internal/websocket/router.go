package websocket

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	gorillaws "github.com/gorilla/websocket"
	"github.com/google/uuid"
	"github.com/meshchat/backend/internal/auth"
)

type MessageService interface {
	Create(ctx context.Context, userID, channelID uuid.UUID, content, iv string, replyToID *uuid.UUID) ([]byte, error)
	Edit(ctx context.Context, userID, messageID uuid.UUID, content, iv string) (channelID uuid.UUID, event []byte, err error)
	Delete(ctx context.Context, userID, messageID uuid.UUID) (channelID uuid.UUID, event []byte, err error)
	AddReaction(ctx context.Context, userID, messageID uuid.UUID, emoji string) (channelID uuid.UUID, event []byte, err error)
	RemoveReaction(ctx context.Context, userID, messageID uuid.UUID, emoji string) (channelID uuid.UUID, event []byte, err error)
	MarkRead(ctx context.Context, userID, channelID, messageID uuid.UUID) error
}

type PresenceService interface {
	SetTyping(ctx context.Context, channelID, userID uuid.UUID, isTyping bool) error
	UpdatePresence(ctx context.Context, userID uuid.UUID, status string) error
}

type Router struct {
	hub             *Hub
	jwtManager      *auth.JWTManager
	messageService  MessageService
	presenceService PresenceService
	upgrader        gorillaws.Upgrader
}

type channelSubscriptionPayload struct {
	ChannelID string `json:"channelId"`
}

type messageSendPayload struct {
	ChannelID string `json:"channelId"`
	Content   string `json:"content"`
	IV        string `json:"iv"`
	ReplyToID string `json:"replyToId,omitempty"`
}

type messageEditPayload struct {
	MessageID string `json:"messageId"`
	Content   string `json:"content"`
	IV        string `json:"iv"`
}

type messageDeletePayload struct {
	MessageID string `json:"messageId"`
}

type reactionPayload struct {
	MessageID string `json:"messageId"`
	Emoji     string `json:"emoji"`
}

type presencePayload struct {
	Status string `json:"status"`
}

type readAckPayload struct {
	ChannelID string `json:"channelId"`
	MessageID string `json:"messageId"`
}

func NewRouter(hub *Hub, jwtManager *auth.JWTManager, messageService MessageService, presenceService PresenceService) *Router {
	return &Router{
		hub:             hub,
		jwtManager:      jwtManager,
		messageService:  messageService,
		presenceService: presenceService,
		upgrader: gorillaws.Upgrader{
			CheckOrigin: func(*http.Request) bool { return true },
		},
	}
}

func (r *Router) ServeWS(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized", "message": "missing websocket token"})
		return
	}

	claims, err := r.jwtManager.ValidateToken(token)
	if err != nil || claims.TokenType != "access" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized", "message": "invalid websocket token"})
		return
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized", "message": "invalid user claim"})
		return
	}

	deviceID, err := uuid.Parse(claims.DeviceID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized", "message": "invalid device claim"})
		return
	}

	conn, err := r.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	client := &Client{
		ID:       uuid.New(),
		UserID:   userID,
		DeviceID: deviceID,
		Conn:     conn,
		Hub:      r.hub,
		Send:     make(chan []byte, 128),
		Channels: make(map[uuid.UUID]bool),
	}

	r.hub.register <- client

	go client.writePump()
	go client.readPump()
}

func (r *Router) HandleClientEvent(client *Client, event WSEvent) {
	switch event.Type {
	case EventChannelJoin:
		r.handleChannelJoin(client, event)
	case EventChannelLeave:
		r.handleChannelLeave(client, event)
	case EventMessageSend:
		r.handleMessageSend(client, event)
	case EventMessageEdit:
		r.handleMessageEdit(client, event)
	case EventMessageDelete:
		r.handleMessageDelete(client, event)
	case EventReactionAdd:
		r.handleReactionAdd(client, event)
	case EventReactionRemove:
		r.handleReactionRemove(client, event)
	case EventTypingStart:
		r.handleTyping(client, event, true)
	case EventTypingStop:
		r.handleTyping(client, event, false)
	case EventPresenceUpdate:
		r.handlePresenceUpdate(client, event)
	case EventReadAck:
		r.handleReadAck(client, event)
	default:
		r.sendError(client, event.RequestID, "unsupported_event", "unsupported websocket event type")
	}
}

func (r *Router) handleChannelJoin(client *Client, event WSEvent) {
	var payload channelSubscriptionPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		r.sendError(client, event.RequestID, "bad_payload", "invalid channel join payload")
		return
	}

	channelID, err := uuid.Parse(payload.ChannelID)
	if err != nil {
		r.sendError(client, event.RequestID, "bad_channel_id", "invalid channel id")
		return
	}

	r.hub.JoinChannel(client, channelID)
}

func (r *Router) handleChannelLeave(client *Client, event WSEvent) {
	var payload channelSubscriptionPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		r.sendError(client, event.RequestID, "bad_payload", "invalid channel leave payload")
		return
	}

	channelID, err := uuid.Parse(payload.ChannelID)
	if err != nil {
		r.sendError(client, event.RequestID, "bad_channel_id", "invalid channel id")
		return
	}

	r.hub.LeaveChannel(client, channelID)
}

func (r *Router) handleMessageSend(client *Client, event WSEvent) {
	if r.messageService == nil {
		r.sendError(client, event.RequestID, "service_unavailable", "message service unavailable")
		return
	}

	var payload messageSendPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		r.sendError(client, event.RequestID, "bad_payload", "invalid message payload")
		return
	}

	channelID, err := uuid.Parse(payload.ChannelID)
	if err != nil {
		r.sendError(client, event.RequestID, "bad_channel_id", "invalid channel id")
		return
	}

	var replyToID *uuid.UUID
	if payload.ReplyToID != "" {
		parsed, err := uuid.Parse(payload.ReplyToID)
		if err != nil {
			r.sendError(client, event.RequestID, "bad_reply_to_id", "invalid reply message id")
			return
		}
		replyToID = &parsed
	}

	responseEvent, err := r.messageService.Create(context.Background(), client.UserID, channelID, payload.Content, payload.IV, replyToID)
	if err != nil {
		r.sendError(client, event.RequestID, "message_send_failed", err.Error())
		return
	}

	r.hub.BroadcastToChannel(channelID, responseEvent, uuid.Nil)
}

func (r *Router) handleMessageEdit(client *Client, event WSEvent) {
	if r.messageService == nil {
		r.sendError(client, event.RequestID, "service_unavailable", "message service unavailable")
		return
	}

	var payload messageEditPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		r.sendError(client, event.RequestID, "bad_payload", "invalid edit payload")
		return
	}

	messageID, err := uuid.Parse(payload.MessageID)
	if err != nil {
		r.sendError(client, event.RequestID, "bad_message_id", "invalid message id")
		return
	}

	channelID, responseEvent, err := r.messageService.Edit(context.Background(), client.UserID, messageID, payload.Content, payload.IV)
	if err != nil {
		r.sendError(client, event.RequestID, "message_edit_failed", err.Error())
		return
	}

	r.hub.BroadcastToChannel(channelID, responseEvent, uuid.Nil)
}

func (r *Router) handleMessageDelete(client *Client, event WSEvent) {
	if r.messageService == nil {
		r.sendError(client, event.RequestID, "service_unavailable", "message service unavailable")
		return
	}

	var payload messageDeletePayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		r.sendError(client, event.RequestID, "bad_payload", "invalid delete payload")
		return
	}

	messageID, err := uuid.Parse(payload.MessageID)
	if err != nil {
		r.sendError(client, event.RequestID, "bad_message_id", "invalid message id")
		return
	}

	channelID, responseEvent, err := r.messageService.Delete(context.Background(), client.UserID, messageID)
	if err != nil {
		r.sendError(client, event.RequestID, "message_delete_failed", err.Error())
		return
	}

	r.hub.BroadcastToChannel(channelID, responseEvent, uuid.Nil)
}

func (r *Router) handleReactionAdd(client *Client, event WSEvent) {
	r.handleReaction(client, event, true)
}

func (r *Router) handleReactionRemove(client *Client, event WSEvent) {
	r.handleReaction(client, event, false)
}

func (r *Router) handleReaction(client *Client, event WSEvent, add bool) {
	if r.messageService == nil {
		r.sendError(client, event.RequestID, "service_unavailable", "message service unavailable")
		return
	}

	var payload reactionPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		r.sendError(client, event.RequestID, "bad_payload", "invalid reaction payload")
		return
	}

	messageID, err := uuid.Parse(payload.MessageID)
	if err != nil {
		r.sendError(client, event.RequestID, "bad_message_id", "invalid message id")
		return
	}

	var (
		channelID     uuid.UUID
		responseEvent []byte
	)
	if add {
		channelID, responseEvent, err = r.messageService.AddReaction(context.Background(), client.UserID, messageID, payload.Emoji)
	} else {
		channelID, responseEvent, err = r.messageService.RemoveReaction(context.Background(), client.UserID, messageID, payload.Emoji)
	}
	if err != nil {
		r.sendError(client, event.RequestID, "reaction_failed", err.Error())
		return
	}

	r.hub.BroadcastToChannel(channelID, responseEvent, uuid.Nil)
}

func (r *Router) handleTyping(client *Client, event WSEvent, isTyping bool) {
	if r.presenceService == nil {
		r.sendError(client, event.RequestID, "service_unavailable", "presence service unavailable")
		return
	}

	var payload channelSubscriptionPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		r.sendError(client, event.RequestID, "bad_payload", "invalid typing payload")
		return
	}

	channelID, err := uuid.Parse(payload.ChannelID)
	if err != nil {
		r.sendError(client, event.RequestID, "bad_channel_id", "invalid channel id")
		return
	}

	if err := r.presenceService.SetTyping(context.Background(), channelID, client.UserID, isTyping); err != nil {
		r.sendError(client, event.RequestID, "typing_failed", err.Error())
		return
	}

	responseEvent, err := marshalEvent(WSEvent{
		Type:      EventTyping,
		RequestID: event.RequestID,
		Payload: mustMarshalRaw(map[string]any{
			"userId":    client.UserID.String(),
			"channelId": channelID.String(),
			"isTyping":  isTyping,
		}),
	})
	if err != nil {
		r.sendError(client, event.RequestID, "typing_failed", "failed to encode typing event")
		return
	}

	r.hub.BroadcastToChannel(channelID, responseEvent, client.ID)
}

func (r *Router) handlePresenceUpdate(client *Client, event WSEvent) {
	if r.presenceService == nil {
		r.sendError(client, event.RequestID, "service_unavailable", "presence service unavailable")
		return
	}

	var payload presencePayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		r.sendError(client, event.RequestID, "bad_payload", "invalid presence payload")
		return
	}

	if err := r.presenceService.UpdatePresence(context.Background(), client.UserID, payload.Status); err != nil {
		r.sendError(client, event.RequestID, "presence_failed", err.Error())
		return
	}

	responseEvent, err := marshalEvent(WSEvent{
		Type:      EventPresence,
		RequestID: event.RequestID,
		Payload: mustMarshalRaw(map[string]any{
			"userId": client.UserID.String(),
			"status": payload.Status,
		}),
	})
	if err != nil {
		r.sendError(client, event.RequestID, "presence_failed", "failed to encode presence event")
		return
	}

	r.hub.BroadcastToUser(client.UserID, responseEvent)
}

func (r *Router) handleReadAck(client *Client, event WSEvent) {
	if r.messageService == nil {
		r.sendError(client, event.RequestID, "service_unavailable", "message service unavailable")
		return
	}

	var payload readAckPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		r.sendError(client, event.RequestID, "bad_payload", "invalid read ack payload")
		return
	}

	channelID, err := uuid.Parse(payload.ChannelID)
	if err != nil {
		r.sendError(client, event.RequestID, "bad_channel_id", "invalid channel id")
		return
	}

	messageID, err := uuid.Parse(payload.MessageID)
	if err != nil {
		r.sendError(client, event.RequestID, "bad_message_id", "invalid message id")
		return
	}

	if err := r.messageService.MarkRead(context.Background(), client.UserID, channelID, messageID); err != nil {
		r.sendError(client, event.RequestID, "read_ack_failed", err.Error())
	}
}

func (r *Router) sendError(client *Client, requestID, code, message string) {
	encoded, err := marshalEvent(WSEvent{
		Type:      EventError,
		RequestID: requestID,
		Payload: mustMarshalRaw(ErrorPayload{
			Code:    code,
			Message: message,
		}),
	})
	if err != nil {
		return
	}

	select {
	case client.Send <- encoded:
	default:
		go func() {
			r.hub.unregister <- client
		}()
	}
}

func marshalEvent(event WSEvent) ([]byte, error) {
	encoded, err := json.Marshal(event)
	if err != nil {
		return nil, err
	}
	return encoded, nil
}

func mustMarshalRaw(value any) json.RawMessage {
	encoded, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return json.RawMessage(encoded)
}

func IsUnauthorized(err error) bool {
	return errors.Is(err, auth.ErrInvalidCredentials)
}
