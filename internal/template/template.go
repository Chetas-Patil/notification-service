package template

import (
	"bytes"
	"fmt"
	"log"
	"sync"
	"text/template"
)

// Template represents a notification template
type Template struct {
	ID       string
	Name     string
	Subject  string
	Body     string
	Channel  string
}

// Engine manages notification templates
type Engine struct {
	templates map[string]*Template
	mu        sync.RWMutex
}

// NewEngine creates a new template engine
func NewEngine() *Engine {
	return &Engine{
		templates: make(map[string]*Template),
	}
}

// RegisterTemplate registers a new template
func (e *Engine) RegisterTemplate(tmpl *Template) error {
	if tmpl == nil {
		return fmt.Errorf("template cannot be nil")
	}

	if tmpl.ID == "" {
		return fmt.Errorf("template ID cannot be empty")
	}

	// Validate template syntax
	if _, err := template.New("subject").Parse(tmpl.Subject); err != nil {
		return fmt.Errorf("invalid subject template: %w", err)
	}

	if _, err := template.New("body").Parse(tmpl.Body); err != nil {
		return fmt.Errorf("invalid body template: %w", err)
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	e.templates[tmpl.ID] = tmpl
	log.Printf("Registered template: %s", tmpl.ID)
	return nil
}

// GetTemplate retrieves a template by ID
func (e *Engine) GetTemplate(id string) (*Template, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	tmpl, exists := e.templates[id]
	if !exists {
		return nil, fmt.Errorf("template %s not found", id)
	}

	return tmpl, nil
}

// RenderTemplate renders a template with the given data
func (e *Engine) RenderTemplate(templateID string, data map[string]interface{}) (*RenderedTemplate, error) {
	tmpl, err := e.GetTemplate(templateID)
	if err != nil {
		return nil, err
	}

	// Render subject
	subjectBuf := new(bytes.Buffer)
	subjectTmpl, err := template.New("subject").Parse(tmpl.Subject)
	if err != nil {
		return nil, fmt.Errorf("failed to parse subject template: %w", err)
	}

	if err := subjectTmpl.Execute(subjectBuf, data); err != nil {
		return nil, fmt.Errorf("failed to render subject: %w", err)
	}

	// Render body
	bodyBuf := new(bytes.Buffer)
	bodyTmpl, err := template.New("body").Parse(tmpl.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse body template: %w", err)
	}

	if err := bodyTmpl.Execute(bodyBuf, data); err != nil {
		return nil, fmt.Errorf("failed to render body: %w", err)
	}

	return &RenderedTemplate{
		Subject: subjectBuf.String(),
		Body:    bodyBuf.String(),
		Channel: tmpl.Channel,
	}, nil
}

// ListTemplates returns all registered templates
func (e *Engine) ListTemplates() []*Template {
	e.mu.RLock()
	defer e.mu.RUnlock()

	templates := make([]*Template, 0, len(e.templates))
	for _, tmpl := range e.templates {
		templates = append(templates, tmpl)
	}
	return templates
}

// DeleteTemplate removes a template
func (e *Engine) DeleteTemplate(id string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, exists := e.templates[id]; !exists {
		return fmt.Errorf("template %s not found", id)
	}

	delete(e.templates, id)
	log.Printf("Deleted template: %s", id)
	return nil
}

// RenderedTemplate represents a rendered template
type RenderedTemplate struct {
	Subject string
	Body    string
	Channel string
}

// Render satisfies the notification.TemplateRenderer interface so the engine
// can be plugged straight into the notification service.
func (e *Engine) Render(templateID string, data map[string]interface{}) (string, string, error) {
	rendered, err := e.RenderTemplate(templateID, data)
	if err != nil {
		return "", "", err
	}
	return rendered.Subject, rendered.Body, nil
}

// UpdateTemplate updates an existing template
func (e *Engine) UpdateTemplate(id string, tmpl *Template) error {
	if _, err := e.GetTemplate(id); err != nil {
		return err
	}

	if tmpl.ID == "" {
		tmpl.ID = id
	}

	// Validate template syntax
	if _, err := template.New("subject").Parse(tmpl.Subject); err != nil {
		return fmt.Errorf("invalid subject template: %w", err)
	}

	if _, err := template.New("body").Parse(tmpl.Body); err != nil {
		return fmt.Errorf("invalid body template: %w", err)
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	e.templates[id] = tmpl
	log.Printf("Updated template: %s", id)
	return nil
}
