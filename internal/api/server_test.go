package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"notification-service/internal/notification"
	"notification-service/internal/template"
)

func newTestApp(t *testing.T) (*Server, *notification.Service, *template.Engine) {
	t.Helper()
	svc := notification.NewService()
	engine := template.NewEngine()
	svc.SetTemplateRenderer(engine)

	if err := svc.RegisterChannel(notification.NewInAppChannel(notification.InAppConfig{})); err != nil {
		t.Fatal(err)
	}
	if err := svc.RegisterChannelMapping(notification.InAppNotification, notification.InApp); err != nil {
		t.Fatal(err)
	}
	return NewServer(":0", svc, engine), svc, engine
}

func TestAPI_PostAndGetInAppNotification(t *testing.T) {
	server, _, _ := newTestApp(t)

	body, _ := json.Marshal(map[string]interface{}{
		"id":        "n-1",
		"type":      "inapp",
		"recipient": "alice",
		"subject":   "hello",
		"data":      map[string]interface{}{"message": "test"},
	})
	req := httptest.NewRequest(http.MethodPost, "/notifications", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("POST /notifications: want 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/notifications/alice", nil)
	rec = httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /notifications/alice: want 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "n-1") {
		t.Errorf("expected response to contain notification id, got %s", rec.Body.String())
	}
}

func TestAPI_RegisterAndUseTemplate(t *testing.T) {
	server, svc, _ := newTestApp(t)

	tmplBody, _ := json.Marshal(map[string]string{
		"ID":      "greet",
		"Name":    "Greeting",
		"Subject": "Hi {{.Name}}",
		"Body":    "Welcome {{.Name}}",
		"Channel": "inapp",
	})
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/templates", bytes.NewReader(tmplBody)))
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST /templates: want 201, got %d body=%s", rec.Code, rec.Body.String())
	}

	payload, _ := json.Marshal(map[string]interface{}{
		"id":            "tpl-send",
		"type":          "inapp",
		"recipient":     "bob",
		"template_id":   "greet",
		"template_data": map[string]string{"Name": "Bob"},
	})
	rec = httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/notifications", bytes.NewReader(payload)))
	if rec.Code != http.StatusOK {
		t.Fatalf("POST /notifications: want 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	channel, _ := svc.GetChannel(notification.InApp)
	stored := channel.(*notification.InAppChannel).GetNotifications("bob")
	if len(stored) != 1 {
		t.Fatalf("expected 1 stored notification, got %d", len(stored))
	}
	if stored[0].Subject != "Hi Bob" {
		t.Errorf("expected rendered subject 'Hi Bob', got %q", stored[0].Subject)
	}
}

func TestAPI_ScheduleNotificationReturns202(t *testing.T) {
	server, _, _ := newTestApp(t)
	if err := server.svc.Start(); err != nil {
		t.Fatal(err)
	}
	defer server.svc.Stop()

	when := time.Now().Add(1 * time.Hour)
	body, _ := json.Marshal(map[string]interface{}{
		"id":           "sched-1",
		"type":         "inapp",
		"recipient":    "alice",
		"subject":      "later",
		"scheduled_at": when,
	})
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/notifications", bytes.NewReader(body)))
	if rec.Code != http.StatusAccepted {
		t.Fatalf("scheduled POST: want 202, got %d body=%s", rec.Code, rec.Body.String())
	}
}
