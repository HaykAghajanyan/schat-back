package websocket

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/HaykAghajanyan/chat-backend/internal/models"
	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// Send pings to peer with this period (must be less than pongWait)
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer
	maxMessageSize = 512
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins (in production, restrict this)
		return true
	},
}

type Connection struct {
	Ws *websocket.Conn
}

func NewConnection(conn *websocket.Conn) *Connection {
	return &Connection{Ws: conn}
}

// ReadPump pumps messages from the websocket connection to the hub
func (c *Client) ReadPump(hub *Hub, messageHandler func(*Client, *models.WSMessage)) {
	defer func() {
		hub.Unregister <- c
		c.Conn.Ws.Close()
	}()

	c.Conn.Ws.SetReadLimit(maxMessageSize)
	c.Conn.Ws.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.Ws.SetPongHandler(func(string) error {
		c.Conn.Ws.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.Conn.Ws.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Parse incoming message
		var wsMsg models.WSMessage
		if err := json.Unmarshal(message, &wsMsg); err != nil {
			log.Printf("Error parsing message: %v", err)
			continue
		}

		// Handle the message
		messageHandler(c, &wsMsg)
	}
}

// WritePump pumps messages from the hub to the websocket connection
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Ws.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.Ws.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Hub closed the channel
				c.Conn.Ws.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.Ws.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages to the current websocket message
			n := len(c.Send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.Send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.Conn.Ws.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.Ws.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
