package message

import "time"

type ReactionDTO struct {
	UserID string `json:"user_id"`
	Emoji  string `json:"emoji"`
}

type ReactionEventDTO struct {
	MessageID string    `json:"message_id"`
	ChannelID string    `json:"channel_id"`
	UserID    string    `json:"user_id"`
	Emoji     string    `json:"emoji"`
	Action    string    `json:"action"`
	EventID   string    `json:"event_id"`
	Sequence  int64     `json:"sequence"`
	CreatedAt time.Time `json:"created_at"`
}

type ReadStateDTO struct {
	ChannelID        string    `json:"channel_id"`
	UserID           string    `json:"user_id"`
	LastReadSequence int64     `json:"last_read_sequence"`
	EventID          string    `json:"event_id"`
	Sequence         int64     `json:"sequence"`
	CreatedAt        time.Time `json:"created_at"`
}

type MessageDTO struct {
	ID            string        `json:"id"`
	ChannelID     string        `json:"channel_id"`
	SenderID      string        `json:"sender_id"`
	Type          string        `json:"type"`
	Content       string        `json:"content"`
	IV            string        `json:"iv,omitempty"`
	ReplyToID     string        `json:"reply_to_id,omitempty"`
	IsEdited      bool          `json:"is_edited"`
	IsDeleted     bool          `json:"is_deleted"`
	Signature     string        `json:"signature,omitempty"`
	ClientEventID string        `json:"client_event_id,omitempty"`
	ServerEventID string        `json:"server_event_id,omitempty"`
	Sequence      int64         `json:"sequence,omitempty"`
	CreatedAt     time.Time     `json:"created_at"`
	UpdatedAt     time.Time     `json:"updated_at"`
	Reactions     []ReactionDTO `json:"reactions,omitempty"`
}

type CreateMessageRequest struct {
	Type          string `json:"type" validate:"omitempty,oneof=text image file system"`
	Content       string `json:"content" validate:"required"`
	IV            string `json:"iv"`
	ReplyToID     string `json:"reply_to_id,omitempty"`
	Signature     string `json:"signature,omitempty"`
	ClientEventID string `json:"client_event_id,omitempty"`
}

type UpdateMessageRequest struct {
	Content string `json:"content" validate:"required"`
	IV      string `json:"iv"`
}

type ReactionRequest struct {
	Emoji string `json:"emoji" validate:"required,min=1,max=10"`
}

type ListResponse struct {
	Messages   []MessageDTO `json:"messages"`
	NextCursor string       `json:"next_cursor,omitempty"`
	HasMore    bool         `json:"has_more"`
}

type SyncEvent struct {
	Type      string            `json:"type"`
	Message   *MessageDTO       `json:"message,omitempty"`
	Reaction  *ReactionEventDTO `json:"reaction,omitempty"`
	ReadState *ReadStateDTO     `json:"read_state,omitempty"`
	EventID   string            `json:"event_id"`
	Sequence  int64             `json:"sequence"`
	ChannelID string            `json:"channel_id"`
}

type SyncResponse struct {
	Events      []SyncEvent `json:"events"`
	LastEventID string      `json:"last_event_id,omitempty"`
	HasMore     bool        `json:"has_more"`
}
