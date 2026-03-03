package channel

import "time"

type ChannelLastMessage struct {
	ID        string    `json:"id"`
	Content   string    `json:"content"`
	Type      string    `json:"type"`
	SenderID  string    `json:"sender_id"`
	CreatedAt time.Time `json:"created_at"`
}

type ChannelDTO struct {
	ID          string              `json:"id"`
	Type        string              `json:"type"`
	Name        string              `json:"name"`
	Description string              `json:"description"`
	InviteCode  string              `json:"invite_code,omitempty"`
	OwnerID     string              `json:"owner_id,omitempty"`
	IsEncrypted bool                `json:"is_encrypted"`
	MaxMembers  int                 `json:"max_members"`
	MemberCount int                 `json:"member_count"`
	CreatedAt   time.Time           `json:"created_at"`
	LastMessage *ChannelLastMessage `json:"last_message,omitempty"`
}

type ChannelMemberDTO struct {
	UserID      string    `json:"user_id"`
	Username    string    `json:"username"`
	DisplayName string    `json:"display_name"`
	Role        string    `json:"role"`
	JoinedAt    time.Time `json:"joined_at"`
	IsMuted     bool      `json:"is_muted"`
}

type CreateChannelRequest struct {
	Type        string `json:"type" validate:"required,oneof=dm group public"`
	Name        string `json:"name" validate:"max=100"`
	Description string `json:"description" validate:"max=500"`
	IsEncrypted *bool  `json:"is_encrypted,omitempty"`
	MaxMembers  *int   `json:"max_members,omitempty" validate:"omitempty,min=1,max=5000"`
}

type UpdateChannelRequest struct {
	Name        string `json:"name" validate:"max=100"`
	Description string `json:"description" validate:"max=500"`
	IsEncrypted *bool  `json:"is_encrypted,omitempty"`
	MaxMembers  *int   `json:"max_members,omitempty" validate:"omitempty,min=1,max=5000"`
}

type InviteRequest struct {
	InviteCode string `json:"invite_code" validate:"required,min=4,max=20"`
}
