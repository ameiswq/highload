package handler

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"PollsVoting/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type PollHandler struct {
	service *service.PollService
}

func NewPollHandler(s *service.PollService) *PollHandler {
	return &PollHandler{service: s}
}

func (h *PollHandler) GetPoll(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)

	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid poll id",
		})
		return
	}

	poll, err := h.service.GetPoll(c.Request.Context(), id)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "poll not found",
			})
			return
		}

		log.Printf("GetPoll failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "internal server error",
		})
		return
	}

	c.JSON(http.StatusOK, poll)
}

type CreatePollRequest struct {
	OwnerID        int64    `json:"owner_id"`
	Question       string   `json:"question"`
	AllowAnonymous *bool    `json:"allow_anonymous"`
	Options        []string `json:"options"`
}

func (h *PollHandler) CreatePoll(c *gin.Context) {
	var req CreatePollRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid JSON request",
		})
		return
	}

	if req.AllowAnonymous == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "allow_anonymous is required",
		})
		return
	}

	ctx, cancel := context.WithTimeout(
		c.Request.Context(),
		5*time.Second,
	)
	defer cancel()

	poll, err := h.service.CreatePoll(
		ctx,
		req.OwnerID,
		req.Question,
		*req.AllowAnonymous,
		req.Options,
	)

	if err != nil {
		if errors.Is(err, service.ErrInvalidPoll) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}

		var pgErr *pgconn.PgError

		if errors.As(err, &pgErr) &&
			pgErr.Code == "23503" &&
			pgErr.ConstraintName == "fk_poll_owner" {

			c.JSON(http.StatusBadRequest, gin.H{
				"error": "owner does not exist",
			})
			return
		}

		log.Printf("CreatePoll failed: %v", err)

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "internal server error",
		})
		return
	}
	c.JSON(http.StatusCreated, poll)
}

type ClosePollRequest struct {
	OwnerID int64  `json:"owner_id"`
	Status  string `json:"status"`
}

func (h *PollHandler) ClosePoll(c *gin.Context) {

	pollID, err := strconv.ParseInt(
		c.Param("id"), 10, 64,
	)

	if err != nil || pollID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid poll id",
		})
		return
	}

	var req ClosePollRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid JSON request",
		})
		return
	}

	if req.Status != "closed" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "status must be closed",
		})
		return
	}

	ctx, cancel := context.WithTimeout(
		c.Request.Context(),
		5*time.Second,
	)
	defer cancel()

	poll, err := h.service.ClosePoll(
		ctx,
		pollID,
		req.OwnerID,
	)

	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidPoll):
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})

		case errors.Is(err, service.ErrNotPollOwner):
			c.JSON(http.StatusForbidden, gin.H{
				"error": "only the owner can close this poll",
			})

		case errors.Is(err, pgx.ErrNoRows):
			c.JSON(http.StatusNotFound, gin.H{
				"error": "poll not found",
			})

		default:
			log.Printf("ClosePoll failed: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "internal server error",
			})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":     poll.ID,
		"status": poll.Status,
	})
}

func (h *PollHandler) GetResults(c *gin.Context) {

	pollID, err := strconv.ParseInt(
		c.Param("id"), 10, 64,
	)

	if err != nil || pollID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid poll id",
		})
		return
	}

	ctx, cancel := context.WithTimeout(
		c.Request.Context(),
		2*time.Second,
	)
	defer cancel()

	results, err := h.service.GetResults(ctx, pollID)

	if err != nil {
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			c.JSON(http.StatusNotFound, gin.H{
				"error": "poll not found",
			})

		case errors.Is(err, context.DeadlineExceeded):
			c.JSON(http.StatusGatewayTimeout, gin.H{
				"error": "request timeout",
			})

		default:
			log.Printf("GetResults failed: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "internal server error",
			})
		}
		return
	}
	c.JSON(http.StatusOK, results)
}
