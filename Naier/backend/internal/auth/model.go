package auth

import "time"

type RegisterRequest struct {
	Username                    string `json:"username" validate:"required,min=3,max=50,alphanum"`
	DisplayName                 string `json:"display_name" validate:"required,min=1,max=100"`
	PublicKey                   string `json:"public_key,omitempty"`
	Signature                   string `json:"signature,omitempty"`
	IdentitySigningKey          string `json:"identity_signing_key,omitempty"`
	IdentityExchangeKey         string `json:"identity_exchange_key,omitempty"`
	DeviceSigningKey            string `json:"device_signing_key,omitempty"`
	DeviceExchangeKey           string `json:"device_exchange_key,omitempty"`
	DeviceSignature             string `json:"device_signature,omitempty"`
	IdentitySignatureOverDevice string `json:"identity_signature_over_device,omitempty"`
	DeviceName                  string `json:"device_name,omitempty"`
	Platform                    string `json:"platform,omitempty"`
}

type LoginRequest struct {
	Username         string `json:"username" validate:"required"`
	Challenge        string `json:"challenge" validate:"required"`
	Signature        string `json:"signature,omitempty"`
	DeviceSignature  string `json:"device_signature,omitempty"`
	DeviceID         string `json:"device_id,omitempty"`
	DeviceSigningKey string `json:"device_signing_key,omitempty"`
	DeviceName       string `json:"device_name,omitempty"`
	Platform         string `json:"platform,omitempty"`
}

type ChallengeRequest struct {
	Username         string `json:"username" validate:"required,min=3,max=50,alphanum"`
	DeviceID         string `json:"device_id,omitempty"`
	DeviceName       string `json:"device_name,omitempty"`
	Platform         string `json:"platform,omitempty"`
	DeviceSigningKey string `json:"device_signing_key,omitempty"`
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
	ID                  string    `json:"id"`
	Username            string    `json:"username"`
	DisplayName         string    `json:"display_name"`
	PublicKey           string    `json:"public_key,omitempty"`
	IdentitySigningKey  string    `json:"identity_signing_key"`
	IdentityExchangeKey string    `json:"identity_exchange_key"`
	AvatarURL           string    `json:"avatar_url,omitempty"`
	Bio                 string    `json:"bio,omitempty"`
	ServerID            string    `json:"server_id"`
	CreatedAt           time.Time `json:"created_at"`
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
	ID                 string     `json:"id"`
	UserID             string     `json:"user_id"`
	DeviceKey          string     `json:"device_key,omitempty"`
	DeviceSigningKey   string     `json:"device_signing_key"`
	DeviceExchangeKey  string     `json:"device_exchange_key"`
	DeviceName         string     `json:"device_name,omitempty"`
	Platform           string     `json:"platform"`
	PushToken          string     `json:"push_token,omitempty"`
	LastSeen           time.Time  `json:"last_seen,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	Current            bool       `json:"current"`
	Trusted            bool       `json:"trusted"`
	ApprovedByDeviceID string     `json:"approved_by_device_id,omitempty"`
	RevokedAt          *time.Time `json:"revoked_at,omitempty"`
}

type ApproveDeviceRequest struct {
	DeviceID string `json:"device_id" validate:"required,uuid"`
}

type RegisterPendingDeviceRequest struct {
	DeviceSigningKey  string `json:"device_signing_key" validate:"required"`
	DeviceExchangeKey string `json:"device_exchange_key" validate:"required"`
	DeviceName        string `json:"device_name,omitempty" validate:"omitempty,max=100"`
	Platform          string `json:"platform,omitempty" validate:"omitempty,oneof=web ios android"`
}

type BackupExportRequest struct {
	BackupBlob    string `json:"backup_blob" validate:"required"`
	BackupVersion int    `json:"backup_version,omitempty" validate:"omitempty,min=1,max=10"`
}

type BackupExportResponse struct {
	BackupVersion int       `json:"backup_version"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type BackupImportResponse struct {
	BackupBlob    string    `json:"backup_blob"`
	BackupVersion int       `json:"backup_version"`
	UpdatedAt     time.Time `json:"updated_at"`
}
