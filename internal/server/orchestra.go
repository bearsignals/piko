package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/gwuah/piko/internal/tmux"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type OrchestraMessage struct {
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload"`
	Timestamp time.Time       `json:"timestamp"`
}

type CCNotification struct {
	ID               string    `json:"id"`
	ProjectID        int64     `json:"project_id"`
	ProjectName      string    `json:"project_name"`
	EnvName          string    `json:"env_name"`
	TmuxSession      string    `json:"tmux_session"`
	TmuxTarget       string    `json:"tmux_target"`
	NotificationType string    `json:"notification_type"`
	Message          string    `json:"message"`
	CreatedAt        time.Time `json:"created_at"`
}

type NotifyRequest struct {
	ProjectID        int64  `json:"project_id"`
	ProjectName      string `json:"project_name"`
	EnvName          string `json:"env_name"`
	TmuxSession      string `json:"tmux_session"`
	TmuxTarget       string `json:"tmux_target"`
	NotificationType string `json:"notification_type"`
	Message          string `json:"message"`
}

type RespondRequest struct {
	NotificationID string `json:"notification_id"`
	Response       string `json:"response"`
	ResponseType   string `json:"response_type"`
}

type Hub struct {
	clients       map[*Client]bool
	broadcast     chan []byte
	register      chan *Client
	unregister    chan *Client
	notifications map[string]*CCNotification
	mu            sync.RWMutex
}

type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan []byte
}

func NewHub() *Hub {
	return &Hub{
		clients:       make(map[*Client]bool),
		broadcast:     make(chan []byte, 256),
		register:      make(chan *Client),
		unregister:    make(chan *Client),
		notifications: make(map[string]*CCNotification),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
		case message := <-h.broadcast:
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
		}
	}
}

func (h *Hub) AddNotification(n *CCNotification) {
	h.mu.Lock()
	h.notifications[n.ID] = n
	h.mu.Unlock()

	payload, err := json.Marshal(n)
	if err != nil {
		log.Printf("failed to marshal notification payload: %v", err)
		return
	}
	msg := OrchestraMessage{
		Type:      "notification",
		Payload:   payload,
		Timestamp: time.Now(),
	}
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("failed to marshal notification message: %v", err)
		return
	}
	h.broadcast <- data
}

func (h *Hub) RemoveNotification(id string) *CCNotification {
	h.mu.Lock()
	n, exists := h.notifications[id]
	if exists {
		delete(h.notifications, id)
	}
	h.mu.Unlock()

	if exists {
		payload, err := json.Marshal(map[string]string{"id": id})
		if err != nil {
			log.Printf("failed to marshal dismiss payload: %v", err)
			return n
		}
		msg := OrchestraMessage{
			Type:      "notification_dismissed",
			Payload:   payload,
			Timestamp: time.Now(),
		}
		data, err := json.Marshal(msg)
		if err != nil {
			log.Printf("failed to marshal dismiss message: %v", err)
			return n
		}
		h.broadcast <- data
	}

	return n
}

func (h *Hub) GetNotification(id string) *CCNotification {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.notifications[id]
}

func (h *Hub) ListNotifications() []*CCNotification {
	h.mu.RLock()
	defer h.mu.RUnlock()

	list := make([]*CCNotification, 0, len(h.notifications))
	for _, n := range h.notifications {
		list = append(list, n)
	}
	return list
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(512 * 1024)
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (s *Server) handleOrchestraWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	client := &Client{
		hub:  s.hub,
		conn: conn,
		send: make(chan []byte, 256),
	}
	s.hub.register <- client

	existing := s.hub.ListNotifications()
	for _, n := range existing {
		payload, err := json.Marshal(n)
		if err != nil {
			log.Printf("failed to marshal existing notification payload: %v", err)
			continue
		}
		msg := OrchestraMessage{
			Type:      "notification",
			Payload:   payload,
			Timestamp: time.Now(),
		}
		data, err := json.Marshal(msg)
		if err != nil {
			log.Printf("failed to marshal existing notification message: %v", err)
			continue
		}
		client.send <- data
	}

	go client.writePump()
	go client.readPump()
}

func (s *Server) handleOrchestraNotify(w http.ResponseWriter, r *http.Request) {
	var req NotifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, SuccessResponse{Success: false, Error: "invalid request body"})
		return
	}

	tmuxTarget := req.TmuxTarget
	if tmuxTarget == "" {
		tmuxTarget = req.TmuxSession
	}

	notification := &CCNotification{
		ID:               uuid.New().String(),
		ProjectID:        req.ProjectID,
		ProjectName:      req.ProjectName,
		EnvName:          req.EnvName,
		TmuxSession:      req.TmuxSession,
		TmuxTarget:       tmuxTarget,
		NotificationType: req.NotificationType,
		Message:          req.Message,
		CreatedAt:        time.Now(),
	}

	s.hub.AddNotification(notification)

	writeJSON(w, http.StatusOK, SuccessResponse{Success: true})
}

func (s *Server) handleOrchestraRespond(w http.ResponseWriter, r *http.Request) {
	var req RespondRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, SuccessResponse{Success: false, Error: "invalid request body"})
		return
	}

	notification := s.hub.RemoveNotification(req.NotificationID)
	if notification == nil {
		writeJSON(w, http.StatusNotFound, SuccessResponse{Success: false, Error: "notification not found"})
		return
	}

	var err error
	if req.ResponseType == "keys" {
		err = tmux.SendKeys(notification.TmuxTarget, req.Response)
	} else if req.Response != "" {
		err = tmux.SendText(notification.TmuxTarget, req.Response)
	}

	if err != nil {
		writeJSON(w, http.StatusInternalServerError, SuccessResponse{
			Success: false,
			Error:   fmt.Sprintf("failed to send response to tmux: %v", err),
		})
		return
	}

	writeJSON(w, http.StatusOK, SuccessResponse{Success: true})
}

func (s *Server) handleOrchestraList(w http.ResponseWriter, r *http.Request) {
	notifications := s.hub.ListNotifications()
	writeJSON(w, http.StatusOK, notifications)
}

func (s *Server) handleOrchestraDismiss(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, SuccessResponse{Success: false, Error: "notification id required"})
		return
	}

	notification := s.hub.RemoveNotification(id)
	if notification == nil {
		writeJSON(w, http.StatusNotFound, SuccessResponse{Success: false, Error: "notification not found"})
		return
	}

	writeJSON(w, http.StatusOK, SuccessResponse{Success: true})
}
