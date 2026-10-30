package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSendVoteWithRetry(t *testing.T) {
	attempts := 0
	const key = "550e8400-e29b-41d4-a716-446655440000"

	server := httptest.NewServer(
		http.HandlerFunc(func(
			w http.ResponseWriter,
			r *http.Request,
		) {
			attempts++

			if r.Header.Get("Idempotency-Key") != key {
				t.Error("idempotency key changed")
			}

			if attempts < 3 {
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}

			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"status":"accepted"}`))
		}),
	)
	defer server.Close()

	ctx, cancel := context.WithTimeout(
		context.Background(),
		3*time.Second,
	)
	defer cancel()

	status, _, err := SendVoteWithRetry(
		ctx,
		server.Client(),
		server.URL,
		key,
		VotePayload{
			UserID:   1,
			OptionID: 2,
		},
	)

	if err != nil {
		t.Fatal(err)
	}

	if status != http.StatusCreated {
		t.Errorf("expected 201, got %d", status)
	}

	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}
