package federation

import (
	"encoding/json"
	"time"

	"github.com/naier/backend/internal/auth"
	"github.com/naier/backend/internal/message"
)

type FederatedEvent struct {
	EventID   string          `json:"event_id"`
	Type      string          `json:"type"`
	ServerID  string          `json:"server_id"`
	Timestamp time.Time       `json:"timestamp"`
	Payload   json.RawMessage `json:"payload"`
	Signature string          `json:"signature"`
}

type ResolvedServer struct {
	Domain    string
	PublicKey string
	Endpoint  string
}

type ServerKeyResponse struct {
	Domain    string `json:"domain"`
	PublicKey string `json:"public_key"`
}

type WellKnownResponse struct {
	Version   string `json:"version"`
	Domain    string `json:"domain"`
	PublicKey string `json:"public_key"`
	Endpoint  string `json:"endpoint"`
}

type EventAckResponse struct {
	Status    string `json:"status"`
	EventID   string `json:"event_id"`
	ServerID  string `json:"server_id"`
	Processed bool   `json:"processed"`
	Duplicate bool   `json:"duplicate,omitempty"`
}

type EventEnvelope struct {
	Event FederatedEvent `json:"event"`
}

type RemoteUserResponse struct {
	User auth.UserDTO `json:"user"`
}

type MessageForwardPayload struct {
	Message message.MessageDTO `json:"message"`
}

type EventProcessingResult struct {
	Processed bool
	Duplicate bool
}

type FederatedChannelMember struct {
	User     auth.UserDTO `json:"user"`
	Role     string       `json:"role"`
	JoinedAt time.Time    `json:"joined_at"`
	IsMuted  bool         `json:"is_muted"`
}

type FederatedChannelStatePayload struct {
	ChannelID   string                   `json:"channel_id"`
	ChannelType string                   `json:"channel_type"`
	Name        string                   `json:"name"`
	Description string                   `json:"description"`
	IsEncrypted bool                     `json:"is_encrypted"`
	MaxMembers  int                      `json:"max_members"`
	MemberCount int                      `json:"member_count"`
	Members     []FederatedChannelMember `json:"members"`
}

type ChannelStateResponse struct {
	Channel FederatedChannelStatePayload `json:"channel"`
}

type ChannelStateSyncRequest struct {
	TargetDomain string `json:"target_domain"`
}
