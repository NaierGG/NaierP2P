package websocket

import (
	"log"
	"sync"
	"time"

	gorillaws "github.com/gorilla/websocket"
	"github.com/google/uuid"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 64 * 1024
)

type Client struct {
	ID       uuid.UUID
	UserID   uuid.UUID
	DeviceID uuid.UUID
	Conn     *gorillaws.Conn
	Hub      *Hub
	Send     chan []byte
	Channels map[uuid.UUID]bool
	mu       sync.RWMutex
}

func (c *Client) readPump() {
	defer func() {
		c.Hub.unregister <- c
		_ = c.Conn.Close()
	}()

	c.Conn.SetReadLimit(maxMessageSize)
	_ = c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error {
		return c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		var event WSEvent
		if err := c.Conn.ReadJSON(&event); err != nil {
			if !gorillaws.IsUnexpectedCloseError(err, gorillaws.CloseGoingAway, gorillaws.CloseAbnormalClosure) {
				log.Printf("websocket read closed for client %s: %v", c.ID, err)
			}
			return
		}

		c.Hub.router.HandleClientEvent(c, event)
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		_ = c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			_ = c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.Conn.WriteMessage(gorillaws.CloseMessage, []byte{})
				return
			}

			writer, err := c.Conn.NextWriter(gorillaws.TextMessage)
			if err != nil {
				return
			}

			if _, err := writer.Write(message); err != nil {
				_ = writer.Close()
				return
			}

			if err := writer.Close(); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(gorillaws.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *Client) subscribe(channelID uuid.UUID) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Channels[channelID] = true
}

func (c *Client) unsubscribe(channelID uuid.UUID) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.Channels, channelID)
}

func (c *Client) isSubscribed(channelID uuid.UUID) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Channels[channelID]
}

