package websocket

import "encoding/json"

const (
	EventMessageSend    = "MESSAGE_SEND"
	EventMessageNew     = "MESSAGE_NEW"
	EventMessageEdit    = "MESSAGE_EDIT"
	EventMessageUpdated = "MESSAGE_UPDATED"
	EventMessageDelete  = "MESSAGE_DELETE"
	EventMessageDeleted = "MESSAGE_DELETED"
	EventTypingStart    = "TYPING_START"
	EventTypingStop     = "TYPING_STOP"
	EventTyping         = "TYPING"
	EventReactionAdd    = "REACTION_ADD"
	EventReactionRemove = "REACTION_REMOVE"
	EventReaction       = "REACTION"
	EventPresenceUpdate = "PRESENCE_UPDATE"
	EventPresence       = "PRESENCE"
	EventDeliveryAck    = "DELIVERY_ACK"
	EventChannelJoin    = "CHANNEL_JOIN"
	EventChannelLeave   = "CHANNEL_LEAVE"
	EventMemberJoined   = "MEMBER_JOINED"
	EventMemberLeft     = "MEMBER_LEFT"
	EventReadAck        = "READ_ACK"
	EventReadState      = "READ_STATE"
	EventError          = "ERROR"
)

type WSEvent struct {
	Type      string          `json:"type"`
	RequestID string          `json:"request_id,omitempty"`
	Payload   json.RawMessage `json:"payload"`
}

type ErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
