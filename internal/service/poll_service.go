package service

import (
	"PollsVoting/internal/model"
	"PollsVoting/internal/repository"
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
)

type PollService struct {
	repo *repository.PollRepository
}

func NewPollService(repo *repository.PollRepository) *PollService {
	return &PollService{repo: repo}
}

func (s *PollService) GetPoll(
	ctx context.Context,
	id int64,
) (*model.PollWithOptions, error) {
	return s.repo.GetByID(ctx, id)
}

var ErrInvalidPoll = errors.New("invalid poll data")
var ErrNotPollOwner = errors.New("user is not the poll owner")

func (s *PollService) CreatePoll(
	ctx context.Context,
	ownerID int64,
	question string,
	allowAnonymous bool,
	options []string,
) (*model.PollWithOptions, error) {

	// Validate owner
	if ownerID <= 0 {
		return nil, fmt.Errorf(
			"%w: invalid owner ID", ErrInvalidPoll,
		)
	}

	// Validate question
	question = strings.TrimSpace(question)

	if question == "" {
		return nil, fmt.Errorf(
			"%w: question cannot be empty", ErrInvalidPoll,
		)
	}

	// Validate number of options
	if len(options) < 2 || len(options) > 10 {
		return nil, fmt.Errorf(
			"%w: provide 2 to 10 options", ErrInvalidPoll,
		)
	}

	// Validate each option
	for i, option := range options {
		options[i] = strings.TrimSpace(option)

		if options[i] == "" {
			return nil, fmt.Errorf(
				"%w: option cannot be empty", ErrInvalidPoll,
			)
		}
	}

	// Save to database
	return s.repo.Create(
		ctx,
		ownerID,
		question,
		allowAnonymous,
		options,
	)
}

func (s *PollService) ClosePoll(
	ctx context.Context,
	pollID int64,
	ownerID int64,
) (*model.Poll, error) {

	// Validate IDs
	if pollID <= 0 || ownerID <= 0 {
		return nil, fmt.Errorf(
			"%w: invalid poll or owner ID",
			ErrInvalidPoll,
		)
	}

	// Check if poll exists
	poll, err := s.repo.GetByID(ctx, pollID)
	if err != nil {
		return nil, err
	}

	// Check ownership
	if poll.OwnerID != ownerID {
		return nil, ErrNotPollOwner
	}

	// Close poll
	return s.repo.Close(ctx, pollID, ownerID)
}

func (s *PollService) GetResults(
	ctx context.Context,
	pollID int64,
) (*model.PollResults, error) {

	if pollID <= 0 {
		return nil, ErrInvalidPoll
	}

	// Get poll with its options from database
	poll, err := s.repo.GetByID(ctx, pollID)
	if err != nil {
		return nil, err
	}

	results := &model.PollResults{
		PollID:         poll.ID,
		Question:       poll.Question,
		Status:         poll.Status,
		AllowAnonymous: poll.AllowAnonymous,
		Options:        make([]model.OptionResult, 0),
	}

	// Calculate total votes
	totalVotes := 0

	for _, option := range poll.Options {
		totalVotes += option.VoteCount
	}

	results.TotalVotes = totalVotes

	// Calculate results for each option
	for _, option := range poll.Options {
		percentage := 0.0

		if totalVotes > 0 {
			percentage = float64(option.VoteCount) /
				float64(totalVotes) * 100

			// Round to 2 decimal places
			percentage = math.Round(percentage*100) / 100
		}

		results.Options = append(
			results.Options,
			model.OptionResult{
				OptionID:   option.ID,
				OptionText: option.OptionText,
				VoteCount:  option.VoteCount,
				Percentage: percentage,
			},
		)
	}

	return results, nil
}
