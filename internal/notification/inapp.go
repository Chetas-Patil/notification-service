package notification

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// InAppChannel implements the Channel interface for in-app notifications
type InAppChannel struct {
	config InAppConfig
	store  map[string][]*NotificationPayload
	mu     sync.RWMutex
}

// NewInAppChannel creates a new in-app notification channel
func NewInAppChannel(config InAppConfig) *InAppChannel {
	channel := &InAppChannel{
		config: config,
		store:  make(map[string][]*NotificationPayload),
	}
	return channel
}

// Send sends an in-app notification
func (ic *InAppChannel) Send(ctx context.Context, payload *NotificationPayload) (*NotificationResponse, error) {
	response := &NotificationResponse{
		Timestamp: time.Now(),
	}

	// Store the notification
	ic.mu.Lock()
	if _, exists := ic.store[payload.Recipient]; !exists {
		ic.store[payload.Recipient] = make([]*NotificationPayload, 0)
	}
	ic.store[payload.Recipient] = append(ic.store[payload.Recipient], payload)
	ic.mu.Unlock()

	log.Printf("In-app notification stored for user %s: %s", payload.Recipient, payload.Subject)

	response.Success = true
	response.MessageID = payload.ID
	return response, nil
}

// CanHandle checks if this channel can handle the notification type
func (ic *InAppChannel) CanHandle(notificationType NotificationType) bool {
	return notificationType == InAppNotification
}

// GetChannelType returns the channel type
func (ic *InAppChannel) GetChannelType() ChannelType {
	return InApp
}

// GetNotifications retrieves notifications for a specific user
func (ic *InAppChannel) GetNotifications(userID string) []*NotificationPayload {
	ic.mu.RLock()
	defer ic.mu.RUnlock()

	notifications, exists := ic.store[userID]
	if !exists {
		return []*NotificationPayload{}
	}

	return notifications
}

// GetUnreadNotifications retrieves unread notifications for a user
// (For simplicity, we consider all stored notifications as unread)
func (ic *InAppChannel) GetUnreadNotifications(userID string) []*NotificationPayload {
	return ic.GetNotifications(userID)
}

// MarkAsRead marks a notification as read
func (ic *InAppChannel) MarkAsRead(userID string, notificationID string) error {
	ic.mu.Lock()
	defer ic.mu.Unlock()

	notifications, exists := ic.store[userID]
	if !exists {
		return fmt.Errorf("no notifications found for user %s", userID)
	}

	// Find and remove the notification
	for i, notification := range notifications {
		if notification.ID == notificationID {
			// Remove from slice
			ic.store[userID] = append(notifications[:i], notifications[i+1:]...)
			return nil
		}
	}

	return fmt.Errorf("notification %s not found for user %s", notificationID, userID)
}

// ClearNotifications clears all notifications for a user
func (ic *InAppChannel) ClearNotifications(userID string) error {
	ic.mu.Lock()
	defer ic.mu.Unlock()

	delete(ic.store, userID)
	log.Printf("Cleared all notifications for user %s", userID)
	return nil
}

// GetNotificationCount returns the count of unread notifications
func (ic *InAppChannel) GetNotificationCount(userID string) int {
	ic.mu.RLock()
	defer ic.mu.RUnlock()

	notifications, exists := ic.store[userID]
	if !exists {
		return 0
	}

	return len(notifications)
}
