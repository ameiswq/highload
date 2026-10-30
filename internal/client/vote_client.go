package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type VotePayload struct {
	UserID   int64 `json:"user_id"`
	OptionID int64 `json:"option_id"`
}

func shouldRetry(status int) bool {
	switch status {
	case 429, 502, 503, 504:
		return true
	default:
		return false
	}
}

func waitBeforeRetry(
	ctx context.Context,
	attempt int,
) error {
	delay := time.Duration(1<<(attempt-1)) *
		100 * time.Millisecond

	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func SendVoteWithRetry(
	ctx context.Context,
	httpClient *http.Client,
	url string,
	key string,
	payload VotePayload,
) (int, []byte, error) {

	body, err := json.Marshal(payload)
	if err != nil {
		return 0, nil, err
	}

	const maxAttempts = 3

	for attempt := 1; attempt <= maxAttempts; attempt++ {

		req, err := http.NewRequestWithContext(
			ctx,
			http.MethodPost,
			url,
			bytes.NewReader(body),
		)
		if err != nil {
			return 0, nil, err
		}

		req.Header.Set(
			"Content-Type",
			"application/json",
		)
		req.Header.Set(
			"Idempotency-Key",
			key,
		)

		resp, err := httpClient.Do(req)

		if err == nil {
			data, readErr := io.ReadAll(resp.Body)
			resp.Body.Close()

			if readErr != nil {
				return 0, nil, readErr
			}

			if !shouldRetry(resp.StatusCode) ||
				attempt == maxAttempts {

				return resp.StatusCode, data, nil
			}

			fmt.Printf(
				"Attempt %d: HTTP %d, retrying...\n",
				attempt,
				resp.StatusCode,
			)

		} else {
			if ctx.Err() != nil {
				return 0, nil, ctx.Err()
			}

			if attempt == maxAttempts {
				return 0, nil, err
			}

			fmt.Printf(
				"Attempt %d failed: %v\n",
				attempt,
				err,
			)
		}

		if err := waitBeforeRetry(ctx, attempt); err != nil {
			return 0, nil, err
		}
	}

	return 0, nil, fmt.Errorf("all attempts failed")
}
