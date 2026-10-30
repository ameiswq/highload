package repository

import (
	"context"
	"errors"
	"fmt"

	"PollsVoting/internal/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrVotePollNotFound    = errors.New("poll not found")
	ErrPollClosed          = errors.New("poll is closed")
	ErrOptionNotFound      = errors.New("option not found")
	ErrVoteUserNotFound    = errors.New("user not found")
	ErrAlreadyVoted        = errors.New("user already voted")
	ErrIdempotencyConflict = errors.New("idempotency key conflict")
)

type VoteRepository struct {
	db *pgxpool.Pool
}

func NewVoteRepository(db *pgxpool.Pool) *VoteRepository {
	return &VoteRepository{db: db}
}

func findVoteByKey(
	ctx context.Context,
	tx pgx.Tx,
	key string,
) (*model.Vote, error) {
	var vote model.Vote

	err := tx.QueryRow(ctx, `
        SELECT id, poll_id, user_id, option_id
        FROM votes
        WHERE idempotency_key = $1
    `, key).Scan(
		&vote.ID,
		&vote.PollID,
		&vote.UserID,
		&vote.OptionID,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &vote, nil
}

func voteReplay(
	vote *model.Vote,
	pollID, userID, optionID int64,
) (*model.CastVoteResult, error) {
	if vote.PollID != pollID ||
		vote.UserID != userID ||
		vote.OptionID != optionID {
		return nil, ErrIdempotencyConflict
	}

	return &model.CastVoteResult{
		VoteID:   vote.ID,
		PollID:   vote.PollID,
		OptionID: vote.OptionID,
		Status:   "accepted",
		Replayed: true,
	}, nil
}

func (r *VoteRepository) CastVote(
	ctx context.Context,
	pollID int64,
	userID int64,
	optionID int64,
	key string,
) (*model.CastVoteResult, error) {

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var status string

	err = tx.QueryRow(ctx, `
        SELECT status
        FROM polls
        WHERE id = $1
        FOR SHARE
    `, pollID).Scan(&status)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrVotePollNotFound
	}
	if err != nil {
		return nil, err
	}

	existing, err := findVoteByKey(ctx, tx, key)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return voteReplay(
			existing, pollID, userID, optionID,
		)
	}

	if status != "open" {
		return nil, ErrPollClosed
	}

	var userExists bool

	err = tx.QueryRow(ctx, `
        SELECT EXISTS (
            SELECT 1 FROM users WHERE id = $1
        )
    `, userID).Scan(&userExists)

	if err != nil {
		return nil, err
	}
	if !userExists {
		return nil, ErrVoteUserNotFound
	}

	var optionExists bool

	err = tx.QueryRow(ctx, `
        SELECT EXISTS (
            SELECT 1 FROM poll_options
            WHERE id = $1 AND poll_id = $2
        )
    `, optionID, pollID).Scan(&optionExists)

	if err != nil {
		return nil, err
	}
	if !optionExists {
		return nil, ErrOptionNotFound
	}

	var insertedUserID int64

	err = tx.QueryRow(ctx, `
        INSERT INTO poll_participations (
            poll_id, user_id
        )
        VALUES ($1, $2)
        ON CONFLICT (poll_id, user_id)
        DO NOTHING
        RETURNING user_id
    `, pollID, userID).Scan(&insertedUserID)

	if errors.Is(err, pgx.ErrNoRows) {
		existing, lookupErr := findVoteByKey(
			ctx, tx, key,
		)
		if lookupErr != nil {
			return nil, lookupErr
		}
		if existing != nil {
			return voteReplay(
				existing, pollID, userID, optionID,
			)
		}

		return nil, ErrAlreadyVoted
	}
	if err != nil {
		return nil, err
	}

	var voteID int64

	err = tx.QueryRow(ctx, `
        INSERT INTO votes (
            poll_id,
            user_id,
            option_id,
            idempotency_key
        )
        VALUES ($1, $2, $3, $4)
        ON CONFLICT (idempotency_key)
        DO NOTHING
        RETURNING id
    `, pollID, userID, optionID, key).Scan(&voteID)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrIdempotencyConflict
	}
	if err != nil {
		return nil, err
	}

	tag, err := tx.Exec(ctx, `
        UPDATE poll_options
        SET vote_count = vote_count + 1
        WHERE id = $1 AND poll_id = $2
    `, optionID, pollID)

	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() != 1 {
		return nil, fmt.Errorf(
			"vote counter update failed",
		)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &model.CastVoteResult{
		VoteID:   voteID,
		PollID:   pollID,
		OptionID: optionID,
		Status:   "accepted",
		Replayed: false,
	}, nil
}
