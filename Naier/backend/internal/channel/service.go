package channel

import (
	"context"
	"crypto/rand"
	"encoding/base32"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	validatorpkg "github.com/naier/backend/pkg/validator"
)

var (
	ErrChannelNotFound = errors.New("channel not found")
	ErrForbidden       = errors.New("forbidden")
	ErrChannelFull     = errors.New("channel is full")
)

type Service struct {
	repo     *Repository
	validate *validatorpkg.Validator
}

func NewService(repo *Repository, validate *validatorpkg.Validator) *Service {
	return &Service{repo: repo, validate: validate}
}

func (s *Service) Create(ctx context.Context, ownerID uuid.UUID, req CreateChannelRequest) (ChannelDTO, error) {
	if err := s.validate.Struct(req); err != nil {
		return ChannelDTO{}, err
	}

	inviteCode, err := s.GenerateInviteCodeValue()
	if err != nil {
		return ChannelDTO{}, err
	}

	record, err := s.repo.Create(ctx, ownerID, req, inviteCode)
	if err != nil {
		return ChannelDTO{}, err
	}

	if err := s.repo.AddMember(ctx, record.ID, ownerID, "owner"); err != nil {
		return ChannelDTO{}, err
	}

	return toChannelDTO(record, 1), nil
}

func (s *Service) Join(ctx context.Context, inviteCode string, userID uuid.UUID) (ChannelDTO, error) {
	record, err := s.repo.FindByInviteCode(ctx, inviteCode)
	if err != nil {
		return ChannelDTO{}, ErrChannelNotFound
	}

	count, err := s.repo.CountMembers(ctx, record.ID)
	if err != nil {
		return ChannelDTO{}, err
	}
	if count >= record.MaxMembers {
		return ChannelDTO{}, ErrChannelFull
	}

	if err := s.repo.AddMember(ctx, record.ID, userID, "member"); err != nil {
		return ChannelDTO{}, err
	}

	return toChannelDTO(record, count+1), nil
}

func (s *Service) Leave(ctx context.Context, channelID, userID uuid.UUID) error {
	record, err := s.repo.GetByID(ctx, channelID)
	if err != nil {
		return ErrChannelNotFound
	}

	role, err := s.repo.GetMemberRole(ctx, channelID, userID)
	if err != nil {
		return ErrForbidden
	}

	if role == "owner" {
		nextOwner, _, err := s.repo.NextOwnerCandidate(ctx, channelID, userID)
		if err != nil {
			return err
		}
		if nextOwner != nil {
			if err := s.repo.UpdateOwner(ctx, channelID, nextOwner); err != nil {
				return err
			}
			if err := s.repo.UpdateMemberRole(ctx, channelID, *nextOwner, "owner"); err != nil {
				return err
			}
		} else {
			if err := s.repo.UpdateOwner(ctx, channelID, nil); err != nil {
				return err
			}
		}
	}

	return s.repo.RemoveMember(ctx, record.ID, userID)
}

func (s *Service) GenerateInviteCode(ctx context.Context, channelID, actorID uuid.UUID) (string, error) {
	role, err := s.repo.GetMemberRole(ctx, channelID, actorID)
	if err != nil {
		return "", ErrForbidden
	}
	if role != "owner" && role != "admin" {
		return "", ErrForbidden
	}

	inviteCode, err := s.GenerateInviteCodeValue()
	if err != nil {
		return "", err
	}

	if err := s.repo.UpdateInviteCode(ctx, channelID, inviteCode); err != nil {
		return "", err
	}

	return inviteCode, nil
}

func (s *Service) GetOrCreateDM(ctx context.Context, userID1, userID2 uuid.UUID) (ChannelDTO, error) {
	record, err := s.repo.FindDMChannel(ctx, userID1, userID2)
	if err == nil {
		count, countErr := s.repo.CountMembers(ctx, record.ID)
		if countErr != nil {
			return ChannelDTO{}, countErr
		}
		return toChannelDTO(record, count), nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return ChannelDTO{}, err
	}

	req := CreateChannelRequest{Type: "dm", Name: "Direct Message"}
	record, createErr := s.repo.Create(ctx, userID1, req, "")
	if createErr != nil {
		return ChannelDTO{}, createErr
	}

	if err := s.repo.AddMember(ctx, record.ID, userID1, "owner"); err != nil {
		return ChannelDTO{}, err
	}
	if err := s.repo.AddMember(ctx, record.ID, userID2, "member"); err != nil {
		return ChannelDTO{}, err
	}

	return toChannelDTO(record, 2), nil
}

func (s *Service) GetUserChannels(ctx context.Context, userID uuid.UUID) ([]ChannelDTO, error) {
	return s.repo.GetUserChannels(ctx, userID)
}

func (s *Service) GetChannel(ctx context.Context, channelID, userID uuid.UUID) (ChannelDTO, error) {
	isMember, err := s.repo.IsMember(ctx, channelID, userID)
	if err != nil || !isMember {
		return ChannelDTO{}, ErrForbidden
	}

	record, err := s.repo.GetByID(ctx, channelID)
	if err != nil {
		return ChannelDTO{}, ErrChannelNotFound
	}

	count, err := s.repo.CountMembers(ctx, channelID)
	if err != nil {
		return ChannelDTO{}, err
	}

	return toChannelDTO(record, count), nil
}

func (s *Service) Update(ctx context.Context, channelID, actorID uuid.UUID, req UpdateChannelRequest) (ChannelDTO, error) {
	if err := s.validate.Struct(req); err != nil {
		return ChannelDTO{}, err
	}

	role, err := s.repo.GetMemberRole(ctx, channelID, actorID)
	if err != nil {
		return ChannelDTO{}, ErrForbidden
	}
	if role != "owner" && role != "admin" {
		return ChannelDTO{}, ErrForbidden
	}

	record, err := s.repo.Update(ctx, channelID, req)
	if err != nil {
		return ChannelDTO{}, err
	}

	count, err := s.repo.CountMembers(ctx, channelID)
	if err != nil {
		return ChannelDTO{}, err
	}

	return toChannelDTO(record, count), nil
}

func (s *Service) Delete(ctx context.Context, channelID, actorID uuid.UUID) error {
	role, err := s.repo.GetMemberRole(ctx, channelID, actorID)
	if err != nil || role != "owner" {
		return ErrForbidden
	}

	return s.repo.Delete(ctx, channelID)
}

func (s *Service) GetMembers(ctx context.Context, channelID, actorID uuid.UUID) ([]ChannelMemberDTO, error) {
	isMember, err := s.repo.IsMember(ctx, channelID, actorID)
	if err != nil || !isMember {
		return nil, ErrForbidden
	}

	return s.repo.GetMembers(ctx, channelID)
}

func (s *Service) RemoveMember(ctx context.Context, channelID, actorID, targetUserID uuid.UUID) error {
	role, err := s.repo.GetMemberRole(ctx, channelID, actorID)
	if err != nil || (role != "owner" && role != "admin") {
		return ErrForbidden
	}

	return s.repo.RemoveMember(ctx, channelID, targetUserID)
}

func (s *Service) GenerateInviteCodeValue() (string, error) {
	raw := make([]byte, 8)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate invite code: %w", err)
	}

	return strings.TrimRight(base32.StdEncoding.EncodeToString(raw), "="), nil
}

func toChannelDTO(record channelRecord, memberCount int) ChannelDTO {
	return ChannelDTO{
		ID:          record.ID.String(),
		Type:        record.Type,
		Name:        record.Name,
		Description: record.Description,
		InviteCode:  record.InviteCode,
		OwnerID:     record.OwnerID,
		IsEncrypted: record.IsEncrypted,
		MaxMembers:  record.MaxMembers,
		MemberCount: memberCount,
		CreatedAt:   record.CreatedAt,
	}
}
