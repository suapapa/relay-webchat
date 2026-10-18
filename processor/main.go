package main

import (
	"cmp"
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
)

var (
	flagWebSocketServer  string
	flagRetriveCnt       int
	flagPromptPreProcess bool
	hominDevAI           *HominDevAI
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = 25 * time.Second
	processTimeout = 25 * time.Second
	maxBackoff     = 30 * time.Second
)

type wsIncomingMsg struct {
	conn    *websocket.Conn
	msgType int
	msg     []byte
}

func main() {
	defer func() { log.Println("Exiting...") }()

	flag.StringVar(&flagWebSocketServer, "ws", cmp.Or(os.Getenv("RELAY_WS"), "ws://localhost:8080"), "WebSocket server address")
	flag.IntVar(&flagRetriveCnt, "retrive", 50, "Retrive count")
	flag.BoolVar(&flagPromptPreProcess, "pre-process", false, "Pre-process prompt")
	flag.Parse()

	secret := strings.TrimSpace(os.Getenv("RELAY_APIKEY"))
	if secret != "" {
		log.Println("using RELAY_APIKEY from environment")
	}

	stat := &Stat{}
	log.Printf("WebSocket server: %s", flagWebSocketServer)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var err error
	hominDevAI, err = NewHominDevAI(ctx)
	if err != nil {
		log.Fatalf("Failed to create HominDevAI: %v", err)
	}

	var (
		conn       *websocket.Conn
		connMutex  sync.Mutex
		writeMutex sync.Mutex
	)

	getConn := func() *websocket.Conn {
		connMutex.Lock()
		defer connMutex.Unlock()
		return conn
	}

	closeConn := func(target *websocket.Conn) {
		connMutex.Lock()
		defer connMutex.Unlock()
		if target != nil {
			_ = target.Close()
		}
		if conn == target {
			conn = nil
		}
	}

	writeWS := func(target *websocket.Conn, msgType int, data []byte) error {
		if target == nil {
			return fmt.Errorf("no WebSocket connection available")
		}
		writeMutex.Lock()
		defer writeMutex.Unlock()
		if err := target.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
			return err
		}
		return target.WriteMessage(msgType, data)
	}

	connectWS := func() error {
		reqHeader := http.Header{}
		if secret != "" {
			reqHeader.Set("Authorization", fmt.Sprintf("Bearer %s", secret))
		}

		dialer := websocket.Dialer{
			Proxy:            http.ProxyFromEnvironment,
			HandshakeTimeout: 10 * time.Second,
		}

		c, _, err := dialer.DialContext(ctx, flagWebSocketServer, reqHeader)
		if err != nil {
			return fmt.Errorf("failed to connect to WebSocket server: %v", err)
		}

		c.SetReadDeadline(time.Now().Add(pongWait))
		c.SetPongHandler(func(appData string) error {
			log.Println("Received pong")
			c.SetReadDeadline(time.Now().Add(pongWait))
			return nil
		})
		defaultPingHandler := c.PingHandler()
		c.SetPingHandler(func(appData string) error {
			c.SetReadDeadline(time.Now().Add(pongWait))
			return defaultPingHandler(appData)
		})

		connMutex.Lock()
		oldConn := conn
		conn = c
		connMutex.Unlock()

		if oldConn != nil && oldConn != c {
			_ = oldConn.Close()
		}

		return nil
	}

	// get termination signals (systemctl restart sends SIGTERM, not os.Interrupt)
	chCtrlC := make(chan os.Signal, 1)
	signal.Notify(chCtrlC, os.Interrupt, syscall.SIGTERM)

	if err := connectWS(); err != nil {
		log.Printf("Initial connection failed (%v), will retry in background...", err)
	}
	defer func() {
		closeConn(getConn())
	}()

	msgChan := make(chan wsIncomingMsg)
	stopReading := make(chan struct{})
	var stopOnce sync.Once
	stopAll := func() {
		stopOnce.Do(func() {
			close(stopReading)
		})
	}

	// Start message reading and automatic reconnection goroutine
	go func() {
		defer close(msgChan)
		backoff := time.Second

		for {
			select {
			case <-stopReading:
				return
			default:
			}

			currentConn := getConn()
			if currentConn == nil {
				log.Printf("Attempting to connect to WebSocket server (%s)...", flagWebSocketServer)
				if err := connectWS(); err != nil {
					log.Printf("Reconnect failed: %v (retrying in %s)", err, backoff)
					select {
					case <-time.After(backoff):
					case <-stopReading:
						return
					}
					backoff *= 2
					if backoff > maxBackoff {
						backoff = maxBackoff
					}
					continue
				}
				log.Printf("Successfully connected to WebSocket server")
				backoff = time.Second
				currentConn = getConn()
				if currentConn == nil {
					continue
				}
			}

			currentConn.SetReadDeadline(time.Now().Add(pongWait))
			msgType, msgBytes, err := currentConn.ReadMessage()
			if err != nil {
				select {
				case <-stopReading:
					return
				default:
				}

				if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
					log.Printf("Connection closed by server (%v), reconnecting...", err)
				} else {
					log.Printf("Read error (%v), reconnecting...", err)
				}
				closeConn(currentConn)

				select {
				case <-time.After(backoff):
				case <-stopReading:
					return
				}
				continue
			}

			backoff = time.Second
			select {
			case msgChan <- wsIncomingMsg{
				conn:    currentConn,
				msgType: msgType,
				msg:     msgBytes,
			}:
			case <-stopReading:
				return
			}
		}
	}()

	// Send periodic ping
	go func() {
		ticker := time.NewTicker(pingPeriod)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				currentConn := getConn()
				if currentConn != nil {
					if err := writeWS(currentConn, websocket.PingMessage, nil); err != nil {
						log.Printf("Ping error: %v", err)
						closeConn(currentConn)
					}
				}
			case <-stopReading:
				return
			}
		}
	}()

	for {
		select {
		case <-chCtrlC:
			log.Println("Termination signal received, exiting...")
			stopAll()
			currentConn := getConn()
			if currentConn != nil {
				_ = writeWS(currentConn, websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
				closeConn(currentConn)
			}
			cancel()
			return
		case <-ctx.Done():
			log.Println("Context canceled, exiting...")
			stopAll()
			return
		case msgData, ok := <-msgChan:
			if !ok {
				log.Println("Message channel closed, exiting...")
				return
			}

			msg := strings.TrimSpace(string(msgData.msg))

			var reply string
			if len([]rune(msg)) > 200 {
				log.Printf("Message is too long: %d", len([]rune(msg)))
				reply = "메시지가 너무 길어요. 200자 이하로 짧게 줄여주세요."
			} else if len(msg) == 0 {
				log.Printf("Empty message")
				reply = makeAboutReply()
			} else {
				log.Printf("Received message: %s", msg)

				var cmd Cmd
				if !strings.HasPrefix(msg, "/") {
					flowCtx, flowCancel := context.WithTimeout(ctx, processTimeout)
					cmd, err = hominDevAI.PreProcess(flowCtx, msg)
					flowCancel()
					if err != nil {
						log.Printf("Failed to run intent flow: %v", err)
						cmd = Cmd{Action: "/search"}
					}
				} else {
					msgParts := strings.Split(msg, " ")
					if len(msgParts) > 0 {
						cmd = Cmd{
							Action: msgParts[0],
							Args:   msgParts[1:],
						}
					}
				}
				log.Printf("Command: %s", cmd)

				switch cmd.Action {
				case "/keyword":
					searchKeywords := strings.Join(cmd.Args, ",")
					posts, err := retrivePost(searchKeywords, flagRetriveCnt)
					if err != nil {
						log.Printf("Failed to retrive post for keywords, %s: %v", searchKeywords, err)
						reply = "검색 중 일시적인 오류가 발생했습니다. 잠시 후 다시 시도해주세요."
						break
					}
					stat.TotalKeywordCnt++
					reply = fmt.Sprintf("%s 에 대한 검색 결과:\n%s", searchKeywords, makePostReply(posts))
				case "/search":
					posts, err := retrivePost(msg, flagRetriveCnt)
					if err != nil {
						log.Printf("Failed to retrive post for msg, %s: %v", msg, err)
						reply = "검색 중 일시적인 오류가 발생했습니다. 잠시 후 다시 시도해주세요."
						break
					}

					stat.TotalSearchCnt++
					if len(posts) == 0 {
						reply = "검색 결과가 없습니다."
					} else {
						reply = "검색 결과:\n" + makePostReply(posts)
					}
				case "/smallchat":
					stat.TotalSmallChatCnt++
					reply = strings.Join(cmd.Args, "\n")
				case "/about", "/start", "/help":
					reply = makeAboutReply()
				case "/stat":
					reply = stat.String()
				default:
					log.Printf("Unknown command: %s", cmd.Action)
					stat.TotalUnknownCnt++
					reply = makeAboutReply()
				}
			}

			if err := writeWS(msgData.conn, msgData.msgType, []byte(reply)); err != nil {
				log.Printf("Write error: %v", err)
				closeConn(msgData.conn)
			}
		}
	} // for
}

func makePostReply(posts []*Post) string {
	// reply := "검색 결과:\n"
	var reply string
	for i, post := range posts {
		if i >= 5 {
			break
		}
		reply += fmt.Sprintf("- [%s](%s) - %s\n", post.Title, post.Url, strings.Join(post.Texts, ","))
	}
	return reply
}

func makeAboutReply() string {
	siteURL := `https://homin.dev/blog/post/20250507_rag_blog_search_webchat_bot/`

	return fmt.Sprintf(
		`내 이름은 **블검봇**.

Homin Lee's blog를 검색합니다. 편하게 물어보세요.
제가 만들어진 내용은 [여기](%s)에 있습니다.`,
		siteURL,
	)
}

type Stat struct {
	TotalSmallChatCnt int
	TotalKeywordCnt   int
	TotalSearchCnt    int
	TotalUnknownCnt   int
}

func (s *Stat) String() string {
	return fmt.Sprintf(`- TotalSmallChat: %d
- TotalKeyword: %d
- TotalSearch: %d
- TotalUnknown: %d`,
		s.TotalSmallChatCnt, s.TotalKeywordCnt, s.TotalSearchCnt, s.TotalUnknownCnt)
}
