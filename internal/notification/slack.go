package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

type SlackChannel struct {
	config     SlackConfig
	httpClient *http.Client
}

func NewSlackChannel(config SlackConfig) *SlackChannel {
	return &SlackChannel{
		config:     config,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

type SlackMessage struct {
	Channel string       `json:"channel,omitempty"`
	Text    string       `json:"text"`
	Blocks  []SlackBlock `json:"blocks,omitempty"`
}

type SlackBlock struct {
	Type   string      `json:"type"`
	Text   SlackText   `json:"text,omitempty"`
	Fields []SlackText `json:"fields,omitempty"`
}

type SlackText struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// Send delivers the payload to Slack via webhook when configured, or logs it
// otherwise so local development works without real credentials.
func (sc *SlackChannel) Send(ctx context.Context, payload *NotificationPayload) (*NotificationResponse, error) {
	response := &NotificationResponse{Timestamp: time.Now()}
	message := sc.prepareSlackMessage(payload)

	if sc.config.WebhookURL == "" {
		msgBytes, _ := json.Marshal(message)
		log.Printf("Slack channel unconfigured; would send to %s: %s", payload.Recipient, string(msgBytes))
		response.Success = true
		response.MessageID = payload.ID
		return response, nil
	}

	if err := sc.postWebhook(ctx, sc.config.WebhookURL, message); err != nil {
		response.Success = false
		response.Error = err
		return response, err
	}

	response.Success = true
	response.MessageID = payload.ID
	return response, nil
}

func (sc *SlackChannel) CanHandle(notificationType NotificationType) bool {
	return notificationType == SlackNotification
}

func (sc *SlackChannel) GetChannelType() ChannelType {
	return Slack
}

func (sc *SlackChannel) prepareSlackMessage(payload *NotificationPayload) *SlackMessage {
	msg := &SlackMessage{
		Channel: payload.Recipient,
		Text:    payload.Subject,
	}

	block := SlackBlock{
		Type: "section",
		Text: SlackText{
			Type: "mrkdwn",
			Text: fmt.Sprintf("*%s*", payload.Subject),
		},
	}

	if body, ok := payload.Data["body"].(string); ok && body != "" {
		block.Text.Text = fmt.Sprintf("*%s*\n%s", payload.Subject, body)
	}

	for key, value := range payload.Data {
		if key == "body" {
			continue
		}
		block.Fields = append(block.Fields, SlackText{
			Type: "mrkdwn",
			Text: fmt.Sprintf("*%s:*\n%v", key, value),
		})
	}

	msg.Blocks = append(msg.Blocks, block)
	return msg
}

func (sc *SlackChannel) postWebhook(ctx context.Context, webhookURL string, message *SlackMessage) error {
	body, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("marshal slack message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("build slack request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := sc.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("post to slack: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("slack webhook returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}
	return nil
}
