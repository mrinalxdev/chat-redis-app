package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
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
	r.POST("/signal", handleSignal)
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


func handleSignal(c *gin.Context){
	var msg struct {
		SessionID string `json:"session_id"`
		Message string `json:"message"`
	}
	if err := c.ShouldBindJSON(&msg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error" : "Invalid request"})
		return
	}

	err := rabbitCh.Publish(
		"",
		msg.SessionID, 
		false,
		false,
		amqp.Publishing{
			ContentType: "text/plain",
			Body: []byte(msg.Message),
		},
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error" : "Failed to send message"})
		return
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