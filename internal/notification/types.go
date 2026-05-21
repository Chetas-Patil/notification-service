package notification

import (
	"context"
	"time"
)

type NotificationType string

const (
	EmailNotification NotificationType = "email"
	SlackNotification NotificationType = "slack"
	InAppNotification NotificationType = "inapp"
)

type ChannelType string

const (
	Email ChannelType = "email"
	Slack ChannelType = "slack"
	InApp ChannelType = "inapp"
)

// NotificationPayload is the request to deliver one notification. If TemplateID
// is set the service renders Subject/body from the template using TemplateData
// before dispatching to the channel.
type NotificationPayload struct {
	ID           string                 `json:"id"`
	Type         NotificationType       `json:"type"`
	Recipient    string                 `json:"recipient"`
	Subject      string                 `json:"subject,omitempty"`
	Data         map[string]interface{} `json:"data,omitempty"`
	TemplateID   string                 `json:"template_id,omitempty"`
	TemplateData map[string]interface{} `json:"template_data,omitempty"`
	ScheduledAt  *time.Time             `json:"scheduled_at,omitempty"`
	CreatedAt    time.Time              `json:"created_at,omitempty"`
}

type NotificationResponse struct {
	Success   bool      `json:"success"`
	MessageID string    `json:"message_id,omitempty"`
	Error     error     `json:"-"`
	ErrorMsg  string    `json:"error,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

type Channel interface {
	Send(ctx context.Context, payload *NotificationPayload) (*NotificationResponse, error)
	CanHandle(notificationType NotificationType) bool
	GetChannelType() ChannelType
}

// TemplateRenderer is implemented by the template engine. Declared here so the
// notification service can depend on the interface rather than the concrete
// package (keeps the dependency one-way and tests trivial).
type TemplateRenderer interface {
	Render(templateID string, data map[string]interface{}) (subject, body string, err error)
}

type EmailConfig struct {
	SMTPHost     string `mapstructure:"smtp_host"`
	SMTPPort     int    `mapstructure:"smtp_port"`
	SMTPUser     string `mapstructure:"smtp_user"`
	SMTPPassword string `mapstructure:"smtp_password"`
	FromAddress  string `mapstructure:"from_address"`
}

type SlackConfig struct {
	WebhookURL string `mapstructure:"webhook_url"`
	Token      string `mapstructure:"token"`
	Channel    string `mapstructure:"channel"`
}

type InAppConfig struct {
	DatabaseURL  string        `mapstructure:"database_url"`
	MaxRetention time.Duration `mapstructure:"max_retention"`
}
