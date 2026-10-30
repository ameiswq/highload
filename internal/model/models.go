package model

type User struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
}

type Poll struct {
	ID             int64  `json:"id"`
	OwnerID        int64  `json:"owner_id"`
	Question       string `json:"question"`
	Status         string `json:"status"`
	AllowAnonymous bool   `json:"allow_anonymous"`
}

type PollOption struct {
	ID         int64  `json:"id"`
	PollID     int64  `json:"poll_id"`
	OptionText string `json:"option_text"`
	VoteCount  int    `json:"vote_count"`
}

type PollParticipation struct {
	PollID int64 `json:"poll_id"`
	UserID int64 `json:"user_id"`
}

type Vote struct {
	ID             int64  `json:"id"`
	PollID         int64  `json:"poll_id"`
	OptionID       int64  `json:"option_id"`
	UserID         int64  `json:"user_id"`
	IdempotencyKey string `json:"-"`
}

type PollWithOptions struct {
	Poll
	Options []PollOption `json:"options"`
}

type CastVoteResult struct {
	VoteID   int64  `json:"vote_id"`
	PollID   int64  `json:"poll_id"`
	OptionID int64  `json:"option_id"`
	Status   string `json:"status"`
	Replayed bool   `json:"replayed"`
}

type OptionResult struct {
	OptionID   int64   `json:"option_id"`
	OptionText string  `json:"option_text"`
	VoteCount  int     `json:"vote_count"`
	Percentage float64 `json:"percentage"`
}

type PollResults struct {
	PollID         int64          `json:"poll_id"`
	Question       string         `json:"question"`
	Status         string         `json:"status"`
	AllowAnonymous bool           `json:"allow_anonymous"`
	TotalVotes     int            `json:"total_votes"`
	Options        []OptionResult `json:"options"`
}
