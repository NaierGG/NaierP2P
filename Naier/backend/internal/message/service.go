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

const globalSyncStreamName = "global"

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

	reactionEvent, err := s.repo.AddReaction(ctx, messageID, userID, emoji)
	if err != nil {
		return uuid.Nil, nil, err
	}
	if reactionEvent == nil {
		return channelID, nil, nil
	}

	event, err := marshalEvent(appws.WSEvent{
		Type:    appws.EventReaction,
		Payload: mustJSON(reactionEvent),
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

	reactionEvent, err := s.repo.RemoveReaction(ctx, messageID, userID, emoji)
	if err != nil {
		return uuid.Nil, nil, err
	}
	if reactionEvent == nil {
		return channelID, nil, nil
	}

	event, err := marshalEvent(appws.WSEvent{
		Type:    appws.EventReaction,
		Payload: mustJSON(reactionEvent),
	})
	if err != nil {
		return uuid.Nil, nil, err
	}

	return channelID, event, nil
}

func (s *Service) MarkRead(ctx context.Context, userID, channelID, messageID uuid.UUID) ([]byte, error) {
	isMember, err := s.repo.IsChannelMember(ctx, channelID, userID)
	if err != nil || !isMember {
		return nil, ErrMessageDenied
	}

	lastReadSequence, err := s.repo.GetMessageSequence(ctx, messageID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrMessageNotFound
	}
	if err != nil {
		return nil, err
	}

	return s.MarkReadSequence(ctx, userID, channelID, lastReadSequence)
}

func (s *Service) MarkReadSequence(ctx context.Context, userID, channelID uuid.UUID, lastReadSequence int64) ([]byte, error) {
	isMember, err := s.repo.IsChannelMember(ctx, channelID, userID)
	if err != nil || !isMember {
		return nil, ErrMessageDenied
	}

	readState, err := s.repo.MarkReadSequence(ctx, channelID, userID, lastReadSequence)
	if err != nil {
		return nil, err
	}
	if readState == nil {
		return nil, nil
	}

	return marshalEvent(appws.WSEvent{
		Type:    appws.EventReadState,
		Payload: mustJSON(readState),
	})
}

func (s *Service) SyncEvents(ctx context.Context, userID, deviceID uuid.UUID, afterEventID string, limit int) (SyncResponse, error) {
	afterEventID, err := s.resolveAfterEventID(ctx, deviceID, globalSyncStreamName, afterEventID)
	if err != nil {
		return SyncResponse{}, err
	}

	afterSequence, err := s.resolveAfterSequence(ctx, afterEventID)
	if err != nil {
		return SyncResponse{}, err
	}

	events, hasMore, err := s.repo.GetEventsAfter(ctx, userID, afterSequence, limit)
	if err != nil {
		return SyncResponse{}, err
	}

	response := SyncResponse{
		Events:      events,
		HasMore:     hasMore,
		LastEventID: afterEventID,
	}
	if len(events) > 0 {
		response.LastEventID = events[len(events)-1].EventID
	}
	if response.LastEventID != "" {
		if err := s.persistOffset(ctx, deviceID, globalSyncStreamName, response.LastEventID); err != nil {
			return SyncResponse{}, err
		}
	}

	return response, nil
}

func (s *Service) SyncChannelEvents(ctx context.Context, userID, deviceID, channelID uuid.UUID, afterEventID string, limit int) (SyncResponse, error) {
	streamName := channelSyncStreamName(channelID)
	afterEventID, err := s.resolveAfterEventID(ctx, deviceID, streamName, afterEventID)
	if err != nil {
		return SyncResponse{}, err
	}

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
		Events:      events,
		HasMore:     hasMore,
		LastEventID: afterEventID,
	}
	if len(events) > 0 {
		response.LastEventID = events[len(events)-1].EventID
	}
	if response.LastEventID != "" {
		if err := s.persistOffset(ctx, deviceID, streamName, response.LastEventID); err != nil {
			return SyncResponse{}, err
		}
	}

	return response, nil
}

func (s *Service) resolveAfterEventID(ctx context.Context, deviceID uuid.UUID, streamName, afterEventID string) (string, error) {
	if afterEventID != "" {
		return afterEventID, nil
	}

	return s.repo.GetEventOffset(ctx, deviceID, streamName)
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

func (s *Service) persistOffset(ctx context.Context, deviceID uuid.UUID, streamName, eventID string) error {
	parsed, err := uuid.Parse(eventID)
	if err != nil {
		return fmt.Errorf("invalid event offset id: %w", err)
	}

	return s.repo.UpsertEventOffset(ctx, deviceID, streamName, parsed)
}

func channelSyncStreamName(channelID uuid.UUID) string {
	return fmt.Sprintf("channel:%s", channelID.String())
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
