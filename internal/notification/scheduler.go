package notification

import (
	"fmt"
	"log"
	"sync"
	"time"
)

// Scheduler manages scheduled notifications
type Scheduler struct {
	notifications map[string]*ScheduledNotification
	mu            sync.RWMutex
	stopCh        chan struct{}
	running       bool
}

// ScheduledNotification represents a scheduled notification task
type ScheduledNotification struct {
	Payload     *NotificationPayload
	Callback    func()
	ScheduledAt time.Time
}

// NewScheduler creates a new scheduler
func NewScheduler() *Scheduler {
	return &Scheduler{
		notifications: make(map[string]*ScheduledNotification),
		stopCh:        make(chan struct{}),
		running:       false,
	}
}

// Schedule adds a notification to the scheduler
func (s *Scheduler) Schedule(payload *NotificationPayload, callback func()) error {
	if payload == nil {
		return fmt.Errorf("payload cannot be nil")
	}

	if payload.ScheduledAt == nil {
		return fmt.Errorf("scheduled_at must be set")
	}

	if callback == nil {
		return fmt.Errorf("callback cannot be nil")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	scheduled := &ScheduledNotification{
		Payload:     payload,
		Callback:    callback,
		ScheduledAt: *payload.ScheduledAt,
	}

	s.notifications[payload.ID] = scheduled
	log.Printf("Scheduled notification %s for %v", payload.ID, payload.ScheduledAt)

	return nil
}

// Start starts the scheduler
func (s *Scheduler) Start() error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return fmt.Errorf("scheduler already running")
	}
	s.running = true
	s.mu.Unlock()

	go s.run()
	log.Println("Scheduler started")
	return nil
}

// Stop stops the scheduler
func (s *Scheduler) Stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	s.running = false
	s.mu.Unlock()

	close(s.stopCh)
	log.Println("Scheduler stopped")
}

// run is the main scheduler loop
func (s *Scheduler) run() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.checkAndExecute()
		}
	}
}

// checkAndExecute checks for notifications that need to be sent
func (s *Scheduler) checkAndExecute() {
	s.mu.Lock()
	now := time.Now()
	var toDelete []string

	for id, scheduled := range s.notifications {
		if now.After(scheduled.ScheduledAt) || now.Equal(scheduled.ScheduledAt) {
			go func(sched *ScheduledNotification) {
				log.Printf("Executing scheduled notification %s", sched.Payload.ID)
				sched.Callback()
			}(scheduled)

			toDelete = append(toDelete, id)
		}
	}

	// Remove executed notifications
	for _, id := range toDelete {
		delete(s.notifications, id)
	}
	s.mu.Unlock()
}

// GetScheduledNotifications returns all scheduled notifications
func (s *Scheduler) GetScheduledNotifications() map[string]*ScheduledNotification {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Return a copy
	copy := make(map[string]*ScheduledNotification)
	for k, v := range s.notifications {
		copy[k] = v
	}
	return copy
}

// CancelNotification cancels a scheduled notification
func (s *Scheduler) CancelNotification(notificationID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.notifications[notificationID]; !exists {
		return fmt.Errorf("notification %s not found", notificationID)
	}

	delete(s.notifications, notificationID)
	log.Printf("Cancelled scheduled notification %s", notificationID)
	return nil
}

// GetScheduledNotification returns a specific scheduled notification
func (s *Scheduler) GetScheduledNotification(notificationID string) (*ScheduledNotification, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	scheduled, exists := s.notifications[notificationID]
	if !exists {
		return nil, fmt.Errorf("notification %s not found", notificationID)
	}

	return scheduled, nil
}
