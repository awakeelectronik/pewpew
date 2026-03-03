package http

import (
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"pewpew/internal/domain"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // MVP: permitir todos (luego restringir a localhost)
	},
}

// EventBroadcaster maneja broadcast de eventos a clientes WebSocket
type EventBroadcaster struct {
	clients  map[*websocket.Conn]bool
	writeMu  map[*websocket.Conn]*sync.Mutex // serializa escrituras por conexión (gorilla no permite concurrent write)
	mu       sync.Mutex
	eventCh  chan *domain.SecurityEvent
}

var broadcaster *EventBroadcaster

// InitEventBroadcaster inicializa el broadcaster
func InitEventBroadcaster() {
	broadcaster = &EventBroadcaster{
		clients: make(map[*websocket.Conn]bool),
		writeMu: make(map[*websocket.Conn]*sync.Mutex),
		eventCh: make(chan *domain.SecurityEvent, 100),
	}

	go broadcaster.run()
}

// BroadcastEvent envía un evento a todos los clientes
func BroadcastEvent(event *domain.SecurityEvent) {
	if broadcaster != nil {
		select {
		case broadcaster.eventCh <- event:
		default:
			log.Println("broadcast channel full, dropping event")
		}
	}
}

// run procesa broadcast de eventos
func (b *EventBroadcaster) run() {
	for event := range b.eventCh {
		b.mu.Lock()
		for client, connMu := range b.writeMu {
			connMu := connMu
			client := client
			e := event
			go func() {
				connMu.Lock()
				defer connMu.Unlock()
				if err := client.WriteJSON(e); err != nil {
					log.Printf("websocket write error: %v", err)
					client.Close()
					b.mu.Lock()
					delete(b.clients, client)
					delete(b.writeMu, client)
					b.mu.Unlock()
				}
			}()
		}
		b.mu.Unlock()
	}
}

// handleWebSocket maneja conexiones WebSocket
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("websocket upgrade error: %v", err)
		return
	}
	defer conn.Close()

	broadcaster.mu.Lock()
	broadcaster.clients[conn] = true
	broadcaster.writeMu[conn] = &sync.Mutex{}
	broadcaster.mu.Unlock()

	log.Println("websocket client connected")

	// Keep connection alive
	for {
		var msg map[string]string
		if err := conn.ReadJSON(&msg); err != nil {
			broadcaster.mu.Lock()
			delete(broadcaster.clients, conn)
			delete(broadcaster.writeMu, conn)
			broadcaster.mu.Unlock()
			log.Println("websocket client disconnected")
			break
		}
	}
}
