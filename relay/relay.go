package main

import (
	"context"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = 25 * time.Second
	replyWait  = 30 * time.Second
)

type wsConn struct {
	conn      *websocket.Conn
	writeMu   sync.Mutex
	replyChan chan string
	done      chan struct{}
	closeOnce sync.Once
}

func newWSConn(conn *websocket.Conn) *wsConn {
	return &wsConn{
		conn:      conn,
		replyChan: make(chan string, 1),
		done:      make(chan struct{}),
	}
}

func (w *wsConn) WriteMessage(messageType int, data []byte) error {
	w.writeMu.Lock()
	defer w.writeMu.Unlock()
	if err := w.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
		return err
	}
	return w.conn.WriteMessage(messageType, data)
}

func (w *wsConn) WritePing() error {
	w.writeMu.Lock()
	defer w.writeMu.Unlock()
	if err := w.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
		return err
	}
	return w.conn.WriteMessage(websocket.PingMessage, nil)
}

func (w *wsConn) Close() {
	w.closeOnce.Do(func() {
		close(w.done)
		w.conn.Close()
	})
}

func (r *Relay) ChatHandler(c *gin.Context) {
	var req ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if r.getProcessor() == nil {
		c.JSON(http.StatusOK, ChatResponse{
			Reply: "No processor connection",
		})
		return
	}

	if strings.TrimSpace(req.Message) == "" {
		c.JSON(http.StatusOK, ChatResponse{
			Reply: "메시지를 입력해주세요.",
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), replyWait)
	defer cancel()

	msg := &Msg{
		SrcIP:    c.ClientIP(),
		ClientID: "123456789",
		MsgID:    uuid.New().String(),
		Content:  req.Message,
		MsgTS:    time.Now(),
		ReplyCh:  make(chan string, 1),
		Ctx:      ctx,
	}

	select {
	case r.msgChan <- msg:
	case <-ctx.Done():
		c.JSON(http.StatusRequestTimeout, ChatResponse{
			Reply: "Apologies, I am unable to complete the request.",
		})
		return
	}

	select {
	case reply := <-msg.ReplyCh:
		c.JSON(http.StatusOK, ChatResponse{
			Reply: reply,
		})
	case <-ctx.Done():
		c.JSON(http.StatusRequestTimeout, ChatResponse{
			Reply: "Apologies, I am unable to complete the request.",
		})
	}
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

	rawConn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	client := newWSConn(rawConn)
	log.Printf("New WebSocket connection from %s", rawConn.RemoteAddr())

	rawConn.SetReadDeadline(time.Now().Add(pongWait))
	rawConn.SetPongHandler(func(appData string) error {
		log.Println("Received pong from processor")
		rawConn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
	defaultPingHandler := rawConn.PingHandler()
	rawConn.SetPingHandler(func(appData string) error {
		rawConn.SetReadDeadline(time.Now().Add(pongWait))
		return defaultPingHandler(appData)
	})

	r.mu.Lock()
	if r.processorConn != nil {
		log.Printf("Replacing existing WebSocket connection from %s", r.processorConn.conn.RemoteAddr())
		r.processorConn.Close()
	}
	r.processorConn = client
	r.mu.Unlock()

	defer func() {
		log.Printf("Closing WebSocket connection from %s", rawConn.RemoteAddr())
		client.Close()
		r.mu.Lock()
		if r.processorConn == client {
			r.processorConn = nil
		}
		r.mu.Unlock()
	}()

	// Continuously read from the WebSocket connection so control frames (Ping/Pong/Close)
	// and processor disconnects are processed even when no chat messages are queued.
	go func() {
		defer client.Close()
		for {
			replyType, reply, err := rawConn.ReadMessage()
			if err != nil {
				log.Printf("Read error: %v", err)
				return
			}
			rawConn.SetReadDeadline(time.Now().Add(pongWait))

			var text string
			if replyType == websocket.TextMessage {
				text = string(reply)
			} else {
				text = "Invalid response format from processor"
			}

			select {
			case client.replyChan <- text:
			default:
				select {
				case <-client.replyChan:
				default:
				}
				client.replyChan <- text
			}
		}
	}()

	pingTicker := time.NewTicker(pingPeriod)
	defer pingTicker.Stop()

	for {
		select {
		case <-client.done:
			return
		case <-pingTicker.C:
			if err := client.WritePing(); err != nil {
				log.Printf("Ping error: %v", err)
				return
			}
		case msg := <-r.msgChan:
			if msg.Ctx != nil && msg.Ctx.Err() != nil {
				continue
			}

			// Drain any stale reply from a previous timed-out message
			select {
			case <-client.replyChan:
			default:
			}

			if err := client.WriteMessage(websocket.TextMessage, []byte(msg.Content)); err != nil {
				log.Printf("Write error: %v", err)
				msg.Reply("Failed to send message to processor")
				return
			}

			replyTimer := time.NewTimer(replyWait)
			waitingReply := true
			for waitingReply {
				select {
				case reply := <-client.replyChan:
					waitingReply = false
					msg.Reply(reply)
				case <-pingTicker.C:
					if err := client.WritePing(); err != nil {
						log.Printf("Ping error: %v", err)
						replyTimer.Stop()
						msg.Reply("Failed to receive response from processor")
						return
					}
				case <-client.done:
					replyTimer.Stop()
					msg.Reply("Failed to receive response from processor")
					return
				case <-replyTimer.C:
					log.Printf("Timeout waiting for processor response")
					msg.Reply("Failed to receive response from processor")
					return
				}
			}
			replyTimer.Stop()
		}
	}
}

type Relay struct {
	mu            sync.RWMutex
	processorConn *wsConn
	msgChan       chan *Msg
}

func NewRelay() *Relay {
	return &Relay{
		msgChan: make(chan *Msg, 100),
	}
}

func (r *Relay) getProcessor() *wsConn {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.processorConn
}

type Msg struct {
	SrcIP          string
	ClientID       string
	MsgID          string
	Content        string
	ReplyCh        chan string
	Ctx            context.Context
	MsgTS, ReplyTS time.Time
}

func (m *Msg) Reply(reply string) {
	select {
	case m.ReplyCh <- reply:
	default:
	}
}

func (m *Msg) Close() {
}
