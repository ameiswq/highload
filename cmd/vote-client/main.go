package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"PollsVoting/internal/client"

	"github.com/google/uuid"
)

func main() {
	pollID := flag.Int64("poll", 1, "Poll ID")
	userID := flag.Int64("user", 1, "User ID")
	optionID := flag.Int64("option", 1, "Option ID")
	flag.Parse()

	url := fmt.Sprintf(
		"http://localhost:8080/v1/polls/%d/votes",
		*pollID,
	)

	key := uuid.NewString()

	httpClient := &http.Client{
		Timeout: 2 * time.Second,
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		8*time.Second,
	)
	defer cancel()

	status, body, err := client.SendVoteWithRetry(
		ctx,
		httpClient,
		url,
		key,
		client.VotePayload{
			UserID:   *userID,
			OptionID: *optionID,
		},
	)

	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("HTTP status:", status)
	fmt.Println("Response:", string(body))
	fmt.Println("Idempotency-Key:", key)
}
