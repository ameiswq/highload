package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"PollsVoting/internal/handler"
	"PollsVoting/internal/repository"
	"PollsVoting/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("Using system environment variables")
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)

	db, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		cancel()
		log.Fatal(err)
	}
	defer db.Close()

	err = db.Ping(ctx)
	cancel()
	if err != nil {
		log.Fatal("Database connection failed: ", err)
	}

	log.Println("Connected to PostgreSQL")

	pollRepo := repository.NewPollRepository(db)
	pollService := service.NewPollService(pollRepo)
	pollHandler := handler.NewPollHandler(pollService)
	voteRepo := repository.NewVoteRepository(db)
	voteService := service.NewVoteService(voteRepo)
	voteHandler := handler.NewVoteHandler(voteService)
	router := gin.Default()

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})
	})
	router.GET("/health/protocol", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"protocol": c.Request.Proto,
		})
	})

	router.GET("/health/db", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(
			c.Request.Context(),
			2*time.Second,
		)
		defer cancel()

		if err := db.Ping(ctx); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"database": "disconnected",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"database": "connected",
		})
	})

	router.POST("/v1/polls", pollHandler.CreatePoll)
	router.GET("/v1/polls/:id", pollHandler.GetPoll)
	router.PATCH("/v1/polls/:id", pollHandler.ClosePoll)
	router.POST("/v1/polls/:id/votes", voteHandler.CastVote)
	router.GET("/v1/polls/:id/results", pollHandler.GetResults)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8443"
	}

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           router,
		ReadHeaderTimeout: 3 * time.Second,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      7 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Printf(
		"HTTPS REST API running on port %s",
		port,
	)

	err = server.ListenAndServeTLS(
		"certs/localhost.pem",
		"certs/localhost-key.pem",
	)

	if err != nil {
		log.Fatal(err)
	}

}
