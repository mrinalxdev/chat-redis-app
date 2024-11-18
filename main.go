package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/websocket"
	"github.com/streadway/amqp"
)

type ChatMessage struct {
	Username string `json:"username"`
	Content string `json:"content"`
	Room string `json:"room"`
	Timestamp time.Time `json:"timestamp"`
}

type Client struct {
	conn *websocket.Conn
	username string
	room string
}

var (
	redisClient = redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	upgrader = websocket.Upgrader{
		ReadBufferSize: 1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	// rabbitmq connection
	rabbitConn, _ = amqp.Dial("amqp://guest:guest@localhost:5672/")
	rabbitChan, _ = rabbitConn.Channel()
)

func main() {
    err := rabbitChan.ExchangeDeclare(
        "chat_exchange", 
        "fanout",     
        true,           
        false,          
        false,         
        false,          
        nil,           
    )
    if err != nil {
        log.Fatal(err)
    }

    http.HandleFunc("/ws", handleWebSocket)
    http.HandleFunc("/history", getChatHistory)
    http.Handle("/", http.FileServer(http.Dir("static")))

    log.Println("Server starting on :8080")
    log.Fatal(http.ListenAndServe(":8080", nil))
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        log.Println(err)
        return
    }

    username := r.URL.Query().Get("username")
    room := r.URL.Query().Get("room")
    
    client := &Client{
        conn:     conn,
        username: username,
        room:     room,
    }

    q, err := rabbitChan.QueueDeclare(
        "",   
        false, 
        true,  
        true,  
        false, 
        nil,   
    )
    if err != nil {
        log.Println(err)
        return
    }

    err = rabbitChan.QueueBind(
        q.Name,         
        "",            
        "chat_exchange", 
        false,
        nil,
    )
    if err != nil {
        log.Println(err)
        return
    }

    go handleMessages(client, q.Name)

    for {
        _, msg, err := conn.ReadMessage()
        if err != nil {
            log.Println(err)
            return
        }

        chatMsg := ChatMessage{
            Username:  client.username,
            Content:   string(msg),
            Room:     client.room,
            Timestamp: time.Now(),
        }

        // Storing in Redis
        msgJSON, _ := json.Marshal(chatMsg)
        err = redisClient.LPush(r.Context(), "chat_history_"+client.room, msgJSON).Err()
        if err != nil {
            log.Println(err)
        }

        // Publish to RabbitMQ
        err = rabbitChan.Publish(
            "chat_exchange",
            "",            
            false,         
            false,         
            amqp.Publishing{
                ContentType: "application/json",
                Body:        msgJSON,
            },
        )
        if err != nil {
            log.Println(err)
        }
    }
}

func handleMessages(client *Client, queueName string) {
    msgs, err := rabbitChan.Consume(
        queueName,
        "",       
        true,     
        false,    
        false,    
        false,   
        nil,      
    )
    if err != nil {
        log.Println(err)
        return
    }

    for msg := range msgs {
        err = client.conn.WriteMessage(websocket.TextMessage, msg.Body)
        if err != nil {
            log.Println(err)
            return
        }
    }
}

func getChatHistory(w http.ResponseWriter, r *http.Request) {
    room := r.URL.Query().Get("room")
    messages, err := redisClient.LRange(r.Context(), "chat_history_"+room, 0, 49).Result()
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(messages)
}