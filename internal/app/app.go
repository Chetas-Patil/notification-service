package app

import (
	"context"
	"fmt"
	"log"

	"notification-service/internal/api"
	"notification-service/config"
	"notification-service/internal/notification"
	"notification-service/internal/template"
)

type App struct {
	Notification *notification.Service
	Template     *template.Engine
	HTTP         *api.Server
	Config       *config.Config
}

func NewApp(cfg *config.Config) *App {
	svc := notification.NewService()
	engine := template.NewEngine()
	return &App{
		Notification: svc,
		Template:     engine,
		HTTP:         api.NewServer(cfg.HTTPAddr, svc, engine),
		Config:       cfg,
	}
}

func (a *App) Init() error {
	a.Notification.SetTemplateRenderer(a.Template)

	channels := []notification.Channel{
		notification.NewEmailChannel(a.Config.Email),
		notification.NewSlackChannel(a.Config.Slack),
		notification.NewInAppChannel(a.Config.InApp),
	}
	for _, ch := range channels {
		if err := a.Notification.RegisterChannel(ch); err != nil {
			return fmt.Errorf("register channel %s: %w", ch.GetChannelType(), err)
		}
	}

	if err := a.Notification.RegisterChannelMapping(notification.EmailNotification, notification.Email); err != nil {
		return err
	}
	if err := a.Notification.RegisterChannelMapping(notification.SlackNotification, notification.Slack); err != nil {
		return err
	}
	if err := a.Notification.RegisterChannelMapping(notification.InAppNotification, notification.InApp); err != nil {
		return err
	}

	if err := a.registerDefaultTemplates(); err != nil {
		return err
	}

	return a.Notification.Start()
}

func (a *App) registerDefaultTemplates() error {
	templates := []*template.Template{
		{
			ID:      "welcome",
			Name:    "Welcome Email",
			Subject: "Welcome to {{.AppName}}, {{.UserName}}!",
			Body:    "Hello {{.UserName}},\n\nWelcome to {{.AppName}}!\n\nBest regards,\nThe Team",
			Channel: "email",
		},
		{
			ID:      "order_confirmation",
			Name:    "Order Confirmation",
			Subject: "Order #{{.OrderID}} Confirmed",
			Body:    "Thank you for your order! Order ID: {{.OrderID}}\nTotal: ${{.Total}}\n\nEstimated Delivery: {{.DeliveryDate}}",
			Channel: "email",
		},
	}
	for _, t := range templates {
		if err := a.Template.RegisterTemplate(t); err != nil {
			return fmt.Errorf("register template %s: %w", t.ID, err)
		}
	}
	return nil
}

// Serve blocks until the HTTP server returns. The caller is responsible for
// invoking Shutdown when a stop signal arrives.
func (a *App) Serve() error {
	return a.HTTP.Start()
}

func (a *App) Shutdown(ctx context.Context) {
	if err := a.HTTP.Shutdown(ctx); err != nil {
		log.Printf("http shutdown: %v", err)
	}
	a.Notification.Stop()
}
