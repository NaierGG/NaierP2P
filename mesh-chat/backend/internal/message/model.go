package message

import "time"

type ReactionDTO struct {
	UserID string `json:"user_id"`
	Emoji  string `json:"emoji"`
}

type MessageDTO struct {
	ID        string        `json:"id"`
	ChannelID string        `json:"channel_id"`
	SenderID  string        `json:"sender_id"`
	Type      string        `json:"type"`
	Content   string        `json:"content"`
	IV        string        `json:"iv,omitempty"`
	ReplyToID string        `json:"reply_to_id,omitempty"`
	IsEdited  bool          `json:"is_edited"`
	IsDeleted bool          `json:"is_deleted"`
	Signature string        `json:"signature,omitempty"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
	Reactions []ReactionDTO `json:"reactions,omitempty"`
}

type CreateMessageRequest struct {
	Type      string `json:"type" validate:"omitempty,oneof=text image file system"`
	Content   string `json:"content" validate:"required"`
	IV        string `json:"iv"`
	ReplyToID string `json:"reply_to_id,omitempty"`
	Signature string `json:"signature,omitempty"`
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
