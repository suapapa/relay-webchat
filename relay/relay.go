package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

func (r *Relay) ChatHandler(c *gin.Context) {
	var req ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if r.processorConn == nil {
		c.JSON(http.StatusOK, ChatResponse{
			Reply: "No processor connection",
		})
		return
	}

	msg := &Msg{
		SrcIP:    c.ClientIP(),
		ClientID: "123456789",
		MsgID:    uuid.New().String(),
		Content:  req.Message,
		MsgTS:    time.Now(),
		ReplyCh:  make(chan string),
	}
	r.msgChan <- msg

	ctx := c.Request.Context()
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	select {
	case reply := <-msg.ReplyCh:
		c.JSON(http.StatusOK, ChatResponse{
			Reply: "Echo: " + reply,
		})
	case <-ctx.Done():
		c.JSON(http.StatusRequestTimeout, ChatResponse{
			Reply: "Request timeout",
		})
	}

	msg.Close()
}

type ChatRequest struct {
	Message string `json:"message"`
}

type ChatResponse struct {
	Reply string `json:"reply"`
}

func (r *Relay) handleWebSocket(c *gin.Context) {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			log.Println("WARNING: origin check disabled")
			log.Printf("origin: %s", r.RemoteAddr)
			return true
		},
	}

	var err error
	r.processorConn, err = upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	log.Printf("New WebSocket connection from %s", r.processorConn.RemoteAddr())
	defer func() {
		log.Printf("Closing WebSocket connection from %s", r.processorConn.RemoteAddr())
		r.processorConn.Close()
		r.processorConn = nil
	}()

	log.Printf("New WebSocket connection from %s", r.processorConn.RemoteAddr())

	for msg := range r.msgChan {
		if err := r.processorConn.WriteMessage(websocket.TextMessage, []byte(msg.Content)); err != nil {
			log.Printf("Write error: %v", err)
			break
		}

		replyType, reply, err := r.processorConn.ReadMessage()
		if err != nil {
			log.Printf("Read error: %v", err)
			break
		}

		if replyType == websocket.TextMessage {
			msg.ReplyCh <- string(reply)
		}
	}
}

type Relay struct {
	processorConn *websocket.Conn
	msgChan       chan *Msg
}

func NewRelay() *Relay {
	return &Relay{
		msgChan: make(chan *Msg, 100),
	}
}

type Msg struct {
	SrcIP          string
	ClientID       string
	MsgID          string
	Content        string
	ReplyCh        chan string
	MsgTS, ReplyTS time.Time
}

func (m *Msg) Close() {
	close(m.ReplyCh)
}
