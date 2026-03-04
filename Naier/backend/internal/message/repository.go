package message

import (
	"context"
	"errors"
	"fmt"
	"sort"
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

type reactionEventRecord struct {
	EventID   uuid.UUID
	Sequence  int64
	MessageID uuid.UUID
	ChannelID uuid.UUID
	UserID    uuid.UUID
	Emoji     string
	Action    string
	CreatedAt time.Time
}

type readEventRecord struct {
	EventID          uuid.UUID
	Sequence         int64
	ChannelID        uuid.UUID
	UserID           uuid.UUID
	LastReadSequence int64
	CreatedAt        time.Time
}

type deliveryTarget struct {
	DeviceID uuid.UUID
	UserID   uuid.UUID
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
		    sequence = nextval('sync_event_sequence')
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
		    sequence = nextval('sync_event_sequence')
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

func (r *Repository) GetMessageSequence(ctx context.Context, messageID uuid.UUID) (int64, error) {
	var sequence int64
	err := r.db.QueryRow(ctx, `SELECT sequence FROM messages WHERE id = $1`, messageID).Scan(&sequence)
	if err != nil {
		return 0, fmt.Errorf("get message sequence: %w", err)
	}

	return sequence, nil
}

func (r *Repository) EnsureMessageDeliveries(ctx context.Context, messageID, channelID, senderDeviceID uuid.UUID, sequence int64) error {
	targets, err := r.getDeliveryTargets(ctx, channelID)
	if err != nil {
		return err
	}
	if len(targets) == 0 {
		return nil
	}

	for _, target := range targets {
		status := "pending"
		var deliveredAt *time.Time
		var readAt *time.Time
		if target.DeviceID == senderDeviceID {
			now := time.Now().UTC()
			deliveredAt = &now
			readAt = &now
			status = "read"
		}

		_, execErr := r.db.Exec(ctx, `
			INSERT INTO message_deliveries (message_id, device_id, delivered_at, read_at, status)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (message_id, device_id) DO NOTHING
		`, messageID, target.DeviceID, deliveredAt, readAt, status)
		if execErr != nil {
			return fmt.Errorf("ensure message delivery rows: %w", execErr)
		}

		if target.DeviceID == senderDeviceID {
			_, execErr = r.db.Exec(ctx, `
				UPDATE channel_members
				SET last_read_at = NOW(),
				    last_read_sequence = GREATEST(last_read_sequence, $3)
				WHERE channel_id = $1 AND user_id = $2
			`, channelID, target.UserID, sequence)
			if execErr != nil {
				return fmt.Errorf("advance sender read pointer: %w", execErr)
			}
		}
	}

	return nil
}

func (r *Repository) MarkMessagesDelivered(ctx context.Context, deviceID uuid.UUID, messageIDs []uuid.UUID) error {
	if len(messageIDs) == 0 {
		return nil
	}

	_, err := r.db.Exec(ctx, `
		UPDATE message_deliveries
		SET delivered_at = COALESCE(delivered_at, NOW()),
		    status = CASE WHEN status = 'read' THEN status ELSE 'delivered' END
		WHERE device_id = $1
		  AND message_id = ANY($2)
	`, deviceID, messageIDs)
	if err != nil {
		return fmt.Errorf("mark messages delivered: %w", err)
	}

	return nil
}

func (r *Repository) AckMessageDelivery(ctx context.Context, deviceID, userID, messageID uuid.UUID) error {
	commandTag, err := r.db.Exec(ctx, `
		UPDATE message_deliveries md
		SET delivered_at = COALESCE(md.delivered_at, NOW()),
		    acked_at = NOW(),
		    status = CASE WHEN md.status = 'read' THEN md.status ELSE 'delivered' END
		FROM messages m
		JOIN channel_members cm ON cm.channel_id = m.channel_id
		WHERE md.message_id = m.id
		  AND md.message_id = $1
		  AND md.device_id = $2
		  AND cm.channel_id = m.channel_id
		  AND cm.user_id = $3
	`, messageID, deviceID, userID)
	if err != nil {
		return fmt.Errorf("ack message delivery: %w", err)
	}
	if commandTag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}

	return nil
}

func (r *Repository) MarkChannelReadDeliveries(ctx context.Context, deviceID, channelID uuid.UUID, lastReadSequence int64) error {
	_, err := r.db.Exec(ctx, `
		UPDATE message_deliveries md
		SET delivered_at = COALESCE(md.delivered_at, NOW()),
		    read_at = COALESCE(md.read_at, NOW()),
		    status = 'read'
		FROM messages m
		WHERE md.message_id = m.id
		  AND md.device_id = $1
		  AND m.channel_id = $2
		  AND m.sequence <= $3
	`, deviceID, channelID, lastReadSequence)
	if err != nil {
		return fmt.Errorf("mark channel read deliveries: %w", err)
	}

	return nil
}

func (r *Repository) AddReaction(ctx context.Context, messageID, userID uuid.UUID, emoji string) (*ReactionEventDTO, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin add reaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var channelID uuid.UUID
	err = tx.QueryRow(ctx, `SELECT channel_id FROM messages WHERE id = $1`, messageID).Scan(&channelID)
	if err != nil {
		return nil, fmt.Errorf("get reaction channel: %w", err)
	}

	result, err := tx.Exec(ctx, `
		INSERT INTO reactions (message_id, user_id, emoji)
		VALUES ($1, $2, $3)
		ON CONFLICT (message_id, user_id, emoji) DO NOTHING
	`, messageID, userID, emoji)
	if err != nil {
		return nil, fmt.Errorf("add reaction: %w", err)
	}
	if result.RowsAffected() == 0 {
		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("commit noop reaction add: %w", err)
		}
		return nil, nil
	}

	var record reactionEventRecord
	err = tx.QueryRow(ctx, `
		INSERT INTO reaction_events (message_id, channel_id, user_id, emoji, action)
		VALUES ($1, $2, $3, $4, 'add')
		RETURNING event_id, sequence, message_id, channel_id, user_id, emoji, action, created_at
	`, messageID, channelID, userID, emoji).Scan(
		&record.EventID,
		&record.Sequence,
		&record.MessageID,
		&record.ChannelID,
		&record.UserID,
		&record.Emoji,
		&record.Action,
		&record.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert reaction event: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit add reaction: %w", err)
	}

	dto := reactionEventDTO(record)
	return &dto, nil
}

func (r *Repository) RemoveReaction(ctx context.Context, messageID, userID uuid.UUID, emoji string) (*ReactionEventDTO, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin remove reaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var channelID uuid.UUID
	err = tx.QueryRow(ctx, `SELECT channel_id FROM messages WHERE id = $1`, messageID).Scan(&channelID)
	if err != nil {
		return nil, fmt.Errorf("get reaction channel: %w", err)
	}

	result, err := tx.Exec(ctx, `DELETE FROM reactions WHERE message_id = $1 AND user_id = $2 AND emoji = $3`, messageID, userID, emoji)
	if err != nil {
		return nil, fmt.Errorf("remove reaction: %w", err)
	}
	if result.RowsAffected() == 0 {
		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("commit noop reaction remove: %w", err)
		}
		return nil, nil
	}

	var record reactionEventRecord
	err = tx.QueryRow(ctx, `
		INSERT INTO reaction_events (message_id, channel_id, user_id, emoji, action)
		VALUES ($1, $2, $3, $4, 'remove')
		RETURNING event_id, sequence, message_id, channel_id, user_id, emoji, action, created_at
	`, messageID, channelID, userID, emoji).Scan(
		&record.EventID,
		&record.Sequence,
		&record.MessageID,
		&record.ChannelID,
		&record.UserID,
		&record.Emoji,
		&record.Action,
		&record.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert reaction event: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit remove reaction: %w", err)
	}

	dto := reactionEventDTO(record)
	return &dto, nil
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

func (r *Repository) MarkReadSequence(ctx context.Context, channelID, userID uuid.UUID, lastReadSequence int64) (*ReadStateDTO, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin mark read sequence: %w", err)
	}
	defer tx.Rollback(ctx)

	var updatedSequence int64
	err = tx.QueryRow(ctx, `
		UPDATE channel_members
		SET last_read_at = NOW(),
		    last_read_sequence = GREATEST(last_read_sequence, $3)
		WHERE channel_id = $1
		  AND user_id = $2
		  AND last_read_sequence < $3
		RETURNING last_read_sequence
	`, channelID, userID, lastReadSequence).Scan(&updatedSequence)
	if errors.Is(err, pgx.ErrNoRows) {
		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("commit noop read state: %w", err)
		}
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("mark read sequence: %w", err)
	}

	var record readEventRecord
	err = tx.QueryRow(ctx, `
		INSERT INTO read_events (channel_id, user_id, last_read_sequence)
		VALUES ($1, $2, $3)
		RETURNING event_id, sequence, channel_id, user_id, last_read_sequence, created_at
	`, channelID, userID, updatedSequence).Scan(
		&record.EventID,
		&record.Sequence,
		&record.ChannelID,
		&record.UserID,
		&record.LastReadSequence,
		&record.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert read event: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit mark read sequence: %w", err)
	}

	dto := readEventDTO(record)
	return &dto, nil
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
	err := r.db.QueryRow(ctx, `
		SELECT sequence
		FROM (
			SELECT sequence FROM messages WHERE server_event_id = $1
			UNION ALL
			SELECT sequence FROM reaction_events WHERE event_id = $1
			UNION ALL
			SELECT sequence FROM read_events WHERE event_id = $1
		) events
		LIMIT 1
	`, serverEventID).Scan(&sequence)
	if err != nil {
		return 0, fmt.Errorf("get event sequence: %w", err)
	}

	return sequence, nil
}

func (r *Repository) GetEventOffset(ctx context.Context, deviceID uuid.UUID, streamName string) (string, error) {
	var eventID uuid.UUID
	err := r.db.QueryRow(ctx, `
		SELECT last_event_id
		FROM event_offsets
		WHERE device_id = $1 AND stream_name = $2
	`, deviceID, streamName).Scan(&eventID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("get event offset: %w", err)
	}

	return eventID.String(), nil
}

func (r *Repository) UpsertEventOffset(ctx context.Context, deviceID uuid.UUID, streamName string, lastEventID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO event_offsets (device_id, stream_name, last_event_id, updated_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (device_id, stream_name)
		DO UPDATE SET last_event_id = EXCLUDED.last_event_id, updated_at = NOW()
	`, deviceID, streamName, lastEventID)
	if err != nil {
		return fmt.Errorf("upsert event offset: %w", err)
	}

	return nil
}

func (r *Repository) getDeliveryTargets(ctx context.Context, channelID uuid.UUID) ([]deliveryTarget, error) {
	rows, err := r.db.Query(ctx, `
		SELECT d.id, d.user_id
		FROM channel_members cm
		INNER JOIN devices d ON d.user_id = cm.user_id
		WHERE cm.channel_id = $1
		  AND d.revoked_at IS NULL
	`, channelID)
	if err != nil {
		return nil, fmt.Errorf("get delivery targets: %w", err)
	}
	defer rows.Close()

	targets := make([]deliveryTarget, 0)
	for rows.Next() {
		var target deliveryTarget
		if err := rows.Scan(&target.DeviceID, &target.UserID); err != nil {
			return nil, fmt.Errorf("scan delivery target: %w", err)
		}
		targets = append(targets, target)
	}

	return targets, rows.Err()
}

func (r *Repository) GetEventsAfter(ctx context.Context, userID uuid.UUID, afterSequence int64, limit int) ([]SyncEvent, bool, error) {
	messageRecords, err := r.getMessageEventsAfter(ctx, userID, afterSequence, limit)
	if err != nil {
		return nil, false, err
	}

	reactionRecords, err := r.getReactionEventsAfter(ctx, userID, afterSequence, limit)
	if err != nil {
		return nil, false, err
	}

	readRecords, err := r.getReadEventsAfter(ctx, userID, afterSequence, limit)
	if err != nil {
		return nil, false, err
	}

	return r.buildSyncEvents(ctx, messageRecords, reactionRecords, readRecords, limit)
}

func (r *Repository) GetChannelEventsAfter(ctx context.Context, channelID, userID uuid.UUID, afterSequence int64, limit int) ([]SyncEvent, bool, error) {
	member, err := r.IsChannelMember(ctx, channelID, userID)
	if err != nil {
		return nil, false, err
	}
	if !member {
		return nil, false, pgx.ErrNoRows
	}

	messageRecords, err := r.getChannelMessageEventsAfter(ctx, channelID, afterSequence, limit)
	if err != nil {
		return nil, false, err
	}

	reactionRecords, err := r.getChannelReactionEventsAfter(ctx, channelID, afterSequence, limit)
	if err != nil {
		return nil, false, err
	}

	readRecords, err := r.getChannelReadEventsAfter(ctx, channelID, afterSequence, limit)
	if err != nil {
		return nil, false, err
	}

	return r.buildSyncEvents(ctx, messageRecords, reactionRecords, readRecords, limit)
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

func (r *Repository) getMessageEventsAfter(ctx context.Context, userID uuid.UUID, afterSequence int64, limit int) ([]messageRecord, error) {
	return r.queryMessageRecords(ctx, `
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
}

func (r *Repository) getChannelMessageEventsAfter(ctx context.Context, channelID uuid.UUID, afterSequence int64, limit int) ([]messageRecord, error) {
	return r.queryMessageRecords(ctx, `
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
}

func (r *Repository) getReactionEventsAfter(ctx context.Context, userID uuid.UUID, afterSequence int64, limit int) ([]reactionEventRecord, error) {
	return r.queryReactionRecords(ctx, `
		SELECT re.event_id, re.sequence, re.message_id, re.channel_id, re.user_id, re.emoji, re.action, re.created_at
		FROM reaction_events re
		INNER JOIN channel_members cm ON cm.channel_id = re.channel_id
		WHERE cm.user_id = $1
		  AND re.sequence > $2
		ORDER BY re.sequence ASC
		LIMIT $3
	`, userID, afterSequence, limit+1)
}

func (r *Repository) getChannelReactionEventsAfter(ctx context.Context, channelID uuid.UUID, afterSequence int64, limit int) ([]reactionEventRecord, error) {
	return r.queryReactionRecords(ctx, `
		SELECT event_id, sequence, message_id, channel_id, user_id, emoji, action, created_at
		FROM reaction_events
		WHERE channel_id = $1
		  AND sequence > $2
		ORDER BY sequence ASC
		LIMIT $3
	`, channelID, afterSequence, limit+1)
}

func (r *Repository) getReadEventsAfter(ctx context.Context, userID uuid.UUID, afterSequence int64, limit int) ([]readEventRecord, error) {
	return r.queryReadRecords(ctx, `
		SELECT re.event_id, re.sequence, re.channel_id, re.user_id, re.last_read_sequence, re.created_at
		FROM read_events re
		INNER JOIN channel_members cm ON cm.channel_id = re.channel_id
		WHERE cm.user_id = $1
		  AND re.sequence > $2
		ORDER BY re.sequence ASC
		LIMIT $3
	`, userID, afterSequence, limit+1)
}

func (r *Repository) getChannelReadEventsAfter(ctx context.Context, channelID uuid.UUID, afterSequence int64, limit int) ([]readEventRecord, error) {
	return r.queryReadRecords(ctx, `
		SELECT event_id, sequence, channel_id, user_id, last_read_sequence, created_at
		FROM read_events
		WHERE channel_id = $1
		  AND sequence > $2
		ORDER BY sequence ASC
		LIMIT $3
	`, channelID, afterSequence, limit+1)
}

func (r *Repository) queryMessageRecords(ctx context.Context, sql string, args ...any) ([]messageRecord, error) {
	rows, err := r.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("query message sync records: %w", err)
	}
	defer rows.Close()

	records := make([]messageRecord, 0)
	for rows.Next() {
		record, scanErr := scanMessage(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		records = append(records, record)
	}

	return records, rows.Err()
}

func (r *Repository) queryReactionRecords(ctx context.Context, sql string, args ...any) ([]reactionEventRecord, error) {
	rows, err := r.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("query reaction sync records: %w", err)
	}
	defer rows.Close()

	records := make([]reactionEventRecord, 0)
	for rows.Next() {
		var record reactionEventRecord
		if err := rows.Scan(
			&record.EventID,
			&record.Sequence,
			&record.MessageID,
			&record.ChannelID,
			&record.UserID,
			&record.Emoji,
			&record.Action,
			&record.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan reaction event: %w", err)
		}
		records = append(records, record)
	}

	return records, rows.Err()
}

func (r *Repository) queryReadRecords(ctx context.Context, sql string, args ...any) ([]readEventRecord, error) {
	rows, err := r.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("query read sync records: %w", err)
	}
	defer rows.Close()

	records := make([]readEventRecord, 0)
	for rows.Next() {
		var record readEventRecord
		if err := rows.Scan(
			&record.EventID,
			&record.Sequence,
			&record.ChannelID,
			&record.UserID,
			&record.LastReadSequence,
			&record.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan read event: %w", err)
		}
		records = append(records, record)
	}

	return records, rows.Err()
}

func (r *Repository) buildSyncEvents(ctx context.Context, messageRecords []messageRecord, reactionRecords []reactionEventRecord, readRecords []readEventRecord, limit int) ([]SyncEvent, bool, error) {
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}

	events := make([]SyncEvent, 0, len(messageRecords)+len(reactionRecords)+len(readRecords))
	for _, record := range messageRecords {
		messageDTO, err := r.toMessageDTO(ctx, record)
		if err != nil {
			return nil, false, err
		}
		dto := messageDTO
		events = append(events, SyncEvent{
			Type:      inferSyncEventType(record),
			Message:   &dto,
			EventID:   record.ServerEventID.String(),
			Sequence:  record.Sequence,
			ChannelID: record.ChannelID.String(),
		})
	}
	for _, record := range reactionRecords {
		dto := reactionEventDTO(record)
		events = append(events, SyncEvent{
			Type:      "REACTION",
			Reaction:  &dto,
			EventID:   record.EventID.String(),
			Sequence:  record.Sequence,
			ChannelID: record.ChannelID.String(),
		})
	}
	for _, record := range readRecords {
		dto := readEventDTO(record)
		events = append(events, SyncEvent{
			Type:      "READ_STATE",
			ReadState: &dto,
			EventID:   record.EventID.String(),
			Sequence:  record.Sequence,
			ChannelID: record.ChannelID.String(),
		})
	}

	sort.Slice(events, func(i, j int) bool {
		return events[i].Sequence < events[j].Sequence
	})

	hasMore := len(events) > limit
	if hasMore {
		events = events[:limit]
	}

	return events, hasMore, nil
}

func reactionEventDTO(record reactionEventRecord) ReactionEventDTO {
	return ReactionEventDTO{
		MessageID: record.MessageID.String(),
		ChannelID: record.ChannelID.String(),
		UserID:    record.UserID.String(),
		Emoji:     record.Emoji,
		Action:    record.Action,
		EventID:   record.EventID.String(),
		Sequence:  record.Sequence,
		CreatedAt: record.CreatedAt,
	}
}

func readEventDTO(record readEventRecord) ReadStateDTO {
	return ReadStateDTO{
		ChannelID:        record.ChannelID.String(),
		UserID:           record.UserID.String(),
		LastReadSequence: record.LastReadSequence,
		EventID:          record.EventID.String(),
		Sequence:         record.Sequence,
		CreatedAt:        record.CreatedAt,
	}
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
