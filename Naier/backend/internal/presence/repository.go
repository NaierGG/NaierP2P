package presence

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type Repository struct {
	redis *redis.Client
}

func NewRepository(redisClient *redis.Client) *Repository {
	return &Repository{redis: redisClient}
}

func (r *Repository) SetOnline(ctx context.Context, userID uuid.UUID, status string, ttl time.Duration) error {
	key := r.presenceKey(userID)
	statusKey := r.presenceStatusKey(userID)

	pipe := r.redis.TxPipeline()
	pipe.HSet(ctx, key, "status", status, "updated_at", time.Now().UTC().Format(time.RFC3339Nano))
	pipe.Expire(ctx, key, ttl)
	pipe.Set(ctx, statusKey, status, ttl)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("set online status: %w", err)
	}

	return nil
}

func (r *Repository) GetStatus(ctx context.Context, userID uuid.UUID) string {
	status, err := r.redis.HGet(ctx, r.presenceKey(userID), "status").Result()
	if err != nil {
		return "offline"
	}

	return status
}

func (r *Repository) GetBulkStatus(ctx context.Context, userIDs []uuid.UUID) map[uuid.UUID]string {
	if len(userIDs) == 0 {
		return map[uuid.UUID]string{}
	}

	keys := make([]string, 0, len(userIDs))
	for _, userID := range userIDs {
		keys = append(keys, r.presenceStatusKey(userID))
	}

	values, err := r.redis.MGet(ctx, keys...).Result()
	if err != nil {
		return map[uuid.UUID]string{}
	}

	statuses := make(map[uuid.UUID]string, len(userIDs))
	for index, userID := range userIDs {
		if values[index] == nil {
			statuses[userID] = "offline"
			continue
		}
		status, ok := values[index].(string)
		if !ok || status == "" {
			statuses[userID] = "offline"
			continue
		}
		statuses[userID] = status
	}

	return statuses
}

func (r *Repository) SetTyping(ctx context.Context, channelID, userID uuid.UUID, ttl time.Duration) error {
	setKey := r.typingSetKey(channelID)
	memberKey := r.typingMemberKey(channelID, userID)

	pipe := r.redis.TxPipeline()
	pipe.SAdd(ctx, setKey, userID.String())
	pipe.Set(ctx, memberKey, "1", ttl)
	pipe.Expire(ctx, setKey, ttl)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("set typing: %w", err)
	}

	return nil
}

func (r *Repository) ClearTyping(ctx context.Context, channelID, userID uuid.UUID) error {
	pipe := r.redis.TxPipeline()
	pipe.Del(ctx, r.typingMemberKey(channelID, userID))
	pipe.SRem(ctx, r.typingSetKey(channelID), userID.String())
	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("clear typing: %w", err)
	}

	return nil
}

func (r *Repository) GetTypingUsers(ctx context.Context, channelID uuid.UUID) []uuid.UUID {
	members, err := r.redis.SMembers(ctx, r.typingSetKey(channelID)).Result()
	if err != nil {
		return nil
	}

	users := make([]uuid.UUID, 0, len(members))
	for _, member := range members {
		userID, err := uuid.Parse(member)
		if err != nil {
			continue
		}

		exists, err := r.redis.Exists(ctx, r.typingMemberKey(channelID, userID)).Result()
		if err != nil || exists == 0 {
			_ = r.redis.SRem(ctx, r.typingSetKey(channelID), member).Err()
			continue
		}

		users = append(users, userID)
	}

	return users
}

func (r *Repository) presenceKey(userID uuid.UUID) string {
	return "user:presence:" + userID.String()
}

func (r *Repository) presenceStatusKey(userID uuid.UUID) string {
	return "user:presence:status:" + userID.String()
}

func (r *Repository) typingSetKey(channelID uuid.UUID) string {
	return "typing:" + channelID.String()
}

func (r *Repository) typingMemberKey(channelID, userID uuid.UUID) string {
	return fmt.Sprintf("typing:%s:%s", channelID.String(), userID.String())
}
