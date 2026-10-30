package handler

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"PollsVoting/internal/repository"
	"PollsVoting/internal/service"

	"github.com/gin-gonic/gin"
)

type VoteHandler struct {
	service *service.VoteService
}

func NewVoteHandler(
	s *service.VoteService,
) *VoteHandler {
	return &VoteHandler{service: s}
}

type CastVoteRequest struct {
	UserID   int64 `json:"user_id"`
	OptionID int64 `json:"option_id"`
}

func (h *VoteHandler) CastVote(c *gin.Context) {

	pollID, err := strconv.ParseInt(
		c.Param("id"), 10, 64,
	)
	if err != nil || pollID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid poll id",
		})
		return
	}

	var req CastVoteRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid JSON",
		})
		return
	}

	key := c.GetHeader("Idempotency-Key")

	ctx, cancel := context.WithTimeout(
		c.Request.Context(),
		5*time.Second,
	)
	defer cancel()

	result, err := h.service.CastVote(
		ctx,
		pollID,
		req.UserID,
		req.OptionID,
		key,
	)

	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidVote):
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid vote data or idempotency key",
			})

		case errors.Is(err, repository.ErrVotePollNotFound):
			c.JSON(http.StatusNotFound, gin.H{
				"error": "poll not found",
			})

		case errors.Is(err, repository.ErrVoteUserNotFound):
			c.JSON(http.StatusNotFound, gin.H{
				"error": "user not found",
			})

		case errors.Is(err, repository.ErrOptionNotFound):
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid option for this poll",
			})

		case errors.Is(err, repository.ErrPollClosed):
			c.JSON(http.StatusConflict, gin.H{
				"error": "poll is closed",
			})

		case errors.Is(err, repository.ErrAlreadyVoted):
			c.JSON(http.StatusConflict, gin.H{
				"error": "user has already voted",
			})

		case errors.Is(err, repository.ErrIdempotencyConflict):
			c.JSON(http.StatusConflict, gin.H{
				"error": "idempotency key already used",
			})

		case errors.Is(err, context.DeadlineExceeded):
			c.JSON(http.StatusGatewayTimeout, gin.H{
				"error": "request timeout",
			})

		default:
			log.Printf("CastVote failed: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "internal server error",
			})
		}
		return
	}

	status := http.StatusCreated

	if result.Replayed {
		status = http.StatusOK
	}

	c.JSON(status, result)
}
