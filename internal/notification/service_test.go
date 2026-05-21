package notification_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"notification-service/internal/notification"
	"notification-service/internal/notification/mocks"
)

// matchPayload returns a testify matcher that checks ID and Type only so that
// fields mutated by the service (CreatedAt, rendered Subject, etc.) don't
// cause spurious failures.
func matchPayload(id string, typ notification.NotificationType) interface{} {
	return mock.MatchedBy(func(p *notification.NotificationPayload) bool {
		return p.ID == id && p.Type == typ
	})
}

func newService(t *testing.T) *notification.Service {
	t.Helper()
	svc := notification.NewService()
	require.NoError(t, svc.Start())
	t.Cleanup(svc.Stop)
	return svc
}

func mockChannel(t *testing.T, channelType notification.ChannelType) *mocks.MockChannel {
	t.Helper()
	ch := mocks.NewMockChannel(t)
	ch.EXPECT().GetChannelType().Return(channelType).Maybe()
	return ch
}

func okResponse(id string) *notification.NotificationResponse {
	return &notification.NotificationResponse{Success: true, MessageID: id, Timestamp: time.Now()}
}

// --- Send ----------------------------------------------------------------

func TestService_Send_RoutesToMappedChannel(t *testing.T) {
	svc := newService(t)
	ch := mockChannel(t, "email")
	ch.EXPECT().Send(context.Background(), matchPayload("n-1", notification.EmailNotification)).
		Return(okResponse("n-1"), nil)

	require.NoError(t, svc.RegisterChannel(ch))
	require.NoError(t, svc.RegisterChannelMapping(notification.EmailNotification, "email"))

	resp, err := svc.Send(context.Background(), &notification.NotificationPayload{
		ID: "n-1", Type: notification.EmailNotification,
	})
	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.Equal(t, "n-1", resp.MessageID)
}

func TestService_Send_NilPayloadReturnsError(t *testing.T) {
	_, err := newService(t).Send(context.Background(), nil)
	require.Error(t, err)
}

func TestService_Send_UnknownTypeReturnsError(t *testing.T) {
	_, err := newService(t).Send(context.Background(), &notification.NotificationPayload{
		ID: "x", Type: "unknown",
	})
	require.Error(t, err)
}

func TestService_Send_ChannelErrorPropagated(t *testing.T) {
	svc := newService(t)
	ch := mockChannel(t, "failing")
	ch.EXPECT().Send(context.Background(), matchPayload("e", "fail_type")).
		Return(nil, errors.New("smtp down"))

	require.NoError(t, svc.RegisterChannel(ch))
	require.NoError(t, svc.RegisterChannelMapping("fail_type", "failing"))

	resp, err := svc.Send(context.Background(), &notification.NotificationPayload{
		ID: "e", Type: "fail_type",
	})
	require.NoError(t, err) // service-level Send doesn't error — failure is in the response
	assert.False(t, resp.Success)
	assert.Equal(t, "smtp down", resp.ErrorMsg)
}

// --- SendToAllChannels ---------------------------------------------------

func TestService_SendToAllChannels_FansOutToEveryMappedChannel(t *testing.T) {
	svc := newService(t)
	chA := mockChannel(t, "a")
	chB := mockChannel(t, "b")
	chA.EXPECT().Send(context.Background(), matchPayload("fan", "multi")).Return(okResponse("fan"), nil)
	chB.EXPECT().Send(context.Background(), matchPayload("fan", "multi")).Return(okResponse("fan"), nil)

	require.NoError(t, svc.RegisterChannel(chA))
	require.NoError(t, svc.RegisterChannel(chB))
	require.NoError(t, svc.RegisterChannelMapping("multi", "a", "b"))

	responses, err := svc.SendToAllChannels(context.Background(), &notification.NotificationPayload{
		ID: "fan", Type: "multi",
	})
	require.NoError(t, err)
	require.Len(t, responses, 2)
	assert.True(t, responses[0].Success)
	assert.True(t, responses[1].Success)
}

// --- Template rendering --------------------------------------------------

func TestService_Send_AppliesTemplate(t *testing.T) {
	svc := newService(t)

	renderer := mocks.NewMockTemplateRenderer(t)
	renderer.EXPECT().
		Render("welcome", map[string]interface{}{"Name": "Ada"}).
		Return("Hi Ada", "Welcome Ada", nil)
	svc.SetTemplateRenderer(renderer)

	ch := mockChannel(t, "inapp")
	ch.EXPECT().
		Send(context.Background(), mock.MatchedBy(func(p *notification.NotificationPayload) bool {
			body, _ := p.Data["body"].(string)
			return p.ID == "t-1" && p.Subject == "Hi Ada" && body == "Welcome Ada"
		})).
		Return(okResponse("t-1"), nil)

	require.NoError(t, svc.RegisterChannel(ch))
	require.NoError(t, svc.RegisterChannelMapping(notification.InAppNotification, "inapp"))

	_, err := svc.Send(context.Background(), &notification.NotificationPayload{
		ID:           "t-1",
		Type:         notification.InAppNotification,
		TemplateID:   "welcome",
		TemplateData: map[string]interface{}{"Name": "Ada"},
	})
	require.NoError(t, err)
}

func TestService_Send_TemplateRendererErrorReturnsError(t *testing.T) {
	svc := newService(t)

	renderer := mocks.NewMockTemplateRenderer(t)
	renderer.EXPECT().Render("bad", map[string]interface{}(nil)).
		Return("", "", errors.New("template not found"))
	svc.SetTemplateRenderer(renderer)

	ch := mockChannel(t, "inapp")
	require.NoError(t, svc.RegisterChannel(ch))
	require.NoError(t, svc.RegisterChannelMapping(notification.InAppNotification, "inapp"))

	_, err := svc.Send(context.Background(), &notification.NotificationPayload{
		ID: "x", Type: notification.InAppNotification, TemplateID: "bad",
	})
	require.Error(t, err)
}

func TestService_Send_NoRendererWithTemplateIDReturnsError(t *testing.T) {
	svc := newService(t)
	ch := mockChannel(t, "inapp")
	require.NoError(t, svc.RegisterChannel(ch))
	require.NoError(t, svc.RegisterChannelMapping(notification.InAppNotification, "inapp"))

	_, err := svc.Send(context.Background(), &notification.NotificationPayload{
		ID: "x", Type: notification.InAppNotification, TemplateID: "welcome",
	})
	require.Error(t, err)
}

// --- Scheduling ----------------------------------------------------------

func TestService_SendScheduled_FiresAfterDelay(t *testing.T) {
	svc := newService(t)

	fired := make(chan struct{}, 1)
	ch := mockChannel(t, "inapp")
	ch.EXPECT().
		Send(context.Background(), matchPayload("s-1", notification.InAppNotification)).
		RunAndReturn(func(_ context.Context, _ *notification.NotificationPayload) (*notification.NotificationResponse, error) {
			fired <- struct{}{}
			return okResponse("s-1"), nil
		})

	require.NoError(t, svc.RegisterChannel(ch))
	require.NoError(t, svc.RegisterChannelMapping(notification.InAppNotification, "inapp"))

	at := time.Now().Add(50 * time.Millisecond)
	require.NoError(t, svc.SendScheduled(context.Background(), &notification.NotificationPayload{
		ID:          "s-1",
		Type:        notification.InAppNotification,
		ScheduledAt: &at,
	}))

	select {
	case <-fired:
	case <-time.After(3 * time.Second):
		t.Fatal("scheduled notification was not fired within deadline")
	}
}

func TestService_SendScheduled_NoScheduledAtReturnsError(t *testing.T) {
	err := newService(t).SendScheduled(context.Background(), &notification.NotificationPayload{
		ID: "x", Type: "inapp",
	})
	require.Error(t, err)
}

// --- Channel registration ------------------------------------------------

func TestService_RegisterChannel_NilReturnsError(t *testing.T) {
	assert.Error(t, newService(t).RegisterChannel(nil))
}

func TestService_RegisterChannelMapping_EmptyChannelsReturnsError(t *testing.T) {
	assert.Error(t, newService(t).RegisterChannelMapping("email"))
}
