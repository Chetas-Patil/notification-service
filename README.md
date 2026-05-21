# Notification Service

A flexible and extensible notification service written in Go that supports multiple delivery channels including Email, Slack, and In-App notifications. The service exposes an HTTP API and includes support for notification scheduling and templating.

## Features

- **Multiple Notification Channels**:
  - Email notifications via SMTP
  - Slack notifications via webhooks
  - In-app notifications with in-memory storage

- **HTTP API**: REST endpoints for sending notifications and managing templates

- **Notification Scheduling**: Schedule notifications to be sent at a future time

- **Template Support**:
  - Register and manage notification templates
  - Dynamic rendering with Go text templates
  - Two built-in default templates (`welcome`, `order_confirmation`)

- **Channel Mapping**: Map notification types to their delivery channels

- **Thread-Safe**: All components use `sync.RWMutex` for safe concurrent use

## Project Structure

```
notification-service/
├── cmd/
│   └── main.go                        # Entry point — loads config, starts HTTP server
├── config/
│   ├── config.go                      # Config loading via Viper
│   └── config.yaml                    # Default configuration
├── internal/
│   ├── api/
│   │   ├── server.go                  # HTTP server and route handlers
│   │   └── server_test.go
│   ├── app/
│   │   └── app.go                     # App bootstrap — wires service, engine, and server
│   ├── notification/
│   │   ├── types.go                   # Core types and interfaces
│   │   ├── service.go                 # Notification service
│   │   ├── email.go                   # Email channel implementation
│   │   ├── slack.go                   # Slack channel implementation
│   │   ├── inapp.go                   # In-app channel implementation
│   │   ├── scheduler.go               # Notification scheduler
│   │   ├── scheduler_test.go
│   │   ├── service_test.go
│   │   └── mocks/                     # Mockery-generated mocks
│   └── template/
│       ├── template.go                # Template engine
│       └── template_test.go
├── Makefile
├── go.mod
└── go.sum
```

## Prerequisites

- Go 1.23 or higher

## Configuration

Edit `config/config.yaml` before running:

```yaml
http_addr: ":8080"
notification:
  email:
    smtp_host: "localhost"
    smtp_port: 1025
    smtp_user: "user"
    smtp_password: "password"
    from_address: "noreply@example.com"
  slack:
    webhook_url: "https://hooks.slack.com/services/YOUR/WEBHOOK/URL"
    token: "xoxb-your-token"
    channel: "#notifications"
  inapp:
    database_url: "memory"
    max_retention: "24h"
```

## Running the Service

```bash
# Build and run
make build
./bin/notification-service

# Or run directly
go run ./cmd

# Run tests
make test
```

The server starts on the address configured in `config.yaml` (default `:8080`).

## HTTP API

### Health check

```
GET /healthz
```

### Send a notification

```
POST /notifications
Content-Type: application/json
```

**Immediate delivery:**
```json
{
  "id": "notif-001",
  "type": "inapp",
  "recipient": "user123",
  "subject": "Hello!",
  "data": {
    "message": "You have a new message"
  }
}
```

**Scheduled delivery** (set `scheduled_at` to a future time):
```json
{
  "id": "notif-002",
  "type": "email",
  "recipient": "user@example.com",
  "subject": "Reminder",
  "scheduled_at": "2025-12-01T09:00:00Z"
}
```

**Using a template** (subject and body are rendered from the template):
```json
{
  "type": "email",
  "recipient": "user@example.com",
  "template_id": "welcome",
  "template_data": {
    "AppName": "MyApp",
    "UserName": "Jane"
  }
}
```

Supported `type` values: `email`, `slack`, `inapp`.

### Get in-app notifications for a user

```
GET /notifications/{userID}
```

### Template management

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/templates` | Register a new template |
| `GET` | `/templates` | List all templates |
| `GET` | `/templates/{id}` | Get a template by ID |
| `PUT` | `/templates/{id}` | Update a template |
| `DELETE` | `/templates/{id}` | Delete a template |

**Template body:**
```json
{
  "id": "welcome",
  "name": "Welcome Email",
  "subject": "Welcome to {{.AppName}}, {{.UserName}}!",
  "body": "Hello {{.UserName}},\n\nWelcome to {{.AppName}}!",
  "channel": "email"
}
```

## Extending the Service

### Creating a Custom Channel

Implement the `Channel` interface from `internal/notification/types.go`:

```go
type CustomChannel struct{}

func (c *CustomChannel) Send(ctx context.Context, payload *notification.NotificationPayload) (*notification.NotificationResponse, error) {
    return &notification.NotificationResponse{
        Success:   true,
        MessageID: payload.ID,
        Timestamp: time.Now(),
    }, nil
}

func (c *CustomChannel) CanHandle(notificationType notification.NotificationType) bool {
    return notificationType == "custom"
}

func (c *CustomChannel) GetChannelType() notification.ChannelType {
    return "custom"
}
```

Register it in `internal/app/app.go` alongside the existing channels.

## Thread Safety

- `Service` uses `sync.RWMutex` for channel registration and mappings
- `InAppChannel` uses `sync.RWMutex` for notification storage
- `Scheduler` uses `sync.RWMutex` for scheduled notification management
- `Engine` (template) uses `sync.RWMutex` for template operations

## License

This project is provided as-is for educational and commercial purposes.
