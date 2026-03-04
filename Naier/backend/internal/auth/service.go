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

	validatorpkg "github.com/naier/backend/pkg/validator"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrChallengeExpired   = errors.New("challenge expired")
	ErrRefreshRevoked     = errors.New("refresh token revoked")
	ErrInvalidDevice      = errors.New("invalid device request")
	ErrBackupNotFound     = errors.New("backup not found")
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
	ID                  uuid.UUID
	Username            string
	DisplayName         string
	PublicKey           string
	IdentitySigningKey  string
	IdentityExchangeKey string
	AvatarURL           string
	Bio                 string
	ServerID            string
	CreatedAt           time.Time
	IsActive            bool
}

type deviceRecord struct {
	ID                 uuid.UUID
	UserID             uuid.UUID
	DeviceKey          string
	DeviceSigningKey   string
	DeviceExchangeKey  string
	DeviceName         string
	Platform           string
	PushToken          string
	LastSeen           time.Time
	CreatedAt          time.Time
	Trusted            bool
	ApprovedByDeviceID *uuid.UUID
	RevokedAt          *time.Time
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

func (s *Service) GetChallenge(ctx context.Context, request ChallengeRequest) (string, error) {
	if err := s.validate.Struct(request); err != nil {
		return "", err
	}

	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("generate challenge: %w", err)
	}

	challenge := base64.RawURLEncoding.EncodeToString(randomBytes)
	if err := s.redis.Set(ctx, s.challengeKey(request), challenge, s.challengeTTL).Err(); err != nil {
		return "", fmt.Errorf("store challenge: %w", err)
	}

	return challenge, nil
}

func (s *Service) Register(ctx context.Context, req RegisterRequest) (AuthResponse, error) {
	if err := s.validate.Struct(req); err != nil {
		return AuthResponse{}, err
	}

	normalizedReq, err := s.normalizeRegisterRequest(req)
	if err != nil {
		return AuthResponse{}, err
	}

	if _, err := Ed25519PublicKeyFromBase64(normalizedReq.IdentitySigningKey); err != nil {
		return AuthResponse{}, err
	}
	if _, err := X25519PublicKeyFromBase64(normalizedReq.IdentityExchangeKey); err != nil {
		return AuthResponse{}, err
	}
	if _, err := Ed25519PublicKeyFromBase64(normalizedReq.DeviceSigningKey); err != nil {
		return AuthResponse{}, err
	}
	if _, err := X25519PublicKeyFromBase64(normalizedReq.DeviceExchangeKey); err != nil {
		return AuthResponse{}, err
	}

	challenge, err := s.loadChallenge(ctx, ChallengeRequest{
		Username:         normalizedReq.Username,
		DeviceName:       normalizedReq.DeviceName,
		Platform:         normalizedReq.Platform,
		DeviceSigningKey: normalizedReq.DeviceSigningKey,
	})
	if err != nil {
		return AuthResponse{}, err
	}

	if !VerifyEd25519Signature(normalizedReq.DeviceSigningKey, challenge, normalizedReq.DeviceSignature) {
		return AuthResponse{}, ErrInvalidCredentials
	}

	deviceProof := buildDeviceProofMessage(normalizedReq.DeviceSigningKey, normalizedReq.DeviceExchangeKey)
	if !VerifyEd25519Signature(normalizedReq.IdentitySigningKey, deviceProof, normalizedReq.IdentitySignatureOverDevice) {
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
		INSERT INTO users (username, display_name, public_key, identity_signing_key, identity_exchange_key)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, username, display_name, public_key, identity_signing_key, identity_exchange_key,
		          COALESCE(avatar_url, ''), COALESCE(bio, ''), server_id, created_at, is_active
	`, normalizedReq.Username, normalizedReq.DisplayName, normalizedReq.IdentitySigningKey, normalizedReq.IdentitySigningKey, normalizedReq.IdentityExchangeKey).Scan(
		&user.ID,
		&user.Username,
		&user.DisplayName,
		&user.PublicKey,
		&user.IdentitySigningKey,
		&user.IdentityExchangeKey,
		&user.AvatarURL,
		&user.Bio,
		&user.ServerID,
		&user.CreatedAt,
		&user.IsActive,
	)
	if err != nil {
		return AuthResponse{}, fmt.Errorf("insert user: %w", err)
	}

	deviceID, err := s.createDevice(ctx, tx, user.ID, createDeviceParams{
		LegacyDeviceKey:   normalizedReq.DeviceSigningKey,
		DeviceSigningKey:  normalizedReq.DeviceSigningKey,
		DeviceExchangeKey: normalizedReq.DeviceExchangeKey,
		DeviceName:        defaultString(normalizedReq.DeviceName, "Primary device"),
		Platform:          defaultString(normalizedReq.Platform, "web"),
		Trusted:           true,
	})
	if err != nil {
		return AuthResponse{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return AuthResponse{}, fmt.Errorf("commit register transaction: %w", err)
	}

	_ = s.redis.Del(ctx, s.challengeKey(ChallengeRequest{
		Username:         normalizedReq.Username,
		DeviceName:       normalizedReq.DeviceName,
		Platform:         normalizedReq.Platform,
		DeviceSigningKey: normalizedReq.DeviceSigningKey,
	})).Err()

	return s.issueAuthResponse(user, deviceID)
}

func (s *Service) Login(ctx context.Context, req LoginRequest) (AuthResponse, error) {
	if err := s.validate.Struct(req); err != nil {
		return AuthResponse{}, err
	}

	challengeRequest := ChallengeRequest{
		Username:         req.Username,
		DeviceID:         req.DeviceID,
		DeviceName:       req.DeviceName,
		Platform:         req.Platform,
		DeviceSigningKey: req.DeviceSigningKey,
	}

	challenge, err := s.loadChallenge(ctx, challengeRequest)
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

	deviceID := uuid.Nil
	if req.DeviceSigningKey != "" || req.DeviceID != "" || req.DeviceSignature != "" {
		signature := defaultString(req.DeviceSignature, req.Signature)
		device, err := s.getTrustedDeviceForLogin(ctx, user.ID, req)
		if err != nil {
			return AuthResponse{}, err
		}
		if !VerifyEd25519Signature(device.DeviceSigningKey, challenge, signature) {
			return AuthResponse{}, ErrInvalidCredentials
		}
		deviceID = device.ID
	} else {
		if !VerifyEd25519Signature(user.IdentitySigningKey, challenge, req.Signature) {
			return AuthResponse{}, ErrInvalidCredentials
		}

		deviceID, err = s.createDevice(ctx, s.db, user.ID, createDeviceParams{
			LegacyDeviceKey:   user.IdentitySigningKey,
			DeviceSigningKey:  user.IdentitySigningKey,
			DeviceExchangeKey: user.IdentityExchangeKey,
			DeviceName:        defaultString(req.DeviceName, "Legacy device"),
			Platform:          defaultString(req.Platform, "web"),
			Trusted:           true,
		})
		if err != nil {
			return AuthResponse{}, err
		}
	}

	_ = s.redis.Del(ctx, s.challengeKey(challengeRequest)).Err()

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
		RETURNING id, username, display_name, public_key, identity_signing_key, identity_exchange_key,
		          COALESCE(avatar_url, ''), COALESCE(bio, ''), server_id, created_at, is_active
	`, userID, req.DisplayName, req.AvatarURL, req.Bio).Scan(
		&user.ID,
		&user.Username,
		&user.DisplayName,
		&user.PublicKey,
		&user.IdentitySigningKey,
		&user.IdentityExchangeKey,
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
		SELECT id, user_id, device_key, device_signing_key, device_exchange_key,
		       COALESCE(device_name, ''), platform, COALESCE(push_token, ''),
		       COALESCE(last_seen, created_at), created_at, trusted, approved_by_device_id, revoked_at
		FROM devices
		WHERE user_id = $1
		  AND revoked_at IS NULL
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
			&record.DeviceSigningKey,
			&record.DeviceExchangeKey,
			&record.DeviceName,
			&record.Platform,
			&record.PushToken,
			&record.LastSeen,
			&record.CreatedAt,
			&record.Trusted,
			&record.ApprovedByDeviceID,
			&record.RevokedAt,
		); err != nil {
			return nil, fmt.Errorf("scan device: %w", err)
		}

		devices = append(devices, DeviceDTO{
			ID:                record.ID.String(),
			UserID:            record.UserID.String(),
			DeviceKey:         record.DeviceKey,
			DeviceSigningKey:  record.DeviceSigningKey,
			DeviceExchangeKey: record.DeviceExchangeKey,
			DeviceName:        record.DeviceName,
			Platform:          record.Platform,
			PushToken:         record.PushToken,
			LastSeen:          record.LastSeen,
			CreatedAt:         record.CreatedAt,
			Current:           record.ID == currentDeviceID,
			Trusted:           record.Trusted,
			ApprovedByDeviceID: func() string {
				if record.ApprovedByDeviceID == nil {
					return ""
				}
				return record.ApprovedByDeviceID.String()
			}(),
			RevokedAt: record.RevokedAt,
		})
	}

	return devices, rows.Err()
}

func (s *Service) RegisterPendingDevice(ctx context.Context, userID uuid.UUID, req RegisterPendingDeviceRequest) (DeviceDTO, error) {
	if err := s.validate.Struct(req); err != nil {
		return DeviceDTO{}, err
	}

	if _, err := Ed25519PublicKeyFromBase64(req.DeviceSigningKey); err != nil {
		return DeviceDTO{}, err
	}
	if _, err := X25519PublicKeyFromBase64(req.DeviceExchangeKey); err != nil {
		return DeviceDTO{}, err
	}

	deviceID, err := s.createDevice(ctx, s.db, userID, createDeviceParams{
		LegacyDeviceKey:   req.DeviceSigningKey,
		DeviceSigningKey:  req.DeviceSigningKey,
		DeviceExchangeKey: req.DeviceExchangeKey,
		DeviceName:        defaultString(req.DeviceName, "Pending device"),
		Platform:          defaultString(req.Platform, "web"),
		Trusted:           false,
	})
	if err != nil {
		return DeviceDTO{}, err
	}

	var record deviceRecord
	err = s.db.QueryRow(ctx, `
		SELECT id, user_id, device_key, device_signing_key, device_exchange_key,
		       COALESCE(device_name, ''), platform, COALESCE(push_token, ''),
		       COALESCE(last_seen, created_at), created_at, trusted, approved_by_device_id, revoked_at
		FROM devices
		WHERE id = $1
	`, deviceID).Scan(
		&record.ID,
		&record.UserID,
		&record.DeviceKey,
		&record.DeviceSigningKey,
		&record.DeviceExchangeKey,
		&record.DeviceName,
		&record.Platform,
		&record.PushToken,
		&record.LastSeen,
		&record.CreatedAt,
		&record.Trusted,
		&record.ApprovedByDeviceID,
		&record.RevokedAt,
	)
	if err != nil {
		return DeviceDTO{}, fmt.Errorf("load pending device: %w", err)
	}

	return DeviceDTO{
		ID:                record.ID.String(),
		UserID:            record.UserID.String(),
		DeviceKey:         record.DeviceKey,
		DeviceSigningKey:  record.DeviceSigningKey,
		DeviceExchangeKey: record.DeviceExchangeKey,
		DeviceName:        record.DeviceName,
		Platform:          record.Platform,
		PushToken:         record.PushToken,
		LastSeen:          record.LastSeen,
		CreatedAt:         record.CreatedAt,
		Trusted:           record.Trusted,
		ApprovedByDeviceID: func() string {
			if record.ApprovedByDeviceID == nil {
				return ""
			}
			return record.ApprovedByDeviceID.String()
		}(),
		RevokedAt: record.RevokedAt,
	}, nil
}

func (s *Service) RevokeDevice(ctx context.Context, userID, currentDeviceID, targetDeviceID uuid.UUID) error {
	if targetDeviceID == currentDeviceID {
		return ErrInvalidDevice
	}

	commandTag, err := s.db.Exec(ctx, `
		UPDATE devices
		SET revoked_at = NOW(), trusted = FALSE
		WHERE id = $1 AND user_id = $2 AND revoked_at IS NULL
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

func (s *Service) ApproveDevice(ctx context.Context, userID, currentDeviceID, targetDeviceID uuid.UUID) error {
	if targetDeviceID == currentDeviceID {
		return ErrInvalidDevice
	}

	commandTag, err := s.db.Exec(ctx, `
		UPDATE devices
		SET trusted = TRUE, approved_by_device_id = $3, revoked_at = NULL
		WHERE id = $1 AND user_id = $2
	`, targetDeviceID, userID, currentDeviceID)
	if err != nil {
		return fmt.Errorf("approve device: %w", err)
	}
	if commandTag.RowsAffected() == 0 {
		return ErrInvalidDevice
	}

	return nil
}

func (s *Service) SaveEncryptedBackup(ctx context.Context, userID uuid.UUID, req BackupExportRequest) (BackupExportResponse, error) {
	if err := s.validate.Struct(req); err != nil {
		return BackupExportResponse{}, err
	}

	backupVersion := req.BackupVersion
	if backupVersion == 0 {
		backupVersion = 1
	}

	var updatedAt time.Time
	err := s.db.QueryRow(ctx, `
		INSERT INTO encrypted_backups (user_id, backup_blob, backup_version)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id) DO UPDATE
		SET backup_blob = EXCLUDED.backup_blob,
		    backup_version = EXCLUDED.backup_version,
		    updated_at = NOW()
		RETURNING updated_at
	`, userID, req.BackupBlob, backupVersion).Scan(&updatedAt)
	if err != nil {
		return BackupExportResponse{}, fmt.Errorf("save encrypted backup: %w", err)
	}

	return BackupExportResponse{
		BackupVersion: backupVersion,
		UpdatedAt:     updatedAt,
	}, nil
}

func (s *Service) LoadEncryptedBackup(ctx context.Context, userID uuid.UUID) (BackupImportResponse, error) {
	var response BackupImportResponse
	err := s.db.QueryRow(ctx, `
		SELECT backup_blob, backup_version, updated_at
		FROM encrypted_backups
		WHERE user_id = $1
	`, userID).Scan(&response.BackupBlob, &response.BackupVersion, &response.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return BackupImportResponse{}, ErrBackupNotFound
	}
	if err != nil {
		return BackupImportResponse{}, fmt.Errorf("load encrypted backup: %w", err)
	}

	return response, nil
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
		SELECT id, username, display_name, public_key, identity_signing_key, identity_exchange_key,
		       COALESCE(avatar_url, ''), COALESCE(bio, ''), server_id, created_at, is_active
		FROM users
		WHERE username = $1
	`, username).Scan(
		&user.ID,
		&user.Username,
		&user.DisplayName,
		&user.PublicKey,
		&user.IdentitySigningKey,
		&user.IdentityExchangeKey,
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
		SELECT id, username, display_name, public_key, identity_signing_key, identity_exchange_key,
		       COALESCE(avatar_url, ''), COALESCE(bio, ''), server_id, created_at, is_active
		FROM users
		WHERE id = $1
	`, userID).Scan(
		&user.ID,
		&user.Username,
		&user.DisplayName,
		&user.PublicKey,
		&user.IdentitySigningKey,
		&user.IdentityExchangeKey,
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

type createDeviceParams struct {
	LegacyDeviceKey   string
	DeviceSigningKey  string
	DeviceExchangeKey string
	DeviceName        string
	Platform          string
	Trusted           bool
	ApprovedByDevice  *uuid.UUID
}

func (s *Service) createDevice(ctx context.Context, executor deviceCreator, userID uuid.UUID, params createDeviceParams) (uuid.UUID, error) {
	var deviceID uuid.UUID
	err := executor.QueryRow(ctx, `
		INSERT INTO devices (
			user_id, device_key, device_signing_key, device_exchange_key,
			device_name, platform, last_seen, trusted, approved_by_device_id
		)
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), $7, $8)
		RETURNING id
	`, userID, params.LegacyDeviceKey, params.DeviceSigningKey, params.DeviceExchangeKey, params.DeviceName, params.Platform, params.Trusted, params.ApprovedByDevice).Scan(&deviceID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("create device: %w", err)
	}

	return deviceID, nil
}

func (s *Service) loadChallenge(ctx context.Context, request ChallengeRequest) (string, error) {
	challenge, err := s.redis.Get(ctx, s.challengeKey(request)).Result()
	if errors.Is(err, redis.Nil) {
		return "", ErrChallengeExpired
	}
	if err != nil {
		return "", fmt.Errorf("load challenge: %w", err)
	}

	return challenge, nil
}

func (s *Service) challengeKey(request ChallengeRequest) string {
	parts := []string{"auth:challenge", strings.ToLower(request.Username)}
	if request.DeviceID != "" {
		parts = append(parts, strings.ToLower(request.DeviceID))
	}
	if request.DeviceSigningKey != "" {
		parts = append(parts, strings.ToLower(request.DeviceSigningKey))
	}

	return strings.Join(parts, ":")
}

func (s *Service) refreshBlacklistKey(deviceID uuid.UUID) string {
	return "auth:refresh:blacklist:" + deviceID.String()
}

func toUserDTO(user userRecord) UserDTO {
	return UserDTO{
		ID:                  user.ID.String(),
		Username:            user.Username,
		DisplayName:         user.DisplayName,
		PublicKey:           user.PublicKey,
		IdentitySigningKey:  user.IdentitySigningKey,
		IdentityExchangeKey: user.IdentityExchangeKey,
		AvatarURL:           user.AvatarURL,
		Bio:                 user.Bio,
		ServerID:            user.ServerID,
		CreatedAt:           user.CreatedAt,
	}
}

func (s *Service) normalizeRegisterRequest(req RegisterRequest) (RegisterRequest, error) {
	if req.IdentitySigningKey == "" && req.PublicKey != "" {
		req.IdentitySigningKey = req.PublicKey
	}
	if req.IdentityExchangeKey == "" && req.PublicKey != "" {
		req.IdentityExchangeKey = req.PublicKey
	}
	if req.DeviceSigningKey == "" && req.PublicKey != "" {
		req.DeviceSigningKey = req.PublicKey
	}
	if req.DeviceExchangeKey == "" && req.PublicKey != "" {
		req.DeviceExchangeKey = req.PublicKey
	}
	if req.DeviceSignature == "" {
		req.DeviceSignature = req.Signature
	}
	if req.IdentitySignatureOverDevice == "" {
		req.IdentitySignatureOverDevice = req.Signature
	}

	if req.IdentitySigningKey == "" || req.IdentityExchangeKey == "" || req.DeviceSigningKey == "" || req.DeviceExchangeKey == "" || req.DeviceSignature == "" || req.IdentitySignatureOverDevice == "" {
		return RegisterRequest{}, ErrInvalidCredentials
	}

	return req, nil
}

func (s *Service) getTrustedDeviceForLogin(ctx context.Context, userID uuid.UUID, req LoginRequest) (deviceRecord, error) {
	query := `
		SELECT id, user_id, device_key, device_signing_key, device_exchange_key,
		       COALESCE(device_name, ''), platform, COALESCE(push_token, ''),
		       COALESCE(last_seen, created_at), created_at, trusted, approved_by_device_id, revoked_at
		FROM devices
		WHERE user_id = $1 AND revoked_at IS NULL
	`
	args := []any{userID}

	switch {
	case req.DeviceID != "":
		query += ` AND id = $2`
		args = append(args, req.DeviceID)
	case req.DeviceSigningKey != "":
		query += ` AND device_signing_key = $2`
		args = append(args, req.DeviceSigningKey)
	default:
		return deviceRecord{}, ErrInvalidDevice
	}

	var device deviceRecord
	err := s.db.QueryRow(ctx, query, args...).Scan(
		&device.ID,
		&device.UserID,
		&device.DeviceKey,
		&device.DeviceSigningKey,
		&device.DeviceExchangeKey,
		&device.DeviceName,
		&device.Platform,
		&device.PushToken,
		&device.LastSeen,
		&device.CreatedAt,
		&device.Trusted,
		&device.ApprovedByDeviceID,
		&device.RevokedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return deviceRecord{}, ErrInvalidDevice
	}
	if err != nil {
		return deviceRecord{}, fmt.Errorf("get trusted device: %w", err)
	}
	if !device.Trusted || device.RevokedAt != nil {
		return deviceRecord{}, ErrInvalidDevice
	}

	return device, nil
}

func buildDeviceProofMessage(deviceSigningKey, deviceExchangeKey string) string {
	return deviceSigningKey + ":" + deviceExchangeKey
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}

	return value
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
