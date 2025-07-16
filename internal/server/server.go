package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"runtime"
	"sync"
	"time"

	"github.com/conneroisu/templar/internal/config"
	"github.com/conneroisu/templar/internal/registry"
	"github.com/conneroisu/templar/internal/renderer"
	"github.com/conneroisu/templar/internal/scanner"
	"github.com/conneroisu/templar/internal/watcher"
	"github.com/gorilla/websocket"
)

// PreviewServer serves components with live reload capability
type PreviewServer struct {
	config       *config.Config
	httpServer   *http.Server
	wsUpgrader   websocket.Upgrader
	clients      map[*websocket.Conn]bool
	clientsMutex sync.RWMutex
	broadcast    chan []byte
	register     chan *websocket.Conn
	unregister   chan *websocket.Conn
	registry     *registry.ComponentRegistry
	watcher      *watcher.FileWatcher
	scanner      *scanner.ComponentScanner
	renderer     *renderer.ComponentRenderer
}

// UpdateMessage represents a message sent to the browser
type UpdateMessage struct {
	Type      string    `json:"type"`
	Target    string    `json:"target,omitempty"`
	Content   string    `json:"content,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// New creates a new preview server
func New(cfg *config.Config) *PreviewServer {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow all origins for development
		},
	}

	registry := registry.NewComponentRegistry()
	
	fileWatcher, err := watcher.NewFileWatcher(300 * time.Millisecond)
	if err != nil {
		log.Fatal("Failed to create file watcher:", err)
	}
	
	scanner := scanner.NewComponentScanner(registry)
	renderer := renderer.NewComponentRenderer(registry)

	return &PreviewServer{
		config:     cfg,
		wsUpgrader: upgrader,
		clients:    make(map[*websocket.Conn]bool),
		broadcast:  make(chan []byte),
		register:   make(chan *websocket.Conn),
		unregister: make(chan *websocket.Conn),
		registry:   registry,
		watcher:    fileWatcher,
		scanner:    scanner,
		renderer:   renderer,
	}
}

// Start starts the preview server
func (s *PreviewServer) Start(ctx context.Context) error {
	// Set up file watcher
	s.setupFileWatcher(ctx)
	
	// Initial scan
	if err := s.initialScan(); err != nil {
		log.Printf("Initial scan failed: %v", err)
	}

	// Start WebSocket hub
	go s.runWebSocketHub(ctx)

	// Set up HTTP routes
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", s.handleWebSocket)
	mux.HandleFunc("/components", s.handleComponents)
	mux.HandleFunc("/component/", s.handleComponent)
	mux.HandleFunc("/render/", s.handleRender)
	mux.HandleFunc("/static/", s.handleStatic)
	
	// Root handler depends on whether specific files are targeted
	if len(s.config.TargetFiles) > 0 {
		mux.HandleFunc("/", s.handleTargetFiles)
	} else {
		mux.HandleFunc("/", s.handleIndex)
	}

	// Add middleware
	handler := s.addMiddleware(mux)

	// Create HTTP server
	addr := fmt.Sprintf("%s:%d", s.config.Server.Host, s.config.Server.Port)
	s.httpServer = &http.Server{
		Addr:    addr,
		Handler: handler,
	}

	// Open browser if configured
	if s.config.Server.Open {
		go s.openBrowser(fmt.Sprintf("http://%s", addr))
	}

	// Start server
	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server error: %w", err)
	}

	return nil
}

func (s *PreviewServer) setupFileWatcher(ctx context.Context) {
	// Add filters
	s.watcher.AddFilter(watcher.TemplFilter)
	s.watcher.AddFilter(watcher.GoFilter)
	s.watcher.AddFilter(watcher.NoTestFilter)
	s.watcher.AddFilter(watcher.NoVendorFilter)
	s.watcher.AddFilter(watcher.NoGitFilter)

	// Add handler
	s.watcher.AddHandler(s.handleFileChange)

	// Add watch paths
	for _, path := range s.config.Components.ScanPaths {
		if err := s.watcher.AddRecursive(path); err != nil {
			log.Printf("Failed to watch path %s: %v", path, err)
		}
	}

	// Start watching
	if err := s.watcher.Start(ctx); err != nil {
		log.Printf("Failed to start file watcher: %v", err)
	}
}

func (s *PreviewServer) initialScan() error {
	log.Printf("Starting initial scan with paths: %v", s.config.Components.ScanPaths)
	for _, path := range s.config.Components.ScanPaths {
		log.Printf("Scanning path: %s", path)
		if err := s.scanner.ScanDirectory(path); err != nil {
			log.Printf("Error scanning %s: %v", path, err)
			// Don't return error, just log and continue
			continue
		}
	}
	
	log.Printf("Found %d components", s.registry.Count())
	return nil
}

func (s *PreviewServer) handleFileChange(events []watcher.ChangeEvent) error {
	for _, event := range events {
		log.Printf("File changed: %s (%s)", event.Path, event.Type)
		
		// Rescan the file
		if err := s.scanner.ScanFile(event.Path); err != nil {
			log.Printf("Failed to rescan file %s: %v", event.Path, err)
		}
	}
	
	// Broadcast reload message
	msg := UpdateMessage{
		Type:      "full_reload",
		Timestamp: time.Now(),
	}
	
	s.broadcastMessage(msg)
	return nil
}

func (s *PreviewServer) openBrowser(url string) {
	time.Sleep(100 * time.Millisecond) // Give server time to start
	
	var err error
	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	
	if err != nil {
		log.Printf("Failed to open browser: %v", err)
	}
}

func (s *PreviewServer) addMiddleware(handler http.Handler) http.Handler {
	// Add CORS and logging middleware
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		
		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		// Log requests
		start := time.Now()
		handler.ServeHTTP(w, r)
		log.Printf("%s %s %v", r.Method, r.URL.Path, time.Since(start))
	})
}

func (s *PreviewServer) broadcastMessage(msg UpdateMessage) {
	// Implementation depends on JSON marshaling
	// For now, just broadcast a simple reload message
	s.broadcast <- []byte(`{"type":"full_reload"}`)
}