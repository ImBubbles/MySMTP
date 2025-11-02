# Using MySMTP as a Go Library

This guide shows you how to use MySMTP as a dependency in your Go projects from GitHub.

## Step 1: Prepare Your GitHub Repository

1. **Update `go.mod`** to use your GitHub repository path:

```bash
# Change from:
module MySMTP

# To (replace with your GitHub username and repo):
module github.com/yourusername/MySMTP
```

2. **Push to GitHub**:
```bash
git init
git add .
git commit -m "Initial commit"
git remote add origin https://github.com/yourusername/MySMTP.git
git push -u origin main
```

3. **Tag a release** (recommended):
```bash
git tag v1.0.0
git push origin v1.0.0
```

## Step 2: Use in Your New Project

### Create a New Project

```bash
mkdir my-email-app
cd my-email-app
go mod init my-email-app
```

### Add MySMTP as a Dependency

```bash
go get github.com/yourusername/MySMTP@latest
# Or for a specific version:
go get github.com/yourusername/MySMTP@v1.0.0
```

## Step 3: Example Usage

### Example 1: Creating an SMTP Server

Create `main.go`:

```go
package main

import (
	"fmt"
	"log"
	"net"
	"os"

	"github.com/yourusername/MySMTP/config"
	"github.com/yourusername/MySMTP/mail"
	"github.com/yourusername/MySMTP/server"
	"github.com/yourusername/MySMTP/smtp"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Error loading configuration: %v\n", err)
	}

	// Print configuration
	cfg.PrintConfig()

	// Create custom handlers
	handlers := smtp.NewHandlers()

	// Override mail handler (called when email is complete)
	handlers.MailHandler = func(m *mail.Mail) error {
		fmt.Printf("Received email from: %s\n", m.GetFrom())
		fmt.Printf("Subject: %s\n", m.GetSubject())
		fmt.Printf("To: %v\n", m.GetTo())
		fmt.Printf("Body:\n%s\n", m.GetData())
		
		// Save to database, forward, etc.
		// Return error to reject, nil to accept
		return nil
	}

	// Override email existence checker (default returns false)
	handlers.EmailExistsChecker = func(email string) bool {
		// Check if email exists in your database
		// Example: return checkEmailInDatabase(email)
		// For now, accept all emails
		return true
	}

	// Set default handlers for all connections
	server.SetDefaultHandlers(handlers)

	// Create and start SMTP server
	socket := server.NewServer(cfg.ServerAddress, cfg.ServerPort)
	server.Listen(socket, cfg)
}
```

### Example 2: Sending an Email (Client)

```go
package main

import (
	"fmt"
	"log"
	"net"

	"github.com/yourusername/MySMTP/mail"
	"github.com/yourusername/MySMTP/smtp"
)

func main() {
	// Create email
	email := mail.NewBlankMail()
	email.SetFrom("sender@example.com")
	email.AppendTo("recipient@example.com")
	email.SetSubject("Test Email")
	email.SetData("This is a test email body.\n\nRegards,\nSender")

	// Connect to SMTP server
	conn, err := net.Dial("tcp", "localhost:2525")
	if err != nil {
		log.Fatalf("Failed to connect: %v\n", err)
	}
	defer conn.Close()

	// Send email
	clientConn := smtp.NewClientConn(conn, *email)
	fmt.Println("Email sent successfully!")
}
```

### Example 3: Using with Custom Handlers

```go
package main

import (
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/yourusername/MySMTP/config"
	"github.com/yourusername/MySMTP/mail"
	"github.com/yourusername/MySMTP/server"
	"github.com/yourusername/MySMTP/smtp"
)

func main() {
	cfg, _ := config.LoadConfig()

	// Custom handlers with business logic
	handlers := smtp.NewHandlers()

	// Reject emails from spam domains
	spamDomains := []string{"spam.com", "malicious.com"}
	handlers.MailHandler = func(m *mail.Mail) error {
		from := m.GetFrom()
		for _, domain := range spamDomains {
			if strings.Contains(from, domain) {
				return errors.New("spam domain rejected")
			}
		}
		
		// Process valid email
		fmt.Printf("Processing email from %s\n", from)
		return nil
	}

	// Check if recipient email exists
	handlers.EmailExistsChecker = func(email string) bool {
		// Your database check logic here
		validEmails := []string{"user@example.com", "admin@example.com"}
		for _, valid := range validEmails {
			if email == valid {
				return true
			}
		}
		return false
	}

	server.SetDefaultHandlers(handlers)

	socket := server.NewServer(cfg.ServerAddress, cfg.ServerPort)
	server.Listen(socket, cfg)
}
```

### Example 4: Using Email Verification

```go
package main

import (
	"github.com/yourusername/MySMTP/config"
	"github.com/yourusername/MySMTP/server"
	"github.com/yourusername/MySMTP/smtp"
	"github.com/yourusername/MySMTP/util/verify"
)

func main() {
	cfg, _ := config.LoadConfig()

	// Create email verifier
	verifier := verify.NewEmailVerifier()
	verifier.SetCheckFormat(true)      // Enable format checking
	verifier.SetCheckMX(true)          // Enable MX record checking
	verifier.SetAllowList([]string{"example.com", "gmail.com"}) // Only allow these domains
	verifier.SetBlockList([]string{"spam.com"}) // Block these domains

	// Note: Sender verification is applied automatically
	// To customize, you would need to access ServerConn after creation

	socket := server.NewServer(cfg.ServerAddress, cfg.ServerPort)
	server.Listen(socket, cfg)
}
```

## Step 4: Update Imports After Publishing

Once your code is on GitHub, update all imports in your MySMTP repository:

```bash
# In your MySMTP repository
find . -name "*.go" -type f -exec sed -i 's|MySMTP/|github.com/yourusername/MySMTP/|g' {} +
```

Or manually replace:
- `"MySMTP/config"` → `"github.com/yourusername/MySMTP/config"`
- `"MySMTP/mail"` → `"github.com/yourusername/MySMTP/mail"`
- `"MySMTP/smtp"` → `"github.com/yourusername/MySMTP/smtp"`
- etc.

## Step 5: Available Packages

- `github.com/yourusername/MySMTP/smtp` - SMTP server and client
- `github.com/yourusername/MySMTP/mail` - Mail structures
- `github.com/yourusername/MySMTP/config` - Configuration management
- `github.com/yourusername/MySMTP/server` - Server wrapper
- `github.com/yourusername/MySMTP/util/verify` - Email verification utilities

## Quick Start

```bash
# In your new project directory
go mod init my-project
go get github.com/yourusername/MySMTP@latest

# Create your main.go file with one of the examples above
# Run it
go run main.go
```

## Configuration

Create a `.env` file or set environment variables:

```bash
SMTP_SERVER_HOSTNAME=localhost
SMTP_SERVER_PORT=2525
SMTP_SERVER_ADDRESS=0.0.0.0
SMTP_SERVER_DOMAIN=localhost
SMTP_CLIENT_HOSTNAME=localhost
SMTP_RELAY=false
SMTP_REQUIRE_TLS=false
```

## Features

- ✅ SMTP Server (receiving emails)
- ✅ SMTP Client (sending emails)
- ✅ Custom mail handlers (override email processing)
- ✅ Email existence checking (override recipient validation)
- ✅ Sender verification
- ✅ TLS/STARTTLS support
- ✅ Email verification utilities
- ✅ Configuration via environment variables

## Need Help?

See the main [README.md](README.md) for more details and examples.

