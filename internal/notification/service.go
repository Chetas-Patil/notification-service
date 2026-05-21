package notification

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// Service routes notifications to the right channels, optionally rendering a
// template first. It is safe for concurrent use.
type Service struct {
	channels     map[ChannelType]Channel
	mappingRules map[NotificationType][]ChannelType
	renderer     TemplateRenderer
	scheduler    *Scheduler
	mu           sync.RWMutex
}

func NewService() *Service {
	return &Service{
		channels:     make(map[ChannelType]Channel),
		mappingRules: make(map[NotificationType][]ChannelType),
		scheduler:    NewScheduler(),
	}
}

// SetTemplateRenderer wires the engine used when a payload carries a TemplateID.
func (s *Service) SetTemplateRenderer(r TemplateRenderer) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.renderer = r
}

func (s *Service) RegisterChannel(channel Channel) error {
	if channel == nil {
		return fmt.Errorf("channel cannot be nil")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.channels[channel.GetChannelType()] = channel
	log.Printf("Registered channel: %s", channel.GetChannelType())
	return nil
}

func (s *Service) RegisterChannelMapping(notificationType NotificationType, channels ...ChannelType) error {
	if len(channels) == 0 {
		return fmt.Errorf("at least one channel must be specified")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.mappingRules[notificationType] = channels
	log.Printf("Registered channel mapping for %s: %v", notificationType, channels)
	return nil
}

// Send dispatches a notification to every channel mapped to its type. If the
// payload references a template, it is rendered first. Returns the first
// channel's response so simple callers don't have to switch on a slice.
func (s *Service) Send(ctx context.Context, payload *NotificationPayload) (*NotificationResponse, error) {
	responses, err := s.SendToAllChannels(ctx, payload)
	if err != nil {
		return nil, err
	}
	if len(responses) == 0 {
		return nil, fmt.Errorf("no responses produced for notification %s", payload.ID)
	}
	return responses[0], nil
}

// SendToAllChannels fans the payload out to every channel mapped to its type.
func (s *Service) SendToAllChannels(ctx context.Context, payload *NotificationPayload) ([]*NotificationResponse, error) {
	if payload == nil {
		return nil, fmt.Errorf("payload cannot be nil")
	}
	if payload.CreatedAt.IsZero() {
		payload.CreatedAt = time.Now()
	}
	if err := s.applyTemplate(payload); err != nil {
		return nil, err
	}

	channels := s.getChannelsForNotificationType(payload.Type)
	if len(channels) == 0 {
		return nil, fmt.Errorf("no channels registered for notification type: %s", payload.Type)
	}

	responses := make([]*NotificationResponse, 0, len(channels))
	for _, channelType := range channels {
		channel, err := s.GetChannel(channelType)
		if err != nil {
			log.Printf("Error getting channel %s: %v", channelType, err)
			responses = append(responses, &NotificationResponse{
				Success:   false,
				Timestamp: time.Now(),
				Error:     err,
				ErrorMsg:  err.Error(),
			})
			continue
		}

		response, err := channel.Send(ctx, payload)
		if err != nil {
			log.Printf("Error sending to channel %s: %v", channelType, err)
			if response == nil {
				response = &NotificationResponse{Timestamp: time.Now()}
			}
			response.Success = false
			response.Error = err
			response.ErrorMsg = err.Error()
		}
		responses = append(responses, response)
	}
	return responses, nil
}

// SendScheduled stores the payload and fires it through Send when its
// ScheduledAt time arrives.
func (s *Service) SendScheduled(ctx context.Context, payload *NotificationPayload) error {
	if payload == nil {
		return fmt.Errorf("payload cannot be nil")
	}
	if payload.ScheduledAt == nil {
		return fmt.Errorf("scheduled_at must be set")
	}
	log.Printf("Scheduling notification %s for %v", payload.ID, payload.ScheduledAt)
	return s.scheduler.Schedule(payload, func() {
		if _, err := s.Send(ctx, payload); err != nil {
			log.Printf("Error sending scheduled notification %s: %v", payload.ID, err)
		}
	})
}

func (s *Service) GetChannel(channelType ChannelType) (Channel, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	channel, exists := s.channels[channelType]
	if !exists {
		return nil, fmt.Errorf("channel type %s not found", channelType)
	}
	return channel, nil
}

func (s *Service) GetChannels() map[ChannelType]Channel {
	s.mu.RLock()
	defer s.mu.RUnlock()
	channelsCopy := make(map[ChannelType]Channel, len(s.channels))
	for k, v := range s.channels {
		channelsCopy[k] = v
	}
	return channelsCopy
}

func (s *Service) getChannelsForNotificationType(notificationType NotificationType) []ChannelType {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.mappingRules[notificationType]
}

func (s *Service) applyTemplate(payload *NotificationPayload) error {
	if payload.TemplateID == "" {
		return nil
	}
	s.mu.RLock()
	renderer := s.renderer
	s.mu.RUnlock()
	if renderer == nil {
		return fmt.Errorf("payload references template %q but no renderer is configured", payload.TemplateID)
	}

	subject, body, err := renderer.Render(payload.TemplateID, payload.TemplateData)
	if err != nil {
		return fmt.Errorf("render template %s: %w", payload.TemplateID, err)
	}

	payload.Subject = subject
	if payload.Data == nil {
		payload.Data = map[string]interface{}{}
	}
	payload.Data["body"] = body
	return nil
}

func (s *Service) Start() error { return s.scheduler.Start() }
func (s *Service) Stop()        { s.scheduler.Stop() }

// Scheduler exposes the underlying scheduler for callers that need to list or
// cancel jobs (e.g. the HTTP layer).
func (s *Service) Scheduler() *Scheduler { return s.scheduler }
