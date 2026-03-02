package auth

import "time"

type RegisterRequest struct {
	Username    string `json:"username" validate:"required,min=3,max=50,alphanum"`
	DisplayName string `json:"display_name" validate:"required,min=1,max=100"`
	PublicKey   string `json:"public_key" validate:"required"`
	Signature   string `json:"signature" validate:"required"`
}

type LoginRequest struct {
	Username  string `json:"username" validate:"required"`
	Challenge string `json:"challenge" validate:"required"`
	Signature string `json:"signature" validate:"required"`
}

type ChallengeRequest struct {
	Username string `json:"username" validate:"required,min=3,max=50,alphanum"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type UpdateProfileRequest struct {
	DisplayName string `json:"display_name" validate:"required,min=1,max=100"`
	AvatarURL   string `json:"avatar_url,omitempty" validate:"omitempty,url"`
	Bio         string `json:"bio,omitempty" validate:"max=500"`
}

type UserDTO struct {
	ID          string    `json:"id"`
	Username    string    `json:"username"`
	DisplayName string    `json:"display_name"`
	PublicKey   string    `json:"public_key"`
	AvatarURL   string    `json:"avatar_url,omitempty"`
	Bio         string    `json:"bio,omitempty"`
	ServerID    string    `json:"server_id"`
	CreatedAt   time.Time `json:"created_at"`
}

type AuthResponse struct {
	AccessToken  string  `json:"access_token"`
	RefreshToken string  `json:"refresh_token"`
	User         UserDTO `json:"user"`
}

type ChallengeResponse struct {
	Challenge string `json:"challenge"`
	TTL       int    `json:"ttl"`
}

type DeviceDTO struct {
	ID         string    `json:"id"`
	UserID     string    `json:"user_id"`
	DeviceKey  string    `json:"device_key"`
	DeviceName string    `json:"device_name,omitempty"`
	Platform   string    `json:"platform"`
	PushToken  string    `json:"push_token,omitempty"`
	LastSeen   time.Time `json:"last_seen,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	Current    bool      `json:"current"`
}
