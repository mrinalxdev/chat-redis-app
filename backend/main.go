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

// clients for redis and message broker
var (
	redisClient *redis.Client
	rabbitConn  *amqp.Connection
	rabbitCh    *amqp.Channel
	ctx         = context.Background()
)

func init() {
	// starting the redis
	redisClient = redis.NewClient(&redis.Options{
		Addr: "redis:6379",
	})
	if _, err := redisClient.Ping(ctx).Result(); err != nil {
		log.Fatalf("Failed to connect to Redis : %v", err)
	}

	// initializing message broker
	var err error
	rabbitConn, err = amqp.Dial("amqp://guest:guest@rabbitmq:5672/")
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ : %v", err)
	}
	rabbitCh, err = rabbitConn.Channel()
	if err != nil {
		log.Fatalf("Failed to open RabbitMQ channel : %v", err)
	}
}

func main () {
	r := gin.Default()

	r.POST("/start-session", startSession)
	r.POST("/offer", handleOffer)
	r.GET("/session/:id", getSession)
}

func startSession(c *gin.Context) {
	sessionID := "session_"  + generateID()
	if err := redisClient.Set(ctx, sessionID , "active", 0).Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error" : "Failed to create session"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"session_id" : sessionID})
}


// func handleSignal(c *gin.Context){
// 	var msg struct {
// 		SessionID string `json:"session_id"`
// 		Message string `json:"message"`
// 	}
// 	if err := c.ShouldBindJSON(&msg); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error" : "Invalid request"})
// 		return
// 	}

// 	err := rabbitCh.Publish(
// 		"",
// 		msg.SessionID, 
// 		false,
// 		false,
// 		amqp.Publishing{
// 			ContentType: "text/plain",
// 			Body: []byte(msg.Message),
// 		},
// 	)

// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error" : "Failed to send message"})
// 		return
// 	}
// }

// Now changing it to handle SDP offers to answer the SDP asnwers
func handleOffer(c *gin.Context) {
	var request struct {
		Offer string `json:"offer"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error" : "Invalid request"})
		return 
	}

	peerConnection, err := webrtc.NewPeerConnection(webrtc.Configuration{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error" : "Failed to create peer connection"})
		return
	}

	peerConnection.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate != nil {
			log.Printf("New ICE candidate : %s\n", candidate.ToJSON().Candidate)
		}
	})

	// setting the remote sdp (offer) from the frontend
	offer := webrtc.SessionDescription{
		Type: webrtc.SDPTypeOffer,
		SDP : request.Offer,
	}

	if err := peerConnection.SetRemoteDescription(offer); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error" : "Failed to set remote description"})
		return
	}

	// creating a sdp answer
	answer, err := peerConnection.CreateAnswer(nil)
	if err != nil {
		// TODO : Logic
	}

}

func getSession(c *gin.Context){
	sessionId := c.Param("id")
	status, err := redisClient.Get(ctx, sessionId).Result()
	if err == redis.Nil {
		c.JSON(http.StatusNotFound, gin.H{"error" : "Session not found"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error" : "Failed to retrieve session"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"session_id" : sessionId, "status" : status})
}

func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}