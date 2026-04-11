package main

import (
	"context"
	"embed"
	"encoding/json"
	"io/fs"
	"log"
	"net/http"
	"sync"

	"nhooyr.io/websocket"
)

//go:embed static/*
var staticFiles embed.FS

// Server handles HTTP requests and WebSocket connections.
type Server struct {
	repoName string
	isLocal  bool
	mu       sync.RWMutex
	latest   *DiffResult
	clients  map[*wsClient]struct{}
	clientMu sync.Mutex
}

type wsClient struct {
	conn *websocket.Conn
	ctx  context.Context
}

// NewServer creates a new Server. When isLocal is true, Origin checks are skipped
// for WebSocket connections (safe for localhost). When false, only same-origin
// connections are accepted.
func NewServer(repoName string, isLocal bool) *Server {
	return &Server{
		repoName: repoName,
		isLocal:  isLocal,
		clients:  make(map[*wsClient]struct{}),
	}
}

// UpdateDiff stores the latest diff and broadcasts to all WebSocket clients.
func (s *Server) UpdateDiff(result *DiffResult) {
	s.mu.Lock()
	s.latest = result
	s.mu.Unlock()

	s.broadcast(result)
}

// Handler returns the HTTP handler for the server.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	staticFS, _ := fs.Sub(staticFiles, "static")
	mux.Handle("/", http.FileServer(http.FS(staticFS)))
	mux.HandleFunc("/api/diff", s.handleDiff)
	mux.HandleFunc("/api/info", s.handleInfo)
	mux.HandleFunc("/ws", s.handleWebSocket)

	return mux
}

func (s *Server) handleDiff(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	result := s.latest
	s.mu.RUnlock()

	if result == nil {
		result = &DiffResult{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (s *Server) handleInfo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"repo": s.repoName,
	})
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: s.isLocal,
	})
	if err != nil {
		log.Printf("websocket accept: %v", err)
		return
	}

	// CloseRead starts a goroutine that reads and discards all incoming messages.
	// The returned context is cancelled when the connection is closed.
	ctx := conn.CloseRead(r.Context())

	client := &wsClient{conn: conn, ctx: ctx}

	s.clientMu.Lock()
	s.clients[client] = struct{}{}
	s.clientMu.Unlock()

	// Send current diff immediately on connect
	s.mu.RLock()
	current := s.latest
	s.mu.RUnlock()
	if current != nil {
		data, _ := json.Marshal(current)
		conn.Write(ctx, websocket.MessageText, data)
	}

	// Block until the connection is closed
	<-ctx.Done()

	s.clientMu.Lock()
	delete(s.clients, client)
	s.clientMu.Unlock()
}

func (s *Server) broadcast(result *DiffResult) {
	data, err := json.Marshal(result)
	if err != nil {
		return
	}

	s.clientMu.Lock()
	clients := make([]*wsClient, 0, len(s.clients))
	for c := range s.clients {
		clients = append(clients, c)
	}
	s.clientMu.Unlock()

	for _, c := range clients {
		if err := c.conn.Write(c.ctx, websocket.MessageText, data); err != nil {
			c.conn.Close(websocket.StatusNormalClosure, "")
		}
	}
}
