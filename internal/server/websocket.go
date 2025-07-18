package server

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

func (s *PreviewServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := s.wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	client := &Client{
		conn:   conn,
		send:   make(chan []byte, 256),
		server: s,
	}

	// Start goroutines for this client first
	go client.writePump()
	go client.readPump()

	// Register client after goroutines are started
	s.register <- client
}

func (s *PreviewServer) runWebSocketHub(ctx context.Context) {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case client := <-s.register:
			if client == nil || client.conn == nil {
				continue
			}
			s.clientsMutex.Lock()
			if s.clients != nil {
				s.clients[client.conn] = client
				clientCount := len(s.clients)
				s.clientsMutex.Unlock()
				log.Printf("Client connected, total: %d", clientCount)
			} else {
				s.clientsMutex.Unlock()
			}

		case conn := <-s.unregister:
			if conn == nil {
				continue
			}
			s.clientsMutex.Lock()
			if s.clients != nil {
				if client, ok := s.clients[conn]; ok {
					delete(s.clients, conn)
					close(client.send)
					conn.Close()
					log.Printf("Client disconnected, total: %d", len(s.clients))
				}
			}
			s.clientsMutex.Unlock()

		case message := <-s.broadcast:
			s.clientsMutex.RLock()
			var failedClients []*websocket.Conn
			if s.clients != nil {
				for conn, client := range s.clients {
					select {
					case client.send <- message:
					default:
						// Client's send channel is full, mark for removal
						failedClients = append(failedClients, conn)
					}
				}
			}
			s.clientsMutex.RUnlock()

			// Clean up failed clients outside the read lock
			if len(failedClients) > 0 {
				s.clientsMutex.Lock()
				if s.clients != nil {
					for _, conn := range failedClients {
						if client, ok := s.clients[conn]; ok {
							delete(s.clients, conn)
							close(client.send)
							conn.Close()
						}
					}
				}
				s.clientsMutex.Unlock()
			}
		}
	}
}

// readPump pumps messages from the websocket connection
func (c *Client) readPump() {
	defer func() {
		c.server.unregister <- c.conn
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}
	}
}

// writePump pumps messages to the websocket connection
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("WebSocket write error: %v", err)
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
