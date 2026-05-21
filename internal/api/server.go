package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"notification-service/internal/notification"
	"notification-service/internal/template"
)

// Server exposes the notification service over HTTP. It owns no state beyond
// the embedded service/engine references and an http.Server it can shut down.
type Server struct {
	svc        *notification.Service
	templates  *template.Engine
	httpServer *http.Server
}

func NewServer(addr string, svc *notification.Service, templates *template.Engine) *Server {
	s := &Server{svc: svc, templates: templates}
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.handleHealth)
	mux.HandleFunc("/notifications", s.handleNotifications)
	mux.HandleFunc("/notifications/", s.handleUserNotifications)
	mux.HandleFunc("/templates", s.handleTemplates)
	mux.HandleFunc("/templates/", s.handleTemplate)

	s.httpServer = &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	return s
}

func (s *Server) Start() error {
	log.Printf("HTTP API listening on %s", s.httpServer.Addr)
	if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

// --- handlers --------------------------------------------------------------

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// POST /notifications  — instant or scheduled (when scheduled_at is set).
func (s *Server) handleNotifications(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var payload notification.NotificationPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid json: %v", err))
		return
	}
	if payload.Type == "" {
		writeError(w, http.StatusBadRequest, "type is required")
		return
	}
	if payload.ID == "" {
		payload.ID = fmt.Sprintf("notif-%d", time.Now().UnixNano())
	}

	if payload.ScheduledAt != nil && payload.ScheduledAt.After(time.Now()) {
		if err := s.svc.SendScheduled(r.Context(), &payload); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusAccepted, map[string]interface{}{
			"id":           payload.ID,
			"scheduled_at": payload.ScheduledAt,
			"status":       "scheduled",
		})
		return
	}

	responses, err := s.svc.SendToAllChannels(r.Context(), &payload)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"id":        payload.ID,
		"responses": responses,
	})
}

// GET /notifications/{userID} — list in-app notifications for one user.
func (s *Server) handleUserNotifications(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	userID := strings.TrimPrefix(r.URL.Path, "/notifications/")
	if userID == "" {
		writeError(w, http.StatusBadRequest, "user id required")
		return
	}
	channel, err := s.svc.GetChannel(notification.InApp)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "in-app channel not registered")
		return
	}
	inApp, ok := channel.(*notification.InAppChannel)
	if !ok {
		writeError(w, http.StatusInternalServerError, "in-app channel has unexpected type")
		return
	}
	writeJSON(w, http.StatusOK, inApp.GetNotifications(userID))
}

// POST /templates / GET /templates — create or list.
func (s *Server) handleTemplates(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		var t template.Template
		if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid json: %v", err))
			return
		}
		if err := s.templates.RegisterTemplate(&t); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, &t)
	case http.MethodGet:
		writeJSON(w, http.StatusOK, s.templates.ListTemplates())
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// GET /templates/{id} / PUT /templates/{id} / DELETE /templates/{id}.
func (s *Server) handleTemplate(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/templates/")
	if id == "" {
		writeError(w, http.StatusBadRequest, "template id required")
		return
	}
	switch r.Method {
	case http.MethodGet:
		t, err := s.templates.GetTemplate(id)
		if err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, t)
	case http.MethodPut:
		var t template.Template
		if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid json: %v", err))
			return
		}
		if err := s.templates.UpdateTemplate(id, &t); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, &t)
	case http.MethodDelete:
		if err := s.templates.DeleteTemplate(id); err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// --- helpers ---------------------------------------------------------------

func writeJSON(w http.ResponseWriter, status int, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// Handler exposes the underlying mux so tests can drive it with httptest.
func (s *Server) Handler() http.Handler { return s.httpServer.Handler }
