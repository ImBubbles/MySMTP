# MySMTP

SMTP server and client implementation in Go.

## Installation

To use this library in your Go project:

```bash
go get github.com/yourusername/MySMTP@latest
```

Replace `yourusername` with your GitHub username. See [USAGE.md](USAGE.md) for detailed usage instructions.

## Configuration

The application uses environment variables for configuration. You can either:
1. Set environment variables directly
2. Create a `.env` file in the project root (see `env.example` for template)

### Available Configuration Variables

- `SMTP_SERVER_HOSTNAME` - Server hostname (default: `localhost`)
- `SMTP_SERVER_PORT` - Server port (default: `2525`)
- `SMTP_SERVER_ADDRESS` - Server bind address (default: `0.0.0.0`)
- `SMTP_SERVER_DOMAIN` - Server domain for EHLO responses (default: `localhost`)
- `SMTP_CLIENT_HOSTNAME` - Client hostname for EHLO (default: `localhost`)
- `SMTP_RELAY` - Enable relay mode (default: `false`)
- `SMTP_REQUIRE_TLS` - Require TLS connections (default: `false`)

### Example `.env` file

Copy `env.example` to `.env` and modify as needed:

```bash
cp env.example .env
```

## Usage

Run the server:
```bash
go run main.go
```

The server will load configuration from environment variables or `.env` file and print the configuration on startup.