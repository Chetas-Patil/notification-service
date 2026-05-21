package notification

import (
	"context"
	"fmt"
	"log"
	"net/smtp"
	"time"
)

type EmailChannel struct {
	config EmailConfig
}

func NewEmailChannel(config EmailConfig) *EmailChannel {
	return &EmailChannel{config: config}
}

// Send delivers the payload via SMTP when configured, or logs it otherwise so
// the demo runs without real credentials.
func (ec *EmailChannel) Send(ctx context.Context, payload *NotificationPayload) (*NotificationResponse, error) {
	response := &NotificationResponse{Timestamp: time.Now()}
	body := ec.prepareEmailBody(payload)

	if ec.config.SMTPHost == "" {
		log.Printf("Email channel unconfigured; would send to %s: subject=%q body=%q", payload.Recipient, payload.Subject, body)
		response.Success = true
		response.MessageID = payload.ID
		return response, nil
	}

	from := ec.config.FromAddress
	message := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s",
		from, payload.Recipient, payload.Subject, body)

	auth := smtp.PlainAuth("", ec.config.SMTPUser, ec.config.SMTPPassword, ec.config.SMTPHost)
	addr := fmt.Sprintf("%s:%d", ec.config.SMTPHost, ec.config.SMTPPort)

	if err := smtp.SendMail(addr, auth, from, []string{payload.Recipient}, []byte(message)); err != nil {
		log.Printf("Error sending email: %v", err)
		response.Success = false
		response.Error = err
		return response, err
	}

	response.Success = true
	response.MessageID = payload.ID
	return response, nil
}

func (ec *EmailChannel) CanHandle(notificationType NotificationType) bool {
	return notificationType == EmailNotification
}

func (ec *EmailChannel) GetChannelType() ChannelType { return Email }

// prepareEmailBody returns the rendered body when present, otherwise serialises
// the payload data as key/value lines.
func (ec *EmailChannel) prepareEmailBody(payload *NotificationPayload) string {
	if body, ok := payload.Data["body"].(string); ok && body != "" {
		return body
	}
	body := ""
	for key, value := range payload.Data {
		body += fmt.Sprintf("%s: %v\n", key, value)
	}
	return body
}
