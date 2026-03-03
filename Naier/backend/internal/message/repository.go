package message

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

type messageRecord struct {
	ID            uuid.UUID
	ChannelID     uuid.UUID
	SenderID      uuid.UUID
	Type          string
	Content       string
	IV            string
	ReplyToID     string
	IsEdited      bool
	IsDeleted     bool
	Signature     string
	ClientEventID string
	ServerEventID uuid.UUID
	Sequence      int64
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) IsChannelMember(ctx context.Context, channelID, userID uuid.UUID) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS (
		  SELECT 1 FROM channel_members WHERE channel_id = $1 AND user_id = $2
		)
	`, channelID, userID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check channel member: %w", err)
	}

	return exists, nil
}

func (r *Repository) GetByChannel(ctx context.Context, channelID uuid.UUID, cursor *time.Time, limit int) ([]MessageDTO, *time.Time, bool, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	rows, err := r.db.Query(ctx, `
		SELECT id, channel_id, sender_id, type, content, COALESCE(iv, ''),
		       COALESCE(reply_to_id::text, ''), is_edited, is_deleted,
		       COALESCE(signature, ''), COALESCE(client_event_id, ''),
		       server_event_id, sequence, created_at, updated_at
		FROM messages
		WHERE channel_id = $1
		  AND ($2::timestamptz IS NULL OR created_at < $2)
		ORDER BY created_at DESC
		LIMIT $3
	`, channelID, cursor, limit+1)
	if err != nil {
		return nil, nil, false, fmt.Errorf("list messages: %w", err)
	}
	defer rows.Close()

	records := make([]messageRecord, 0, limit+1)
	for rows.Next() {
		record, scanErr := scanMessage(rows)
		if scanErr != nil {
			return nil, nil, false, scanErr
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, false, err
	}

	hasMore := len(records) > limit
	if hasMore {
		records = records[:limit]
	}

	messages := make([]MessageDTO, 0, len(records))
	for _, record := range records {
		dto, convErr := r.toMessageDTO(ctx, record)
		if convErr != nil {
			return nil, nil, false, convErr
		}
		messages = append(messages, dto)
	}

	var nextCursor *time.Time
	if hasMore && len(records) > 0 {
		lastTime := records[len(records)-1].CreatedAt
		nextCursor = &lastTime
	}

	return messages, nextCursor, hasMore, nil
}

func (r *Repository) Create(ctx context.Context, channelID, senderID uuid.UUID, messageType, content, iv string, replyToID *uuid.UUID, signature, clientEventID string) (MessageDTO, error) {
	typeValue := messageType
	if typeValue == "" {
		typeValue = "text"
	}

	if clientEventID != "" {
		existing, err := r.GetBySenderClientEventID(ctx, senderID, clientEventID)
		switch {
		case err == nil:
			return existing, nil
		case errors.Is(err, pgx.ErrNoRows):
		default:
			return MessageDTO{}, err
		}
	}

	var record messageRecord
	err := r.db.QueryRow(ctx, `
		INSERT INTO messages (channel_id, sender_id, type, content, iv, reply_to_id, signature, client_event_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NULLIF($8, ''))
		RETURNING id, channel_id, sender_id, type, content, COALESCE(iv, ''),
		          COALESCE(reply_to_id::text, ''), is_edited, is_deleted,
		          COALESCE(signature, ''), COALESCE(client_event_id, ''),
		          server_event_id, sequence, created_at, updated_at
	`, channelID, senderID, typeValue, content, iv, replyToID, signature, clientEventID).Scan(
		&record.ID,
		&record.ChannelID,
		&record.SenderID,
		&record.Type,
		&record.Content,
		&record.IV,
		&record.ReplyToID,
		&record.IsEdited,
		&record.IsDeleted,
		&record.Signature,
		&record.ClientEventID,
		&record.ServerEventID,
		&record.Sequence,
		&record.CreatedAt,
		&record.UpdatedAt,
	)
	if err != nil {
		return MessageDTO{}, fmt.Errorf("create message: %w", err)
	}

	return r.toMessageDTO(ctx, record)
}

func (r *Repository) Update(ctx context.Context, messageID uuid.UUID, content, iv string) (MessageDTO, error) {
	var record messageRecord
	err := r.db.QueryRow(ctx, `
		UPDATE messages
		SET content = $2, iv = $3, is_edited = TRUE, updated_at = NOW(),
		    server_event_id = gen_random_uuid(),
		    sequence = nextval(pg_get_serial_sequence('messages', 'sequence'))
		WHERE id = $1
		RETURNING id, channel_id, sender_id, type, content, COALESCE(iv, ''),
		          COALESCE(reply_to_id::text, ''), is_edited, is_deleted,
		          COALESCE(signature, ''), COALESCE(client_event_id, ''),
		          server_event_id, sequence, created_at, updated_at
	`, messageID, content, iv).Scan(
		&record.ID,
		&record.ChannelID,
		&record.SenderID,
		&record.Type,
		&record.Content,
		&record.IV,
		&record.ReplyToID,
		&record.IsEdited,
		&record.IsDeleted,
		&record.Signature,
		&record.ClientEventID,
		&record.ServerEventID,
		&record.Sequence,
		&record.CreatedAt,
		&record.UpdatedAt,
	)
	if err != nil {
		return MessageDTO{}, fmt.Errorf("update message: %w", err)
	}

	return r.toMessageDTO(ctx, record)
}

func (r *Repository) SoftDelete(ctx context.Context, messageID uuid.UUID) (MessageDTO, error) {
	var record messageRecord
	err := r.db.QueryRow(ctx, `
		UPDATE messages
		SET content = '', iv = '', is_deleted = TRUE, updated_at = NOW(),
		    server_event_id = gen_random_uuid(),
		    sequence = nextval(pg_get_serial_sequence('messages', 'sequence'))
		WHERE id = $1
		RETURNING id, channel_id, sender_id, type, content, COALESCE(iv, ''),
		          COALESCE(reply_to_id::text, ''), is_edited, is_deleted,
		          COALESCE(signature, ''), COALESCE(client_event_id, ''),
		          server_event_id, sequence, created_at, updated_at
	`, messageID).Scan(
		&record.ID,
		&record.ChannelID,
		&record.SenderID,
		&record.Type,
		&record.Content,
		&record.IV,
		&record.ReplyToID,
		&record.IsEdited,
		&record.IsDeleted,
		&record.Signature,
		&record.ClientEventID,
		&record.ServerEventID,
		&record.Sequence,
		&record.CreatedAt,
		&record.UpdatedAt,
	)
	if err != nil {
		return MessageDTO{}, fmt.Errorf("soft delete message: %w", err)
	}

	return r.toMessageDTO(ctx, record)
}

func (r *Repository) GetMessageMeta(ctx context.Context, messageID uuid.UUID) (channelID, senderID uuid.UUID, err error) {
	err = r.db.QueryRow(ctx, `
		SELECT channel_id, sender_id
		FROM messages
		WHERE id = $1
	`, messageID).Scan(&channelID, &senderID)
	if err != nil {
		return uuid.Nil, uuid.Nil, fmt.Errorf("get message meta: %w", err)
	}

	return channelID, senderID, nil
}

func (r *Repository) AddReaction(ctx context.Context, messageID, userID uuid.UUID, emoji string) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO reactions (message_id, user_id, emoji)
		VALUES ($1, $2, $3)
		ON CONFLICT (message_id, user_id, emoji) DO NOTHING
	`, messageID, userID, emoji)
	if err != nil {
		return fmt.Errorf("add reaction: %w", err)
	}

	return nil
}

func (r *Repository) RemoveReaction(ctx context.Context, messageID, userID uuid.UUID, emoji string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM reactions WHERE message_id = $1 AND user_id = $2 AND emoji = $3`, messageID, userID, emoji)
	if err != nil {
		return fmt.Errorf("remove reaction: %w", err)
	}

	return nil
}

func (r *Repository) BulkGetAfter(ctx context.Context, channelID uuid.UUID, afterTime time.Time) ([]MessageDTO, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, channel_id, sender_id, type, content, COALESCE(iv, ''),
		       COALESCE(reply_to_id::text, ''), is_edited, is_deleted,
		       COALESCE(signature, ''), COALESCE(client_event_id, ''),
		       server_event_id, sequence, created_at, updated_at
		FROM messages
		WHERE channel_id = $1 AND created_at > $2
		ORDER BY created_at ASC
	`, channelID, afterTime)
	if err != nil {
		return nil, fmt.Errorf("bulk get messages: %w", err)
	}
	defer rows.Close()

	messages := make([]MessageDTO, 0)
	for rows.Next() {
		record, scanErr := scanMessage(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		dto, convErr := r.toMessageDTO(ctx, record)
		if convErr != nil {
			return nil, convErr
		}
		messages = append(messages, dto)
	}

	return messages, rows.Err()
}

func (r *Repository) MarkRead(ctx context.Context, channelID, userID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `
		UPDATE channel_members
		SET last_read_at = NOW()
		WHERE channel_id = $1 AND user_id = $2
	`, channelID, userID)
	if err != nil {
		return fmt.Errorf("mark read: %w", err)
	}

	return nil
}

func (r *Repository) MarkReadSequence(ctx context.Context, channelID, userID uuid.UUID, lastReadSequence int64) error {
	_, err := r.db.Exec(ctx, `
		UPDATE channel_members
		SET last_read_at = NOW(),
		    last_read_sequence = GREATEST(last_read_sequence, $3)
		WHERE channel_id = $1 AND user_id = $2
	`, channelID, userID, lastReadSequence)
	if err != nil {
		return fmt.Errorf("mark read sequence: %w", err)
	}

	return nil
}

func (r *Repository) GetBySenderClientEventID(ctx context.Context, senderID uuid.UUID, clientEventID string) (MessageDTO, error) {
	var record messageRecord
	err := r.db.QueryRow(ctx, `
		SELECT id, channel_id, sender_id, type, content, COALESCE(iv, ''),
		       COALESCE(reply_to_id::text, ''), is_edited, is_deleted,
		       COALESCE(signature, ''), COALESCE(client_event_id, ''),
		       server_event_id, sequence, created_at, updated_at
		FROM messages
		WHERE sender_id = $1 AND client_event_id = $2
	`, senderID, clientEventID).Scan(
		&record.ID,
		&record.ChannelID,
		&record.SenderID,
		&record.Type,
		&record.Content,
		&record.IV,
		&record.ReplyToID,
		&record.IsEdited,
		&record.IsDeleted,
		&record.Signature,
		&record.ClientEventID,
		&record.ServerEventID,
		&record.Sequence,
		&record.CreatedAt,
		&record.UpdatedAt,
	)
	if err != nil {
		return MessageDTO{}, fmt.Errorf("get message by client event id: %w", err)
	}

	return r.toMessageDTO(ctx, record)
}

func (r *Repository) GetEventSequence(ctx context.Context, serverEventID uuid.UUID) (int64, error) {
	var sequence int64
	err := r.db.QueryRow(ctx, `SELECT sequence FROM messages WHERE server_event_id = $1`, serverEventID).Scan(&sequence)
	if err != nil {
		return 0, fmt.Errorf("get event sequence: %w", err)
	}

	return sequence, nil
}

func (r *Repository) GetEventsAfter(ctx context.Context, userID uuid.UUID, afterSequence int64, limit int) ([]SyncEvent, bool, error) {
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}

	rows, err := r.db.Query(ctx, `
		SELECT m.id, m.channel_id, m.sender_id, m.type, m.content, COALESCE(m.iv, ''),
		       COALESCE(m.reply_to_id::text, ''), m.is_edited, m.is_deleted,
		       COALESCE(m.signature, ''), COALESCE(m.client_event_id, ''),
		       m.server_event_id, m.sequence, m.created_at, m.updated_at
		FROM messages m
		INNER JOIN channel_members cm ON cm.channel_id = m.channel_id
		WHERE cm.user_id = $1
		  AND m.sequence > $2
		ORDER BY m.sequence ASC
		LIMIT $3
	`, userID, afterSequence, limit+1)
	if err != nil {
		return nil, false, fmt.Errorf("get sync events: %w", err)
	}
	defer rows.Close()

	records := make([]messageRecord, 0, limit+1)
	for rows.Next() {
		record, scanErr := scanMessage(rows)
		if scanErr != nil {
			return nil, false, scanErr
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, false, err
	}

	hasMore := len(records) > limit
	if hasMore {
		records = records[:limit]
	}

	events := make([]SyncEvent, 0, len(records))
	for _, record := range records {
		messageDTO, convErr := r.toMessageDTO(ctx, record)
		if convErr != nil {
			return nil, false, convErr
		}
		events = append(events, SyncEvent{
			Type:      inferSyncEventType(record),
			Message:   messageDTO,
			EventID:   record.ServerEventID.String(),
			Sequence:  record.Sequence,
			ChannelID: record.ChannelID.String(),
		})
	}

	return events, hasMore, nil
}

func (r *Repository) GetChannelEventsAfter(ctx context.Context, channelID, userID uuid.UUID, afterSequence int64, limit int) ([]SyncEvent, bool, error) {
	member, err := r.IsChannelMember(ctx, channelID, userID)
	if err != nil {
		return nil, false, err
	}
	if !member {
		return nil, false, pgx.ErrNoRows
	}

	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}

	rows, err := r.db.Query(ctx, `
		SELECT id, channel_id, sender_id, type, content, COALESCE(iv, ''),
		       COALESCE(reply_to_id::text, ''), is_edited, is_deleted,
		       COALESCE(signature, ''), COALESCE(client_event_id, ''),
		       server_event_id, sequence, created_at, updated_at
		FROM messages
		WHERE channel_id = $1
		  AND sequence > $2
		ORDER BY sequence ASC
		LIMIT $3
	`, channelID, afterSequence, limit+1)
	if err != nil {
		return nil, false, fmt.Errorf("get channel sync events: %w", err)
	}
	defer rows.Close()

	records := make([]messageRecord, 0, limit+1)
	for rows.Next() {
		record, scanErr := scanMessage(rows)
		if scanErr != nil {
			return nil, false, scanErr
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, false, err
	}

	hasMore := len(records) > limit
	if hasMore {
		records = records[:limit]
	}

	events := make([]SyncEvent, 0, len(records))
	for _, record := range records {
		messageDTO, convErr := r.toMessageDTO(ctx, record)
		if convErr != nil {
			return nil, false, convErr
		}
		events = append(events, SyncEvent{
			Type:      inferSyncEventType(record),
			Message:   messageDTO,
			EventID:   record.ServerEventID.String(),
			Sequence:  record.Sequence,
			ChannelID: record.ChannelID.String(),
		})
	}

	return events, hasMore, nil
}

func (r *Repository) GetReactions(ctx context.Context, messageID uuid.UUID) ([]ReactionDTO, error) {
	rows, err := r.db.Query(ctx, `
		SELECT user_id::text, emoji
		FROM reactions
		WHERE message_id = $1
		ORDER BY emoji, user_id
	`, messageID)
	if err != nil {
		return nil, fmt.Errorf("get reactions: %w", err)
	}
	defer rows.Close()

	reactions := make([]ReactionDTO, 0)
	for rows.Next() {
		var reaction ReactionDTO
		if err := rows.Scan(&reaction.UserID, &reaction.Emoji); err != nil {
			return nil, fmt.Errorf("scan reaction: %w", err)
		}
		reactions = append(reactions, reaction)
	}

	return reactions, rows.Err()
}

func (r *Repository) toMessageDTO(ctx context.Context, record messageRecord) (MessageDTO, error) {
	reactions, err := r.GetReactions(ctx, record.ID)
	if err != nil {
		return MessageDTO{}, err
	}

	return MessageDTO{
		ID:            record.ID.String(),
		ChannelID:     record.ChannelID.String(),
		SenderID:      record.SenderID.String(),
		Type:          record.Type,
		Content:       record.Content,
		IV:            record.IV,
		ReplyToID:     record.ReplyToID,
		IsEdited:      record.IsEdited,
		IsDeleted:     record.IsDeleted,
		Signature:     record.Signature,
		ClientEventID: record.ClientEventID,
		ServerEventID: record.ServerEventID.String(),
		Sequence:      record.Sequence,
		CreatedAt:     record.CreatedAt,
		UpdatedAt:     record.UpdatedAt,
		Reactions:     reactions,
	}, nil
}

func scanMessage(row pgx.Row) (messageRecord, error) {
	var record messageRecord
	err := row.Scan(
		&record.ID,
		&record.ChannelID,
		&record.SenderID,
		&record.Type,
		&record.Content,
		&record.IV,
		&record.ReplyToID,
		&record.IsEdited,
		&record.IsDeleted,
		&record.Signature,
		&record.ClientEventID,
		&record.ServerEventID,
		&record.Sequence,
		&record.CreatedAt,
		&record.UpdatedAt,
	)
	if err != nil {
		return messageRecord{}, fmt.Errorf("scan message: %w", err)
	}

	return record, nil
}

func inferSyncEventType(record messageRecord) string {
	if record.IsDeleted {
		return "MESSAGE_DELETED"
	}
	if record.IsEdited {
		return "MESSAGE_UPDATED"
	}
	return "MESSAGE_NEW"
}
