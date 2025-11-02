package smtp

import "MySMTP/mail"

// MailHandler is a function that processes a completed email
// Return an error to reject the email, or nil to accept it
type MailHandler func(m *mail.Mail) error

// EmailExistsChecker is a function that checks if an email address exists
// Return true if the email exists, false otherwise
// Default implementation returns false
type EmailExistsChecker func(email string) bool

// Handlers holds all the callback handlers for the SMTP server
type Handlers struct {
	MailHandler        MailHandler
	EmailExistsChecker EmailExistsChecker
}

// NewHandlers creates a new Handlers instance with default implementations
func NewHandlers() *Handlers {
	return &Handlers{
		MailHandler:        nil, // No handler by default (accept all)
		EmailExistsChecker: defaultEmailExistsChecker,
	}
}

// defaultEmailExistsChecker is the default implementation that returns false
func defaultEmailExistsChecker(email string) bool {
	return false
}

