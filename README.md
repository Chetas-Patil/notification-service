# Notification Service

A flexible and extensible notification service written in Go that supports multiple delivery channels including Email, Slack, and In-App notifications. This service includes support for notification scheduling, templating, and channel mapping.

## Features

- **Multiple Notification Channels**: 
  - Email notifications via SMTP
  - Slack notifications via webhooks
  - In-app notifications with storage

- **Notification Scheduling**: Schedule notifications to be sent at specific times

- **Template Support**: 
  - Predefined notification templates
  - Dynamic template rendering with custom data
  - Support for Go text templates

- **Channel Mapping**: Map notification types to their delivery channels

- **Thread-Safe**: All components are designed to be thread-safe for concurrent usage

## Project Structure

```
notification-service/
├── cmd/
│   └── main.go                 # Example usage and demo
├── pkg/
│   ├── notification/
│   │   ├── types.go            # Core types and interfaces
│   │   ├── service.go          # Main notification service
│   │   ├── email.go            # Email channel implementation
│   │   ├── slack.go            # Slack channel implementation
│   │   ├── inapp.go            # In-app channel implementation
│   │   └── scheduler.go        # Notification scheduler
│   ├── template/
│   │   └── template.go         # Template engine
│   └── scheduler/
│       └── scheduler.go        # Scheduler implementation
├── go.mod                      # Go module file
└── README.md                   # This file
```

## Installation

### Prerequisites
- Go 1.21 or higher

### Setup

1. Clone or navigate to the project directory:
```bash
cd notification-service
```

2. Initialize dependencies:
```bash
go mod tidy
```

## Usage

### Basic Setup

```go
package main

import (
    "context"
    "notification-service/pkg/notification"
)

func main() {
    // Create a new service
    service := notification.NewService()
    
    // Configure and register channels
    emailConfig := notification.EmailConfig{
        SMTPHost:     "smtp.example.com",
        SMTPPort:     587,
        SMTPUser:     "user@example.com",
        SMTPPassword: "password",
        FromAddress:  "noreply@example.com",
    }
    
    emailChannel := notification.NewEmailChannel(emailConfig)
    service.RegisterChannel(emailChannel)
    
    // Register channel mappings
    service.RegisterChannelMapping(
        notification.EmailNotification,
        notification.EmailChannel,
    )
    
    // Start the scheduler
    service.Start()
    defer service.Stop()
    
    // Send a notification
    payload := &notification.NotificationPayload{
        ID:        "notif-001",
        Type:      notification.EmailNotification,
        Recipient: "user@example.com",
        Subject:   "Hello!",
        Data: map[string]interface{}{
            "message": "This is a test notification",
        },
    }
    
    ctx := context.Background()
    response, err := service.Send(ctx, payload)
    if err != nil {
        // Handle error
    }
}
```

### Scheduling Notifications

```go
import "time"

// Schedule a notification for later
scheduledTime := time.Now().Add(1 * time.Hour)

payload := &notification.NotificationPayload{
    ID:          "notif-scheduled",
    Type:        notification.InAppNotification,
    Recipient:   "user123",
    Subject:     "Scheduled notification",
    ScheduledAt: &scheduledTime,
    Data: map[string]interface{}{
        "message": "This will be sent in 1 hour",
    },
}

err := service.SendScheduled(ctx, payload)
```

### Using Templates

```go
import "notification-service/pkg/template"

// Create template engine
engine := template.NewEngine()

// Register a template
tmpl := &template.Template{
    ID:      "welcome",
    Name:    "Welcome Email",
    Subject: "Welcome to {{.AppName}}, {{.UserName}}!",
    Body:    "Hello {{.UserName}},\n\nWelcome to {{.AppName}}!",
    Channel: "email",
}

engine.RegisterTemplate(tmpl)

// Render template with data
rendered, err := engine.RenderTemplate("welcome", map[string]interface{}{
    "AppName":  "MyApp",
    "UserName": "John Doe",
})

if err != nil {
    // Handle error
}

// Use rendered template in notification
payload.Subject = rendered.Subject
payload.Data["body"] = rendered.Body
```

### Multi-Channel Delivery

```go
// Send to all configured channels for a notification type
responses, err := service.SendToAllChannels(ctx, payload)

for _, response := range responses {
    if response.Success {
        // Handle success
    } else {
        // Handle error
    }
}
```

## Supported Notification Types

### 1. Email Notifications
Sent via SMTP server. Requires SMTP configuration.

```go
payload := &notification.NotificationPayload{
    Type:      notification.EmailNotification,
    Recipient: "user@example.com",
    Subject:   "Email Subject",
}
```

### 2. Slack Notifications
Sent via Slack webhook. Supports rich formatting with blocks.

```go
payload := &notification.NotificationPayload{
    Type:      notification.SlackNotification,
    Recipient: "#general",  // Channel name or user ID
    Subject:   "Slack Message",
}
```

### 3. In-App Notifications
Stored in memory for user access. Thread-safe with read/write operations.

```go
payload := &notification.NotificationPayload{
    Type:      notification.InAppNotification,
    Recipient: "user123",
    Subject:   "In-App Message",
}

// Later, retrieve notifications
inAppChannel := service.GetChannel(notification.InAppChannel).(notification.InAppChannel)
notifications := inAppChannel.GetNotifications("user123")
```

## API Reference

### Service Methods

- `RegisterChannel(channel Channel) error` - Register a new notification channel
- `RegisterChannelMapping(notificationType NotificationType, channels ...ChannelType) error` - Map notification types to channels
- `Send(ctx context.Context, payload *NotificationPayload) (*NotificationResponse, error)` - Send notification immediately
- `SendToAllChannels(ctx context.Context, payload *NotificationPayload) ([]*NotificationResponse, error)` - Send to all configured channels
- `SendScheduled(ctx context.Context, payload *NotificationPayload) error` - Schedule a notification
- `GetChannel(channelType ChannelType) (Channel, error)` - Get a registered channel
- `Start() error` - Start the scheduler
- `Stop()` - Stop the scheduler

### Template Engine Methods

- `RegisterTemplate(tmpl *Template) error` - Register a new template
- `GetTemplate(id string) (*Template, error)` - Get a template by ID
- `RenderTemplate(templateID string, data map[string]interface{}) (*RenderedTemplate, error)` - Render a template
- `ListTemplates() []*Template` - List all templates
- `DeleteTemplate(id string) error` - Delete a template
- `UpdateTemplate(id string, tmpl *Template) error` - Update an existing template

## Running the Demo

To run the example demonstration:

```bash
go run cmd/main.go
```

This will demonstrate:
1. Immediate in-app notifications
2. Template rendering
3. Notification scheduling
4. Multi-channel delivery
5. Notification retrieval

## Error Handling

All methods return errors following Go conventions. Always check errors:

```go
if err := service.RegisterChannel(channel); err != nil {
    log.Printf("Error registering channel: %v", err)
    // Handle error appropriately
}
```

## Extending the Service

### Creating a Custom Channel

```go
type CustomChannel struct {
    // Your fields
}

func (c *CustomChannel) Send(ctx context.Context, payload *notification.NotificationPayload) (*notification.NotificationResponse, error) {
    // Your implementation
    return &notification.NotificationResponse{
        Success:   true,
        MessageID: payload.ID,
        Timestamp: time.Now(),
    }, nil
}

func (c *CustomChannel) CanHandle(notificationType notification.NotificationType) bool {
    // Your logic
    return true
}

func (c *CustomChannel) GetChannelType() notification.ChannelType {
    return "custom"
}

// Register it
service.RegisterChannel(customChannel)
```

## Thread Safety

- The `Service` uses `sync.RWMutex` for thread-safe channel registration and mappings
- The `InAppChannel` uses `sync.RWMutex` for thread-safe notification storage
- The `Scheduler` uses `sync.RWMutex` for thread-safe scheduled notification management
- The `Template Engine` uses `sync.RWMutex` for thread-safe template operations

## Best Practices

1. **Always Start and Stop the Scheduler**: Call `service.Start()` and `service.Stop()` properly
2. **Use Context**: Pass appropriate context for cancellation support
3. **Validate Configuration**: Ensure all required configuration is provided before registering channels
4. **Handle Errors**: Always check and handle errors from service methods
5. **Template Validation**: Validate template syntax when registering templates
6. **Resource Cleanup**: Use defer to ensure proper cleanup of resources

## Future Enhancements

- Webhook callbacks for delivery notifications
- Retry logic for failed deliveries
- Notification priority levels
- Rate limiting
- Delivery status tracking
- Database persistence for in-app notifications
- Integration with more notification services (SMS, Push Notifications, etc.)

## License

This project is provided as-is for educational and commercial purposes.
