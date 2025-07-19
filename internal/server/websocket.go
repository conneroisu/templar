package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"

	"nhooyr.io/websocket"
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
	// Validate origin before accepting connection
	if !s.checkOrigin(r) {
		http.Error(w, "Origin not allowed", http.StatusForbidden)
		return
	}

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: false, // Always verify origin
	})
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

// checkOrigin validates the request origin for security
func (s *PreviewServer) checkOrigin(r *http.Request) bool {
	// Get the origin from the request
	origin := r.Header.Get("Origin")
	if origin == "" {
		// Reject connections without origin header for security
		return false
	}

	// Parse the origin URL
	originURL, err := url.Parse(origin)
	if err != nil {
		return false
	}

	// First check scheme - only allow http/https
	if originURL.Scheme != "http" && originURL.Scheme != "https" {
		return false
	}

	// Strict origin validation - only allow specific origins
	expectedHost := fmt.Sprintf("%s:%d", s.config.Server.Host, s.config.Server.Port)
	allowedOrigins := []string{
		expectedHost,
		fmt.Sprintf("localhost:%d", s.config.Server.Port),
		fmt.Sprintf("127.0.0.1:%d", s.config.Server.Port),
		"localhost:3000", // Common dev server
		"127.0.0.1:3000", // Common dev server
	}

	// Check if origin is in allowed list
	for _, allowed := range allowedOrigins {
		if originURL.Host == allowed {
			return true
		}
	}

	return false
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
					conn.Close(websocket.StatusNormalClosure, "")
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
							conn.Close(websocket.StatusNormalClosure, "")
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
		c.conn.Close(websocket.StatusNormalClosure, "")
	}()

	// Set read limit
	c.conn.SetReadLimit(maxMessageSize)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for {
		// Set read timeout
		readCtx, readCancel := context.WithTimeout(ctx, pongWait)
		_, _, err := c.conn.Read(readCtx)
		readCancel()

		if err != nil {
			// Check if it's a normal closure
			if websocket.CloseStatus(err) != websocket.StatusNormalClosure &&
				websocket.CloseStatus(err) != websocket.StatusGoingAway {
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
		c.conn.Close(websocket.StatusNormalClosure, "")
	}()

	ctx := context.Background()

	for {
		select {
		case message, ok := <-c.send:
			writeCtx, cancel := context.WithTimeout(ctx, writeWait)
			if !ok {
				c.conn.Close(websocket.StatusNormalClosure, "")
				cancel()
				return
			}

			if err := c.conn.Write(writeCtx, websocket.MessageText, message); err != nil {
				log.Printf("WebSocket write error: %v", err)
				cancel()
				return
			}
			cancel()

		case <-ticker.C:
			pingCtx, cancel := context.WithTimeout(ctx, writeWait)
			if err := c.conn.Ping(pingCtx); err != nil {
				cancel()
				return
			}
			cancel()
		}
	}
}
