// Package smtp provides SMTP server and client implementations for Go.
//
// This package allows you to create SMTP servers and clients with customizable
// handlers for processing emails and verifying email addresses.
//
// Example server usage:
//
//	import (
//		"MySMTP/smtp"
//		"MySMTP/mail"
//		"net"
//	)
//
//	// Create handlers
//	handlers := smtp.NewHandlers()
//
//	// Set mail handler (called when email is complete)
//	handlers.MailHandler = func(m *mail.Mail) error {
//		// Process the email
//		fmt.Printf("Received email from %s\n", m.GetFrom())
//		return nil // Accept the email
//	}
//
//	// Set email existence checker (default returns false)
//	handlers.EmailExistsChecker = func(email string) bool {
//		// Check if email exists in your database
//		return checkEmailInDatabase(email)
//	}
//
//	// Create server connection with handlers
//	conn, _ := net.Dial("tcp", "localhost:2525")
//	connPtr := &conn
//	smtp.NewServerConnWithHandlers(connPtr, config, handlers)
package smtp

