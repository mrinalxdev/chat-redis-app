package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pion/webrtc/v3"
	"github.com/redis/go-redis/v9"
	"github.com/streadway/amqp"
)

var (
	redisClient *redis.Client
	rabbitConn  *amqp.Connection
	rabbitCh    *amqp.Channel
	ctx         = context.Background()
)

func init() {
	// Initialize Redis
	redisClient = redis.NewClient(&redis.Options{
		Addr: "redis:6379",
	})
	if _, err := redisClient.Ping(ctx).Result(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	// Initialize RabbitMQ
	var err error
	rabbitConn, err = amqp.Dial("amqp://guest:guest@rabbitmq:5672/")
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	rabbitCh, err = rabbitConn.Channel()
	if err != nil {
		log.Fatalf("Failed to open RabbitMQ channel: %v", err)
	}
}

func main() {
	r := gin.Default()

	// API Endpoints
	r.POST("/start-session", startSession)
	r.POST("/offer", handleOffer)
	r.GET("/session/:id", getSession)

	// Start the server
	log.Println("Server running on port 8080...")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func startSession(c *gin.Context) {
	sessionID := "session_" + generateID()
	if err := redisClient.Set(ctx, sessionID, "active", 0).Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create session"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"session_id": sessionID})
}

func handleOffer(c *gin.Context) {
	var request struct {
		Offer     string `json:"offer"`
		SessionID string `json:"session_id"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Ensure session exists in Redis
	sessionExists, err := redisClient.Exists(ctx, request.SessionID).Result()
	if err != nil || sessionExists == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
		return
	}

	// Create a WebRTC PeerConnection
	peerConnection, err := webrtc.NewPeerConnection(webrtc.Configuration{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create peer connection"})
		return
	}

	// Handle ICE candidates
	peerConnection.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate != nil {
			log.Printf("New ICE candidate: %s\n", candidate.ToJSON().Candidate)
		}
	})

	// Set the remote description (SDP offer)
	offer := webrtc.SessionDescription{
		Type: webrtc.SDPTypeOffer,
		SDP:  request.Offer,
	}
	if err := peerConnection.SetRemoteDescription(offer); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to set remote description"})
		return
	}

	// Create and set a local SDP answer
	answer, err := peerConnection.CreateAnswer(nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create answer"})
		return
	}
	if err := peerConnection.SetLocalDescription(answer); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to set local description"})
		return
	}

	// Publish the SDP answer to RabbitMQ for signaling
	err = rabbitCh.Publish(
		"", // Default exchange
		request.SessionID,
		false,
		false,
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(answer.SDP),
		},
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send answer via RabbitMQ"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"answer": answer.SDP})
}

func getSession(c *gin.Context) {
	sessionID := c.Param("id")
	status, err := redisClient.Get(ctx, sessionID).Result()
	if err == redis.Nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve session"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"session_id": sessionID, "status": status})
}

func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
