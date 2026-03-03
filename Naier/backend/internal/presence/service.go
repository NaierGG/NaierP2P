package presence

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type OnlineUserProvider interface {
	GetOnlineUsersInChannel(channelID uuid.UUID) []uuid.UUID
}

type Service struct {
	repo     *Repository
	provider OnlineUserProvider
	ttl      time.Duration
}

func NewService(repo *Repository, provider OnlineUserProvider) *Service {
	return &Service{
		repo:     repo,
		provider: provider,
		ttl:      90 * time.Second,
	}
}

func (s *Service) UpdatePresence(ctx context.Context, userID uuid.UUID, status string) error {
	return s.repo.SetOnline(ctx, userID, status, s.ttl)
}

func (s *Service) SetTyping(ctx context.Context, channelID, userID uuid.UUID, isTyping bool) error {
	if isTyping {
		return s.repo.SetTyping(ctx, channelID, userID, 5*time.Second)
	}

	return s.repo.ClearTyping(ctx, channelID, userID)
}

func (s *Service) HeartbeatLoop(ctx context.Context, userID uuid.UUID, status string) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = s.repo.SetOnline(ctx, userID, status, s.ttl)
		}
	}
}

func (s *Service) GetChannelPresence(ctx context.Context, channelID uuid.UUID) map[uuid.UUID]string {
	if s.provider == nil {
		return map[uuid.UUID]string{}
	}

	userIDs := s.provider.GetOnlineUsersInChannel(channelID)
	return s.repo.GetBulkStatus(ctx, userIDs)
}
