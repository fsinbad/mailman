# Mailman - Email Management Service

Mailman is a Go-based email management service that provides APIs for managing email accounts, fetching emails via IMAP, and parsing email content.

## Features

- **Email Account Management**: Create, update, delete, and list email accounts
- **Mail Provider Support**: Pre-configured support for Gmail, Outlook, Yahoo, iCloud, and custom providers
- **IMAP Email Fetching**: Fetch emails from any IMAP server
- **Proxy Support**: Connect through SOCKS5 proxies for email fetching
- **Email Parsing**: Parse email content including text, HTML, and attachments
- **Database Storage**: Store emails and accounts in SQLite, MySQL, or PostgreSQL
- **RESTful API**: Clean REST API with Swagger documentation
- **CORS Support**: Built-in CORS middleware for frontend integration
- **Email Subscription Mode**: Real-time email monitoring with WebSocket support
- **Email Caching**: Efficient caching system for improved performance
- **Multi-folder Support**: Fetch emails from all IMAP folders

## Prerequisites

- Go 1.24.2 or higher
- SQLite, MySQL, or PostgreSQL (optional, defaults to SQLite)

## Installation

1. Clone the repository:
```bash
git clone https://github.com/seongminhwan/mailman.git
cd mailman
```

2. Install dependencies:
```bash
go mod download
```

3. Copy the environment configuration:
```bash
cp .env.example .env
```

4. Edit `.env` to configure your database and server settings.

## Running the Service

```bash
go run cmd/mailman/main.go
```

The server will start on `http://localhost:8080` by default.

## API Endpoints

### Health Check
- `GET /health` - Check if the service is running

### Email Accounts
- `POST /accounts` - Create a new email account
- `GET /accounts` - List all email accounts
- `GET /accounts/{id}` - Get a specific email account
- `PUT /accounts/{id}` - Update an email account
- `DELETE /accounts/{id}` - Delete an email account

### Email Operations
- `POST /accounts/{id}/fetch` - Fetch and store emails for an account
- `GET /accounts/{id}/emails` - Get emails for an account (with pagination)
- `GET /emails/{id}` - Get a specific email with full details

### Email Subscriptions (New)
- `POST /api/subscriptions` - Create a new email subscription
- `GET /api/subscriptions` - List all active subscriptions
- `DELETE /api/subscriptions/{id}` - Delete a subscription
- `GET /api/cache/stats` - Get email cache statistics
- `POST /api/emails/fetch-now` - Trigger immediate email fetch for all subscriptions
- `WS /ws` - WebSocket endpoint for real-time email updates

### Mail Providers
- `GET /providers` - List all available mail providers

### Legacy Endpoint
- `POST /fetch-emails` - Fetch emails (deprecated, use `/accounts/{id}/fetch`)

## Example Usage

### Create an Email Account

```bash
curl -X POST http://localhost:8080/accounts \
  -H "Content-Type: application/json" \
  -d '{
    "email_address": "user@gmail.com",
    "password": "app-specific-password",
    "mail_provider_id": 1,
    "auth_type": "password"
  }'
```

### Fetch Emails for an Account

```bash
curl -X POST http://localhost:8080/accounts/1/fetch
```

### Get Emails with Pagination

```bash
curl "http://localhost:8080/accounts/1/emails?limit=20&offset=0"
```

## Email Subscription Mode

The email subscription mode allows you to monitor email accounts in real-time and receive notifications when new emails arrive.

### Creating a Subscription

```bash
curl -X POST http://localhost:8080/api/subscriptions \
  -H "Content-Type: application/json" \
  -d '{
    "account_id": 1,
    "fetch_interval": 60
  }'
```

### WebSocket Connection

Connect to the WebSocket endpoint to receive real-time email updates:

```javascript
const ws = new WebSocket('ws://localhost:8080/ws');

ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  console.log('New email:', data);
};

// Subscribe to specific accounts
ws.send(JSON.stringify({
  type: 'subscribe',
  account_ids: [1, 2, 3]
}));
```

### Testing Subscription Features

Use the provided test scripts:

```bash
# Test REST API endpoints
./backend/scripts/test_subscription_api.sh

# Test WebSocket functionality
./backend/scripts/test_subscription_websocket.sh
```

## Using with Proxy

To fetch emails through a SOCKS5 proxy, include the proxy URL when creating an account:

```json
{
  "email_address": "user@gmail.com",
  "password": "password",
  "mail_provider_id": 1,
  "proxy": "socks5://username:password@proxy-host:1080"
}
```

## Database Schema

The service automatically creates the following tables:

- `mail_providers` - Email service provider configurations
- `email_accounts` - User email accounts
- `emails` - Fetched emails
- `attachments` - Email attachments
- `mailboxes` - IMAP mailboxes
- `email_subscriptions` - Active email monitoring subscriptions

## Development

### Project Structure

```
mailman/
├── cmd/
│   └── mailman/
│       └── main.go          # Application entry point
├── internal/
│   ├── api/
│   │   ├── handlers.go      # HTTP handlers
│   │   ├── router.go        # Route definitions
│   │   └── websocket.go     # WebSocket handler
│   ├── config/
│   │   └── config.go        # Configuration management
│   ├── database/
│   │   └── database.go      # Database connection and migrations
│   ├── models/
│   │   └── models.go        # Data models
│   ├── repository/
│   │   ├── email.go         # Email repository
│   │   ├── email_account.go # Account repository
│   │   └── mail_provider.go # Provider repository
│   └── services/
│       ├── fetcher.go       # Email fetching service
│       ├── parser.go        # Email parsing service
│       ├── email_subscription_manager.go # Subscription management
│       ├── email_fetch_scheduler.go      # Scheduled fetching
│       ├── email_cache.go               # Email caching
│       └── email_worker_pool.go         # Worker pool for concurrent fetching
├── .env.example             # Example environment configuration
├── go.mod                   # Go module file
└── README.md               # This file
```

### Adding a New Mail Provider

1. Add the provider configuration to the `SeedDefaultProviders` method in `internal/repository/mail_provider.go`
2. The provider will be automatically available after restarting the service

## Testing

### Manual Testing with cURL

See the test files `test_account.json` and `test_account_with_proxy.json` for example account configurations.

```bash
# Test email fetching
curl -X POST http://localhost:8080/fetch-emails \
  -H "Content-Type: application/json" \
  -d @test_account.json
```

### End-to-End Testing

An example end-to-end test is available at `backend/examples/subscription_e2e_test.go.example`. To use it:

```bash
cp backend/examples/subscription_e2e_test.go.example backend/examples/subscription_e2e_test.go
# Edit the file to add your test configuration
go run backend/examples/subscription_e2e_test.go
```

## Security Notes

- Store email passwords securely (consider using environment variables or a secrets manager)
- Use app-specific passwords for Gmail and other providers that support them
- Enable SSL/TLS for database connections in production
- Consider implementing authentication for the API endpoints
- WebSocket connections should be secured with WSS in production

## License

This project is licensed under the MIT License.
