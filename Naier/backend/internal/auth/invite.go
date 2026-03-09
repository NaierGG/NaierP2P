package auth

import (
	"context"
	"crypto/rand"
	"encoding/base32"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type inviteRecord struct {
	ID         uuid.UUID
	Code       string
	Note       string
	CreatedBy  string
	MaxUses    int
	UseCount   int
	ExpiresAt  *time.Time
	DisabledAt *time.Time
	LastUsedAt *time.Time
	CreatedAt  time.Time
}

func (s *Service) ListInvites(ctx context.Context) ([]InviteDTO, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, code, COALESCE(note, ''), created_by, max_uses, use_count,
		       expires_at, disabled_at, last_used_at, created_at
		FROM beta_invites
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("list invites: %w", err)
	}
	defer rows.Close()

	invites := make([]InviteDTO, 0)
	for rows.Next() {
		var record inviteRecord
		if err := rows.Scan(
			&record.ID,
			&record.Code,
			&record.Note,
			&record.CreatedBy,
			&record.MaxUses,
			&record.UseCount,
			&record.ExpiresAt,
			&record.DisabledAt,
			&record.LastUsedAt,
			&record.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan invite: %w", err)
		}
		invites = append(invites, toInviteDTO(record))
	}

	return invites, rows.Err()
}

func (s *Service) CreateInvite(ctx context.Context, actor string, req CreateInviteRequest) (InviteDTO, error) {
	if err := s.validate.Struct(req); err != nil {
		return InviteDTO{}, err
	}

	code := normalizeInviteCode(req.Code)
	if code == "" {
		generated, err := generateInviteCode()
		if err != nil {
			return InviteDTO{}, err
		}
		code = generated
	}

	maxUses := 1
	if req.MaxUses != nil {
		maxUses = *req.MaxUses
	}

	var record inviteRecord
	err := s.db.QueryRow(ctx, `
		INSERT INTO beta_invites (code, note, created_by, max_uses, expires_at)
		VALUES ($1, NULLIF($2, ''), $3, $4, $5)
		RETURNING id, code, COALESCE(note, ''), created_by, max_uses, use_count,
		          expires_at, disabled_at, last_used_at, created_at
	`, code, strings.TrimSpace(req.Note), defaultString(actor, "admin-api"), maxUses, req.ExpiresAt).Scan(
		&record.ID,
		&record.Code,
		&record.Note,
		&record.CreatedBy,
		&record.MaxUses,
		&record.UseCount,
		&record.ExpiresAt,
		&record.DisabledAt,
		&record.LastUsedAt,
		&record.CreatedAt,
	)
	if err != nil {
		return InviteDTO{}, fmt.Errorf("create invite: %w", err)
	}

	return toInviteDTO(record), nil
}

func (s *Service) DisableInvite(ctx context.Context, inviteID uuid.UUID) error {
	commandTag, err := s.db.Exec(ctx, `
		UPDATE beta_invites
		SET disabled_at = NOW()
		WHERE id = $1 AND disabled_at IS NULL
	`, inviteID)
	if err != nil {
		return fmt.Errorf("disable invite: %w", err)
	}
	if commandTag.RowsAffected() == 0 {
		return ErrInviteInvalid
	}

	return nil
}

func (s *Service) consumeInvite(ctx context.Context, tx pgx.Tx, inviteCode string, userID uuid.UUID) error {
	trimmed := normalizeInviteCode(inviteCode)
	if trimmed == "" {
		if s.inviteOnly {
			return ErrInviteRequired
		}
		return nil
	}

	var record inviteRecord
	err := tx.QueryRow(ctx, `
		SELECT id, code, COALESCE(note, ''), created_by, max_uses, use_count,
		       expires_at, disabled_at, last_used_at, created_at
		FROM beta_invites
		WHERE code = $1
		FOR UPDATE
	`, trimmed).Scan(
		&record.ID,
		&record.Code,
		&record.Note,
		&record.CreatedBy,
		&record.MaxUses,
		&record.UseCount,
		&record.ExpiresAt,
		&record.DisabledAt,
		&record.LastUsedAt,
		&record.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrInviteInvalid
	}
	if err != nil {
		return fmt.Errorf("load invite: %w", err)
	}
	if record.DisabledAt != nil {
		return ErrInviteDisabled
	}
	if record.ExpiresAt != nil && record.ExpiresAt.Before(time.Now().UTC()) {
		return ErrInviteExpired
	}
	if record.UseCount >= record.MaxUses {
		return ErrInviteExhausted
	}

	if _, err := tx.Exec(ctx, `
		UPDATE beta_invites
		SET use_count = use_count + 1,
		    last_used_at = NOW()
		WHERE id = $1
	`, record.ID); err != nil {
		return fmt.Errorf("increment invite usage: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO beta_invite_redemptions (invite_id, user_id, code)
		VALUES ($1, $2, $3)
	`, record.ID, userID, record.Code); err != nil {
		return fmt.Errorf("record invite redemption: %w", err)
	}

	return nil
}

func normalizeInviteCode(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}

func toInviteDTO(record inviteRecord) InviteDTO {
	return InviteDTO{
		ID:         record.ID.String(),
		Code:       record.Code,
		Note:       record.Note,
		CreatedBy:  record.CreatedBy,
		MaxUses:    record.MaxUses,
		UseCount:   record.UseCount,
		ExpiresAt:  record.ExpiresAt,
		DisabledAt: record.DisabledAt,
		LastUsedAt: record.LastUsedAt,
		CreatedAt:  record.CreatedAt,
	}
}

func generateInviteCode() (string, error) {
	buffer := make([]byte, 10)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("generate invite code: %w", err)
	}

	encoded := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(buffer)
	return strings.ToUpper(encoded[:12]), nil
}
