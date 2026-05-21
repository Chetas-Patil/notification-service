package notification_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"notification-service/internal/notification"
)

func TestScheduler_FiresCallbackAtScheduledTime(t *testing.T) {
	s := notification.NewScheduler()
	require.NoError(t, s.Start())
	defer s.Stop()

	fired := make(chan struct{}, 1)
	scheduledAt := time.Now().Add(50 * time.Millisecond)
	err := s.Schedule(
		&notification.NotificationPayload{ID: "sc-1", ScheduledAt: &scheduledAt},
		func() { fired <- struct{}{} },
	)
	require.NoError(t, err)

	select {
	case <-fired:
	case <-time.After(3 * time.Second):
		t.Fatal("callback was not fired within deadline")
	}
}

func TestScheduler_CancelPreventsExecution(t *testing.T) {
	s := notification.NewScheduler()
	require.NoError(t, s.Start())
	defer s.Stop()

	fired := make(chan struct{}, 1)
	scheduledAt := time.Now().Add(500 * time.Millisecond)
	err := s.Schedule(
		&notification.NotificationPayload{ID: "sc-cancel", ScheduledAt: &scheduledAt},
		func() { fired <- struct{}{} },
	)
	require.NoError(t, err)
	require.NoError(t, s.CancelNotification("sc-cancel"))

	select {
	case <-fired:
		t.Fatal("cancelled notification should not have fired")
	case <-time.After(700 * time.Millisecond):
		// correct: nothing fired
	}
}

func TestScheduler_CancelUnknownIDReturnsError(t *testing.T) {
	s := notification.NewScheduler()
	assert.Error(t, s.CancelNotification("nonexistent"))
}

func TestScheduler_NilPayloadReturnsError(t *testing.T) {
	s := notification.NewScheduler()
	assert.Error(t, s.Schedule(nil, func() {}))
}

func TestScheduler_NoScheduledAtReturnsError(t *testing.T) {
	s := notification.NewScheduler()
	assert.Error(t, s.Schedule(&notification.NotificationPayload{ID: "x"}, func() {}))
}

func TestScheduler_NilCallbackReturnsError(t *testing.T) {
	s := notification.NewScheduler()
	at := time.Now().Add(time.Hour)
	assert.Error(t, s.Schedule(&notification.NotificationPayload{ID: "x", ScheduledAt: &at}, nil))
}

func TestScheduler_DoubleStartReturnsError(t *testing.T) {
	s := notification.NewScheduler()
	require.NoError(t, s.Start())
	defer s.Stop()
	assert.Error(t, s.Start())
}

func TestScheduler_GetScheduledNotification_Found(t *testing.T) {
	s := notification.NewScheduler()
	at := time.Now().Add(time.Hour)
	require.NoError(t, s.Schedule(
		&notification.NotificationPayload{ID: "find-me", ScheduledAt: &at},
		func() {},
	))
	sn, err := s.GetScheduledNotification("find-me")
	require.NoError(t, err)
	assert.Equal(t, "find-me", sn.Payload.ID)
}

func TestScheduler_GetScheduledNotification_NotFound(t *testing.T) {
	s := notification.NewScheduler()
	_, err := s.GetScheduledNotification("nope")
	assert.Error(t, err)
}
