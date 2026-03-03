package message

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	appws "github.com/naier/backend/internal/websocket"

	validatorpkg "github.com/naier/backend/pkg/validator"
)

var (
	ErrMessageNotFound = errors.New("message not found")
	ErrMessageDenied   = errors.New("message action forbidden")
)

type Service struct {
	repo     *Repository
	validate *validatorpkg.Validator
}

func NewService(repo *Repository, validate *validatorpkg.Validator) *Service {
	return &Service{repo: repo, validate: validate}
}

func (s *Service) ListByChannel(ctx context.Context, channelID, userID uuid.UUID, cursor string, limit int) (ListResponse, error) {
	isMember, err := s.repo.IsChannelMember(ctx, channelID, userID)
	if err != nil || !isMember {
		return ListResponse{}, ErrMessageDenied
	}

	var cursorTime *time.Time
	if cursor != "" {
		parsed, err := time.Parse(time.RFC3339Nano, cursor)
		if err != nil {
			return ListResponse{}, fmt.Errorf("invalid cursor: %w", err)
		}
		cursorTime = &parsed
	}

	messages, nextCursor, hasMore, err := s.repo.GetByChannel(ctx, channelID, cursorTime, limit)
	if err != nil {
		return ListResponse{}, err
	}

	response := ListResponse{
		Messages: messages,
		HasMore:  hasMore,
	}
	if nextCursor != nil {
		response.NextCursor = nextCursor.Format(time.RFC3339Nano)
	}

	return response, nil
}

func (s *Service) CreateHTTP(ctx context.Context, channelID, userID uuid.UUID, req CreateMessageRequest) (MessageDTO, error) {
	if err := s.validate.Struct(req); err != nil {
		return MessageDTO{}, err
	}

	isMember, err := s.repo.IsChannelMember(ctx, channelID, userID)
	if err != nil || !isMember {
		return MessageDTO{}, ErrMessageDenied
	}

	var replyToID *uuid.UUID
	if req.ReplyToID != "" {
		parsed, err := uuid.Parse(req.ReplyToID)
		if err != nil {
			return MessageDTO{}, fmt.Errorf("invalid reply_to_id: %w", err)
		}
		replyToID = &parsed
	}

	return s.repo.Create(ctx, channelID, userID, req.Type, req.Content, req.IV, replyToID, req.Signature, req.ClientEventID)
}

func (s *Service) UpdateHTTP(ctx context.Context, messageID, userID uuid.UUID, req UpdateMessageRequest) (MessageDTO, error) {
	if err := s.validate.Struct(req); err != nil {
		return MessageDTO{}, err
	}

	_, senderID, err := s.repo.GetMessageMeta(ctx, messageID)
	if errors.Is(err, pgx.ErrNoRows) {
		return MessageDTO{}, ErrMessageNotFound
	}
	if err != nil {
		return MessageDTO{}, err
	}
	if senderID != userID {
		return MessageDTO{}, ErrMessageDenied
	}

	return s.repo.Update(ctx, messageID, req.Content, req.IV)
}

func (s *Service) DeleteHTTP(ctx context.Context, messageID, userID uuid.UUID) (MessageDTO, error) {
	_, senderID, err := s.repo.GetMessageMeta(ctx, messageID)
	if errors.Is(err, pgx.ErrNoRows) {
		return MessageDTO{}, ErrMessageNotFound
	}
	if err != nil {
		return MessageDTO{}, err
	}
	if senderID != userID {
		return MessageDTO{}, ErrMessageDenied
	}

	return s.repo.SoftDelete(ctx, messageID)
}

func (s *Service) Create(ctx context.Context, userID, channelID uuid.UUID, content, iv string, replyToID *uuid.UUID, clientEventID string) ([]byte, error) {
	message, err := s.repo.Create(ctx, channelID, userID, "text", content, iv, replyToID, "", clientEventID)
	if err != nil {
		return nil, err
	}

	return marshalEvent(appws.WSEvent{
		Type:    appws.EventMessageNew,
		Payload: mustJSON(message),
	})
}

func (s *Service) Edit(ctx context.Context, userID, messageID uuid.UUID, content, iv string) (uuid.UUID, []byte, error) {
	channelID, senderID, err := s.repo.GetMessageMeta(ctx, messageID)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, nil, ErrMessageNotFound
	}
	if err != nil {
		return uuid.Nil, nil, err
	}
	if senderID != userID {
		return uuid.Nil, nil, ErrMessageDenied
	}

	message, err := s.repo.Update(ctx, messageID, content, iv)
	if err != nil {
		return uuid.Nil, nil, err
	}

	event, err := marshalEvent(appws.WSEvent{
		Type:    appws.EventMessageUpdated,
		Payload: mustJSON(message),
	})
	if err != nil {
		return uuid.Nil, nil, err
	}

	return channelID, event, nil
}

func (s *Service) Delete(ctx context.Context, userID, messageID uuid.UUID) (uuid.UUID, []byte, error) {
	channelID, senderID, err := s.repo.GetMessageMeta(ctx, messageID)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, nil, ErrMessageNotFound
	}
	if err != nil {
		return uuid.Nil, nil, err
	}
	if senderID != userID {
		return uuid.Nil, nil, ErrMessageDenied
	}

	if _, err := s.repo.SoftDelete(ctx, messageID); err != nil {
		return uuid.Nil, nil, err
	}

	event, err := marshalEvent(appws.WSEvent{
		Type: appws.EventMessageDeleted,
		Payload: mustJSON(map[string]string{
			"messageId": messageID.String(),
			"channelId": channelID.String(),
		}),
	})
	if err != nil {
		return uuid.Nil, nil, err
	}

	return channelID, event, nil
}

func (s *Service) AddReaction(ctx context.Context, userID, messageID uuid.UUID, emoji string) (uuid.UUID, []byte, error) {
	channelID, _, err := s.repo.GetMessageMeta(ctx, messageID)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, nil, ErrMessageNotFound
	}
	if err != nil {
		return uuid.Nil, nil, err
	}

	if err := s.repo.AddReaction(ctx, messageID, userID, emoji); err != nil {
		return uuid.Nil, nil, err
	}

	event, err := marshalEvent(appws.WSEvent{
		Type: appws.EventReaction,
		Payload: mustJSON(map[string]string{
			"messageId": messageID.String(),
			"emoji":     emoji,
			"userId":    userID.String(),
			"action":    "add",
		}),
	})
	if err != nil {
		return uuid.Nil, nil, err
	}

	return channelID, event, nil
}

func (s *Service) RemoveReaction(ctx context.Context, userID, messageID uuid.UUID, emoji string) (uuid.UUID, []byte, error) {
	channelID, _, err := s.repo.GetMessageMeta(ctx, messageID)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, nil, ErrMessageNotFound
	}
	if err != nil {
		return uuid.Nil, nil, err
	}

	if err := s.repo.RemoveReaction(ctx, messageID, userID, emoji); err != nil {
		return uuid.Nil, nil, err
	}

	event, err := marshalEvent(appws.WSEvent{
		Type: appws.EventReaction,
		Payload: mustJSON(map[string]string{
			"messageId": messageID.String(),
			"emoji":     emoji,
			"userId":    userID.String(),
			"action":    "remove",
		}),
	})
	if err != nil {
		return uuid.Nil, nil, err
	}

	return channelID, event, nil
}

func (s *Service) MarkRead(ctx context.Context, userID, channelID, messageID uuid.UUID) error {
	isMember, err := s.repo.IsChannelMember(ctx, channelID, userID)
	if err != nil || !isMember {
		return ErrMessageDenied
	}

	return s.repo.MarkRead(ctx, channelID, userID)
}

func (s *Service) MarkReadSequence(ctx context.Context, userID, channelID uuid.UUID, lastReadSequence int64) error {
	isMember, err := s.repo.IsChannelMember(ctx, channelID, userID)
	if err != nil || !isMember {
		return ErrMessageDenied
	}

	return s.repo.MarkReadSequence(ctx, channelID, userID, lastReadSequence)
}

func (s *Service) SyncEvents(ctx context.Context, userID uuid.UUID, afterEventID string, limit int) (SyncResponse, error) {
	afterSequence, err := s.resolveAfterSequence(ctx, afterEventID)
	if err != nil {
		return SyncResponse{}, err
	}

	events, hasMore, err := s.repo.GetEventsAfter(ctx, userID, afterSequence, limit)
	if err != nil {
		return SyncResponse{}, err
	}

	response := SyncResponse{
		Events:   events,
		HasMore:  hasMore,
	}
	if len(events) > 0 {
		response.LastEventID = events[len(events)-1].EventID
	}

	return response, nil
}

func (s *Service) SyncChannelEvents(ctx context.Context, userID, channelID uuid.UUID, afterEventID string, limit int) (SyncResponse, error) {
	afterSequence, err := s.resolveAfterSequence(ctx, afterEventID)
	if err != nil {
		return SyncResponse{}, err
	}

	events, hasMore, err := s.repo.GetChannelEventsAfter(ctx, channelID, userID, afterSequence, limit)
	if errors.Is(err, pgx.ErrNoRows) {
		return SyncResponse{}, ErrMessageDenied
	}
	if err != nil {
		return SyncResponse{}, err
	}

	response := SyncResponse{
		Events:   events,
		HasMore:  hasMore,
	}
	if len(events) > 0 {
		response.LastEventID = events[len(events)-1].EventID
	}

	return response, nil
}

func (s *Service) resolveAfterSequence(ctx context.Context, afterEventID string) (int64, error) {
	if afterEventID == "" {
		return 0, nil
	}

	eventID, err := uuid.Parse(afterEventID)
	if err != nil {
		return 0, fmt.Errorf("invalid after event id: %w", err)
	}

	sequence, err := s.repo.GetEventSequence(ctx, eventID)
	if err != nil {
		return 0, err
	}

	return sequence, nil
}

func marshalEvent(event appws.WSEvent) ([]byte, error) {
	return json.Marshal(event)
}

func mustJSON(value any) json.RawMessage {
	encoded, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return json.RawMessage(encoded)
}
