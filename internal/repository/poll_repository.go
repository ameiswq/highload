package repository

import (
	"context"

	"PollsVoting/internal/model"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PollRepository struct {
	db *pgxpool.Pool
}

func NewPollRepository(db *pgxpool.Pool) *PollRepository {
	return &PollRepository{db: db}
}

func (r *PollRepository) GetByID(
	ctx context.Context,
	pollID int64,
) (*model.PollWithOptions, error) {

	var poll model.Poll

	err := r.db.QueryRow(ctx, `
        SELECT id, owner_id, question, status, allow_anonymous
        FROM polls
        WHERE id = $1
    `, pollID).Scan(
		&poll.ID,
		&poll.OwnerID,
		&poll.Question,
		&poll.Status,
		&poll.AllowAnonymous,
	)

	if err != nil {
		return nil, err
	}

	rows, err := r.db.Query(ctx, `
        SELECT id, poll_id, option_text, vote_count
        FROM poll_options
        WHERE poll_id = $1
        ORDER BY id
    `, pollID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	options := make([]model.PollOption, 0)

	for rows.Next() {
		var option model.PollOption

		err := rows.Scan(
			&option.ID,
			&option.PollID,
			&option.OptionText,
			&option.VoteCount,
		)
		if err != nil {
			return nil, err
		}

		options = append(options, option)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &model.PollWithOptions{
		Poll:    poll,
		Options: options,
	}, nil
}

func (r *PollRepository) Create(
	ctx context.Context,
	ownerID int64,
	question string,
	allowAnonymous bool,
	options []string,
) (*model.PollWithOptions, error) {

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var poll model.Poll

	err = tx.QueryRow(ctx, `
        INSERT INTO polls (
            owner_id,
            question,
            allow_anonymous
        )
        VALUES ($1, $2, $3)
        RETURNING
            id, owner_id, question,
            status, allow_anonymous
    `, ownerID, question, allowAnonymous).Scan(
		&poll.ID,
		&poll.OwnerID,
		&poll.Question,
		&poll.Status,
		&poll.AllowAnonymous,
	)

	if err != nil {
		return nil, err
	}

	result := &model.PollWithOptions{
		Poll:    poll,
		Options: make([]model.PollOption, 0, len(options)),
	}

	for _, text := range options {
		var option model.PollOption

		err = tx.QueryRow(ctx, `
            INSERT INTO poll_options (
                poll_id,
                option_text
            )
            VALUES ($1, $2)
            RETURNING
                id, poll_id, option_text, vote_count
        `, poll.ID, text).Scan(
			&option.ID,
			&option.PollID,
			&option.OptionText,
			&option.VoteCount,
		)

		if err != nil {
			return nil, err
		}

		result.Options = append(result.Options, option)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return result, nil
}

func (r *PollRepository) Close(
	ctx context.Context,
	pollID int64,
	ownerID int64,
) (*model.Poll, error) {

	var poll model.Poll

	err := r.db.QueryRow(ctx, `
        UPDATE polls
        SET status = 'closed'
        WHERE id = $1 AND owner_id = $2
        RETURNING
            id,
            owner_id,
            question,
            status,
            allow_anonymous
    `, pollID, ownerID).Scan(
		&poll.ID,
		&poll.OwnerID,
		&poll.Question,
		&poll.Status,
		&poll.AllowAnonymous,
	)

	if err != nil {
		return nil, err
	}

	return &poll, nil
}
