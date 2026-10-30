package service

import (
	"context"
	"errors"
	"strings"

	"PollsVoting/internal/model"
	"PollsVoting/internal/repository"

	"github.com/google/uuid"
)

var ErrInvalidVote = errors.New("invalid vote request")

type VoteService struct {
	repo *repository.VoteRepository
}

func NewVoteService(
	repo *repository.VoteRepository,
) *VoteService {
	return &VoteService{repo: repo}
}

func (s *VoteService) CastVote(
	ctx context.Context,
	pollID int64,
	userID int64,
	optionID int64,
	idempotencyKey string,
) (*model.CastVoteResult, error) {

	// Validate IDs
	if pollID <= 0 || userID <= 0 || optionID <= 0 {
		return nil, ErrInvalidVote
	}

	// Validate idempotency key
	idempotencyKey = strings.TrimSpace(idempotencyKey)

	parsedKey, err := uuid.Parse(idempotencyKey)
	if err != nil {
		return nil, ErrInvalidVote
	}

	// Call repository
	return s.repo.CastVote(
		ctx,
		pollID,
		userID,
		optionID,
		parsedKey.String(),
	)
}
