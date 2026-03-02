package channel

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

type channelRecord struct {
	ID          uuid.UUID
	Type        string
	Name        string
	Description string
	InviteCode  string
	OwnerID     string
	IsEncrypted bool
	MaxMembers  int
	CreatedAt   time.Time
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, ownerID uuid.UUID, req CreateChannelRequest, inviteCode string) (channelRecord, error) {
	var channel channelRecord
	name := req.Name
	if req.Type == "dm" && strings.TrimSpace(name) == "" {
		name = "Direct Message"
	}

	isEncrypted := true
	if req.IsEncrypted != nil {
		isEncrypted = *req.IsEncrypted
	}

	maxMembers := 1000
	if req.MaxMembers != nil {
		maxMembers = *req.MaxMembers
	}

	err := r.db.QueryRow(ctx, `
		INSERT INTO channels (type, name, description, invite_code, owner_id, is_encrypted, max_members)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, type, COALESCE(name, ''), COALESCE(description, ''),
		          COALESCE(invite_code, ''), COALESCE(owner_id::text, ''), is_encrypted, max_members, created_at
	`, req.Type, name, req.Description, inviteCode, ownerID, isEncrypted, maxMembers).Scan(
		&channel.ID,
		&channel.Type,
		&channel.Name,
		&channel.Description,
		&channel.InviteCode,
		&channel.OwnerID,
		&channel.IsEncrypted,
		&channel.MaxMembers,
		&channel.CreatedAt,
	)
	if err != nil {
		return channelRecord{}, fmt.Errorf("create channel: %w", err)
	}

	return channel, nil
}

func (r *Repository) GetByID(ctx context.Context, channelID uuid.UUID) (channelRecord, error) {
	var channel channelRecord
	err := r.db.QueryRow(ctx, `
		SELECT id, type, COALESCE(name, ''), COALESCE(description, ''),
		       COALESCE(invite_code, ''), COALESCE(owner_id::text, ''), is_encrypted, max_members, created_at
		FROM channels
		WHERE id = $1
	`, channelID).Scan(
		&channel.ID,
		&channel.Type,
		&channel.Name,
		&channel.Description,
		&channel.InviteCode,
		&channel.OwnerID,
		&channel.IsEncrypted,
		&channel.MaxMembers,
		&channel.CreatedAt,
	)
	if err != nil {
		return channelRecord{}, err
	}

	return channel, nil
}

func (r *Repository) Update(ctx context.Context, channelID uuid.UUID, req UpdateChannelRequest) (channelRecord, error) {
	var channel channelRecord
	err := r.db.QueryRow(ctx, `
		UPDATE channels
		SET name = CASE WHEN $2 = '' THEN name ELSE $2 END,
		    description = CASE WHEN $3 = '' THEN description ELSE $3 END,
		    is_encrypted = COALESCE($4, is_encrypted),
		    max_members = COALESCE($5, max_members)
		WHERE id = $1
		RETURNING id, type, COALESCE(name, ''), COALESCE(description, ''),
		          COALESCE(invite_code, ''), COALESCE(owner_id::text, ''), is_encrypted, max_members, created_at
	`, channelID, req.Name, req.Description, req.IsEncrypted, req.MaxMembers).Scan(
		&channel.ID,
		&channel.Type,
		&channel.Name,
		&channel.Description,
		&channel.InviteCode,
		&channel.OwnerID,
		&channel.IsEncrypted,
		&channel.MaxMembers,
		&channel.CreatedAt,
	)
	if err != nil {
		return channelRecord{}, fmt.Errorf("update channel: %w", err)
	}

	return channel, nil
}

func (r *Repository) Delete(ctx context.Context, channelID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM channels WHERE id = $1`, channelID)
	if err != nil {
		return fmt.Errorf("delete channel: %w", err)
	}

	return nil
}

func (r *Repository) FindByInviteCode(ctx context.Context, inviteCode string) (channelRecord, error) {
	var channel channelRecord
	err := r.db.QueryRow(ctx, `
		SELECT id, type, COALESCE(name, ''), COALESCE(description, ''),
		       COALESCE(invite_code, ''), COALESCE(owner_id::text, ''), is_encrypted, max_members, created_at
		FROM channels
		WHERE invite_code = $1
	`, inviteCode).Scan(
		&channel.ID,
		&channel.Type,
		&channel.Name,
		&channel.Description,
		&channel.InviteCode,
		&channel.OwnerID,
		&channel.IsEncrypted,
		&channel.MaxMembers,
		&channel.CreatedAt,
	)
	if err != nil {
		return channelRecord{}, fmt.Errorf("find channel by invite: %w", err)
	}

	return channel, nil
}

func (r *Repository) GetUserChannels(ctx context.Context, userID uuid.UUID) ([]ChannelDTO, error) {
	rows, err := r.db.Query(ctx, `
		SELECT c.id, c.type, COALESCE(c.name, ''), COALESCE(c.description, ''),
		       COALESCE(c.invite_code, ''), COALESCE(c.owner_id::text, ''), c.is_encrypted, c.max_members,
		       c.created_at, counts.member_count,
		       COALESCE(m.id::text, ''), COALESCE(m.content, ''), COALESCE(m.type, ''), COALESCE(m.sender_id::text, ''),
		       COALESCE(m.created_at, c.created_at)
		FROM channel_members cm
		JOIN channels c ON c.id = cm.channel_id
		JOIN LATERAL (
		  SELECT COUNT(*)::int AS member_count
		  FROM channel_members cm2
		  WHERE cm2.channel_id = c.id
		) counts ON TRUE
		LEFT JOIN LATERAL (
		  SELECT id, content, type, sender_id, created_at
		  FROM messages
		  WHERE channel_id = c.id
		  ORDER BY created_at DESC
		  LIMIT 1
		) m ON TRUE
		WHERE cm.user_id = $1
		ORDER BY COALESCE(m.created_at, c.created_at) DESC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("get user channels: %w", err)
	}
	defer rows.Close()

	channels := make([]ChannelDTO, 0)
	for rows.Next() {
		var channel ChannelDTO
		var lastMessageID, lastContent, lastType, lastSenderID string
		var lastCreatedAt time.Time
		if err := rows.Scan(
			&channel.ID,
			&channel.Type,
			&channel.Name,
			&channel.Description,
			&channel.InviteCode,
			&channel.OwnerID,
			&channel.IsEncrypted,
			&channel.MaxMembers,
			&channel.CreatedAt,
			&channel.MemberCount,
			&lastMessageID,
			&lastContent,
			&lastType,
			&lastSenderID,
			&lastCreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan channel row: %w", err)
		}

		if lastMessageID != "" {
			channel.LastMessage = &ChannelLastMessage{
				ID:        lastMessageID,
				Content:   lastContent,
				Type:      lastType,
				SenderID:  lastSenderID,
				CreatedAt: lastCreatedAt,
			}
		}

		channels = append(channels, channel)
	}

	return channels, rows.Err()
}

func (r *Repository) GetMembers(ctx context.Context, channelID uuid.UUID) ([]ChannelMemberDTO, error) {
	rows, err := r.db.Query(ctx, `
		SELECT u.id::text, u.username, u.display_name, cm.role, cm.joined_at, cm.is_muted
		FROM channel_members cm
		JOIN users u ON u.id = cm.user_id
		WHERE cm.channel_id = $1
		ORDER BY CASE cm.role WHEN 'owner' THEN 0 WHEN 'admin' THEN 1 ELSE 2 END, cm.joined_at
	`, channelID)
	if err != nil {
		return nil, fmt.Errorf("get channel members: %w", err)
	}
	defer rows.Close()

	members := make([]ChannelMemberDTO, 0)
	for rows.Next() {
		var member ChannelMemberDTO
		if err := rows.Scan(&member.UserID, &member.Username, &member.DisplayName, &member.Role, &member.JoinedAt, &member.IsMuted); err != nil {
			return nil, fmt.Errorf("scan channel member: %w", err)
		}
		members = append(members, member)
	}

	return members, rows.Err()
}

func (r *Repository) AddMember(ctx context.Context, channelID, userID uuid.UUID, role string) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO channel_members (channel_id, user_id, role)
		VALUES ($1, $2, $3)
		ON CONFLICT (channel_id, user_id) DO NOTHING
	`, channelID, userID, role)
	if err != nil {
		return fmt.Errorf("add channel member: %w", err)
	}

	return nil
}

func (r *Repository) RemoveMember(ctx context.Context, channelID, userID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM channel_members WHERE channel_id = $1 AND user_id = $2`, channelID, userID)
	if err != nil {
		return fmt.Errorf("remove channel member: %w", err)
	}

	return nil
}

func (r *Repository) UpdateMemberRole(ctx context.Context, channelID, userID uuid.UUID, role string) error {
	_, err := r.db.Exec(ctx, `UPDATE channel_members SET role = $3 WHERE channel_id = $1 AND user_id = $2`, channelID, userID, role)
	if err != nil {
		return fmt.Errorf("update member role: %w", err)
	}

	return nil
}

func (r *Repository) GetMemberRole(ctx context.Context, channelID, userID uuid.UUID) (string, error) {
	var role string
	err := r.db.QueryRow(ctx, `SELECT role FROM channel_members WHERE channel_id = $1 AND user_id = $2`, channelID, userID).Scan(&role)
	if err != nil {
		return "", err
	}

	return role, nil
}

func (r *Repository) CountMembers(ctx context.Context, channelID uuid.UUID) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*)::int FROM channel_members WHERE channel_id = $1`, channelID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count members: %w", err)
	}

	return count, nil
}

func (r *Repository) UpdateInviteCode(ctx context.Context, channelID uuid.UUID, inviteCode string) error {
	_, err := r.db.Exec(ctx, `UPDATE channels SET invite_code = $2 WHERE id = $1`, channelID, inviteCode)
	if err != nil {
		return fmt.Errorf("update invite code: %w", err)
	}

	return nil
}

func (r *Repository) UpdateOwner(ctx context.Context, channelID uuid.UUID, ownerID *uuid.UUID) error {
	_, err := r.db.Exec(ctx, `UPDATE channels SET owner_id = $2 WHERE id = $1`, channelID, ownerID)
	if err != nil {
		return fmt.Errorf("update owner: %w", err)
	}

	return nil
}

func (r *Repository) FindDMChannel(ctx context.Context, userID1, userID2 uuid.UUID) (channelRecord, error) {
	var channel channelRecord
	err := r.db.QueryRow(ctx, `
		SELECT c.id, c.type, COALESCE(c.name, ''), COALESCE(c.description, ''),
		       COALESCE(c.invite_code, ''), COALESCE(c.owner_id::text, ''), c.is_encrypted, c.max_members, c.created_at
		FROM channels c
		JOIN channel_members cm ON cm.channel_id = c.id
		WHERE c.type = 'dm'
		  AND cm.user_id IN ($1, $2)
		GROUP BY c.id
		HAVING COUNT(DISTINCT cm.user_id) = 2
		LIMIT 1
	`, userID1, userID2).Scan(
		&channel.ID,
		&channel.Type,
		&channel.Name,
		&channel.Description,
		&channel.InviteCode,
		&channel.OwnerID,
		&channel.IsEncrypted,
		&channel.MaxMembers,
		&channel.CreatedAt,
	)
	if err != nil {
		return channelRecord{}, err
	}

	return channel, nil
}

func (r *Repository) IsMember(ctx context.Context, channelID, userID uuid.UUID) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS (
		  SELECT 1 FROM channel_members WHERE channel_id = $1 AND user_id = $2
		)
	`, channelID, userID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check channel membership: %w", err)
	}

	return exists, nil
}

func (r *Repository) NextOwnerCandidate(ctx context.Context, channelID, excludingUserID uuid.UUID) (*uuid.UUID, string, error) {
	var userID uuid.UUID
	var role string
	err := r.db.QueryRow(ctx, `
		SELECT user_id, role
		FROM channel_members
		WHERE channel_id = $1 AND user_id <> $2
		ORDER BY CASE role WHEN 'admin' THEN 0 WHEN 'member' THEN 1 ELSE 2 END, joined_at
		LIMIT 1
	`, channelID, excludingUserID).Scan(&userID, &role)
	if err == pgx.ErrNoRows {
		return nil, "", nil
	}
	if err != nil {
		return nil, "", fmt.Errorf("get next owner candidate: %w", err)
	}

	return &userID, role, nil
}
