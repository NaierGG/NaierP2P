package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	validatorpkg "github.com/meshchat/backend/pkg/validator"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrChallengeExpired   = errors.New("challenge expired")
	ErrRefreshRevoked     = errors.New("refresh token revoked")
	ErrInvalidDevice      = errors.New("invalid device request")
)

type Service struct {
	db           *pgxpool.Pool
	redis        *redis.Client
	validate     *validatorpkg.Validator
	jwt          *JWTManager
	challengeTTL time.Duration
	refreshTTL   time.Duration
}

type userRecord struct {
	ID          uuid.UUID
	Username    string
	DisplayName string
	PublicKey   string
	AvatarURL   string
	Bio         string
	ServerID    string
	CreatedAt   time.Time
	IsActive    bool
}

type deviceRecord struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	DeviceKey  string
	DeviceName string
	Platform   string
	PushToken  string
	LastSeen   time.Time
	CreatedAt  time.Time
}

func NewService(db *pgxpool.Pool, redisClient *redis.Client, validate *validatorpkg.Validator, jwtManager *JWTManager, refreshTTL time.Duration) *Service {
	return &Service{
		db:           db,
		redis:        redisClient,
		validate:     validate,
		jwt:          jwtManager,
		challengeTTL: 5 * time.Minute,
		refreshTTL:   refreshTTL,
	}
}

func (s *Service) GetChallenge(ctx context.Context, username string) (string, error) {
	request := ChallengeRequest{Username: username}
	if err := s.validate.Struct(request); err != nil {
		return "", err
	}

	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("generate challenge: %w", err)
	}

	challenge := base64.RawURLEncoding.EncodeToString(randomBytes)
	if err := s.redis.Set(ctx, s.challengeKey(username), challenge, s.challengeTTL).Err(); err != nil {
		return "", fmt.Errorf("store challenge: %w", err)
	}

	return challenge, nil
}

func (s *Service) Register(ctx context.Context, req RegisterRequest) (AuthResponse, error) {
	if err := s.validate.Struct(req); err != nil {
		return AuthResponse{}, err
	}

	if _, err := X25519PublicKeyFromBase64(req.PublicKey); err != nil {
		return AuthResponse{}, err
	}

	challenge, err := s.loadChallenge(ctx, req.Username)
	if err != nil {
		return AuthResponse{}, err
	}

	if !VerifyEd25519Signature(req.PublicKey, challenge, req.Signature) {
		return AuthResponse{}, ErrInvalidCredentials
	}

	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return AuthResponse{}, fmt.Errorf("begin register transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var user userRecord
	err = tx.QueryRow(ctx, `
		INSERT INTO users (username, display_name, public_key)
		VALUES ($1, $2, $3)
		RETURNING id, username, display_name, public_key,
		          COALESCE(avatar_url, ''), COALESCE(bio, ''), server_id, created_at, is_active
	`, req.Username, req.DisplayName, req.PublicKey).Scan(
		&user.ID,
		&user.Username,
		&user.DisplayName,
		&user.PublicKey,
		&user.AvatarURL,
		&user.Bio,
		&user.ServerID,
		&user.CreatedAt,
		&user.IsActive,
	)
	if err != nil {
		return AuthResponse{}, fmt.Errorf("insert user: %w", err)
	}

	deviceID, err := s.createDevice(ctx, tx, user.ID, "web", "web")
	if err != nil {
		return AuthResponse{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return AuthResponse{}, fmt.Errorf("commit register transaction: %w", err)
	}

	_ = s.redis.Del(ctx, s.challengeKey(req.Username)).Err()

	return s.issueAuthResponse(user, deviceID)
}

func (s *Service) Login(ctx context.Context, req LoginRequest) (AuthResponse, error) {
	if err := s.validate.Struct(req); err != nil {
		return AuthResponse{}, err
	}

	challenge, err := s.loadChallenge(ctx, req.Username)
	if err != nil {
		return AuthResponse{}, err
	}

	if subtleCompare(challenge, req.Challenge) == false {
		return AuthResponse{}, ErrInvalidCredentials
	}

	user, err := s.getUserByUsername(ctx, req.Username)
	if err != nil {
		return AuthResponse{}, err
	}
	if !user.IsActive {
		return AuthResponse{}, ErrInvalidCredentials
	}

	if !VerifyEd25519Signature(user.PublicKey, challenge, req.Signature) {
		return AuthResponse{}, ErrInvalidCredentials
	}

	deviceID, err := s.createDevice(ctx, s.db, user.ID, "web", "web")
	if err != nil {
		return AuthResponse{}, err
	}

	_ = s.redis.Del(ctx, s.challengeKey(req.Username)).Err()

	return s.issueAuthResponse(user, deviceID)
}

func (s *Service) RefreshToken(ctx context.Context, refreshToken string) (AuthResponse, error) {
	claims, err := s.jwt.ValidateToken(refreshToken)
	if err != nil {
		return AuthResponse{}, ErrInvalidCredentials
	}

	if claims.TokenType != tokenTypeRefresh {
		return AuthResponse{}, ErrInvalidCredentials
	}

	deviceID, err := uuid.Parse(claims.DeviceID)
	if err != nil {
		return AuthResponse{}, ErrInvalidCredentials
	}

	revoked, err := s.redis.Exists(ctx, s.refreshBlacklistKey(deviceID)).Result()
	if err != nil {
		return AuthResponse{}, fmt.Errorf("check refresh blacklist: %w", err)
	}
	if revoked > 0 {
		return AuthResponse{}, ErrRefreshRevoked
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return AuthResponse{}, ErrInvalidCredentials
	}

	user, err := s.getUserByID(ctx, userID)
	if err != nil {
		return AuthResponse{}, err
	}
	if !user.IsActive {
		return AuthResponse{}, ErrInvalidCredentials
	}

	return s.issueAuthResponse(user, deviceID)
}

func (s *Service) Logout(ctx context.Context, deviceID uuid.UUID) error {
	if err := s.redis.Set(ctx, s.refreshBlacklistKey(deviceID), "revoked", s.refreshTTL).Err(); err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}

	_, err := s.db.Exec(ctx, `UPDATE devices SET last_seen = NOW() WHERE id = $1`, deviceID)
	if err != nil {
		return fmt.Errorf("update device logout time: %w", err)
	}

	return nil
}

func (s *Service) GetProfile(ctx context.Context, userID uuid.UUID) (UserDTO, error) {
	user, err := s.getUserByID(ctx, userID)
	if err != nil {
		return UserDTO{}, err
	}

	return toUserDTO(user), nil
}

func (s *Service) UpdateProfile(ctx context.Context, userID uuid.UUID, req UpdateProfileRequest) (UserDTO, error) {
	if err := s.validate.Struct(req); err != nil {
		return UserDTO{}, err
	}

	var user userRecord
	err := s.db.QueryRow(ctx, `
		UPDATE users
		SET display_name = $2,
		    avatar_url = NULLIF($3, ''),
		    bio = NULLIF($4, ''),
		    updated_at = NOW()
		WHERE id = $1
		RETURNING id, username, display_name, public_key,
		          COALESCE(avatar_url, ''), COALESCE(bio, ''), server_id, created_at, is_active
	`, userID, req.DisplayName, req.AvatarURL, req.Bio).Scan(
		&user.ID,
		&user.Username,
		&user.DisplayName,
		&user.PublicKey,
		&user.AvatarURL,
		&user.Bio,
		&user.ServerID,
		&user.CreatedAt,
		&user.IsActive,
	)
	if err != nil {
		return UserDTO{}, fmt.Errorf("update profile: %w", err)
	}

	return toUserDTO(user), nil
}

func (s *Service) ListDevices(ctx context.Context, userID, currentDeviceID uuid.UUID) ([]DeviceDTO, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, user_id, device_key, COALESCE(device_name, ''), platform,
		       COALESCE(push_token, ''), COALESCE(last_seen, created_at), created_at
		FROM devices
		WHERE user_id = $1
		ORDER BY CASE WHEN id = $2 THEN 0 ELSE 1 END, created_at DESC
	`, userID, currentDeviceID)
	if err != nil {
		return nil, fmt.Errorf("list devices: %w", err)
	}
	defer rows.Close()

	devices := make([]DeviceDTO, 0)
	for rows.Next() {
		var record deviceRecord
		if err := rows.Scan(
			&record.ID,
			&record.UserID,
			&record.DeviceKey,
			&record.DeviceName,
			&record.Platform,
			&record.PushToken,
			&record.LastSeen,
			&record.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan device: %w", err)
		}

		devices = append(devices, DeviceDTO{
			ID:         record.ID.String(),
			UserID:     record.UserID.String(),
			DeviceKey:  record.DeviceKey,
			DeviceName: record.DeviceName,
			Platform:   record.Platform,
			PushToken:  record.PushToken,
			LastSeen:   record.LastSeen,
			CreatedAt:  record.CreatedAt,
			Current:    record.ID == currentDeviceID,
		})
	}

	return devices, rows.Err()
}

func (s *Service) RevokeDevice(ctx context.Context, userID, currentDeviceID, targetDeviceID uuid.UUID) error {
	if targetDeviceID == currentDeviceID {
		return ErrInvalidDevice
	}

	commandTag, err := s.db.Exec(ctx, `
		DELETE FROM devices
		WHERE id = $1 AND user_id = $2
	`, targetDeviceID, userID)
	if err != nil {
		return fmt.Errorf("revoke device: %w", err)
	}
	if commandTag.RowsAffected() == 0 {
		return ErrInvalidCredentials
	}

	if err := s.redis.Set(ctx, s.refreshBlacklistKey(targetDeviceID), "revoked", s.refreshTTL).Err(); err != nil {
		return fmt.Errorf("blacklist revoked device token: %w", err)
	}

	return nil
}

func (s *Service) issueAuthResponse(user userRecord, deviceID uuid.UUID) (AuthResponse, error) {
	accessToken, refreshToken, err := s.jwt.GenerateTokenPair(user.ID, deviceID)
	if err != nil {
		return AuthResponse{}, fmt.Errorf("generate token pair: %w", err)
	}

	return AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         toUserDTO(user),
	}, nil
}

func (s *Service) getUserByUsername(ctx context.Context, username string) (userRecord, error) {
	var user userRecord
	err := s.db.QueryRow(ctx, `
		SELECT id, username, display_name, public_key,
		       COALESCE(avatar_url, ''), COALESCE(bio, ''), server_id, created_at, is_active
		FROM users
		WHERE username = $1
	`, username).Scan(
		&user.ID,
		&user.Username,
		&user.DisplayName,
		&user.PublicKey,
		&user.AvatarURL,
		&user.Bio,
		&user.ServerID,
		&user.CreatedAt,
		&user.IsActive,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return userRecord{}, ErrInvalidCredentials
	}
	if err != nil {
		return userRecord{}, fmt.Errorf("get user by username: %w", err)
	}

	return user, nil
}

func (s *Service) getUserByID(ctx context.Context, userID uuid.UUID) (userRecord, error) {
	var user userRecord
	err := s.db.QueryRow(ctx, `
		SELECT id, username, display_name, public_key,
		       COALESCE(avatar_url, ''), COALESCE(bio, ''), server_id, created_at, is_active
		FROM users
		WHERE id = $1
	`, userID).Scan(
		&user.ID,
		&user.Username,
		&user.DisplayName,
		&user.PublicKey,
		&user.AvatarURL,
		&user.Bio,
		&user.ServerID,
		&user.CreatedAt,
		&user.IsActive,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return userRecord{}, ErrInvalidCredentials
	}
	if err != nil {
		return userRecord{}, fmt.Errorf("get user by id: %w", err)
	}

	return user, nil
}

type deviceCreator interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func (s *Service) createDevice(ctx context.Context, executor deviceCreator, userID uuid.UUID, platform, deviceName string) (uuid.UUID, error) {
	var deviceID uuid.UUID
	err := executor.QueryRow(ctx, `
		INSERT INTO devices (user_id, device_key, device_name, platform, last_seen)
		VALUES ($1, $2, $3, $4, NOW())
		RETURNING id
	`, userID, uuid.NewString(), deviceName, platform).Scan(&deviceID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("create device: %w", err)
	}

	return deviceID, nil
}

func (s *Service) loadChallenge(ctx context.Context, username string) (string, error) {
	challenge, err := s.redis.Get(ctx, s.challengeKey(username)).Result()
	if errors.Is(err, redis.Nil) {
		return "", ErrChallengeExpired
	}
	if err != nil {
		return "", fmt.Errorf("load challenge: %w", err)
	}

	return challenge, nil
}

func (s *Service) challengeKey(username string) string {
	return "auth:challenge:" + strings.ToLower(username)
}

func (s *Service) refreshBlacklistKey(deviceID uuid.UUID) string {
	return "auth:refresh:blacklist:" + deviceID.String()
}

func toUserDTO(user userRecord) UserDTO {
	return UserDTO{
		ID:          user.ID.String(),
		Username:    user.Username,
		DisplayName: user.DisplayName,
		PublicKey:   user.PublicKey,
		AvatarURL:   user.AvatarURL,
		Bio:         user.Bio,
		ServerID:    user.ServerID,
		CreatedAt:   user.CreatedAt,
	}
}

func subtleCompare(left, right string) bool {
	if len(left) != len(right) {
		return false
	}

	var diff byte
	for i := range left {
		diff |= left[i] ^ right[i]
	}

	return diff == 0
}
