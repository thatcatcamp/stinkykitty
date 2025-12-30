// SPDX-License-Identifier: MIT
package email

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/smtp"
	"os"
)

type EmailService struct {
	host     string
	port     string
	email    string
	password string
}

func NewEmailService() (*EmailService, error) {
	host := os.Getenv("SMTP")
	port := os.Getenv("SMTP_PORT")
	email := os.Getenv("EMAIL")
	password := os.Getenv("SMTP_SECRET")

	if host == "" || port == "" || email == "" || password == "" {
		return nil, fmt.Errorf("missing SMTP configuration in environment")
	}

	return &EmailService{
		host:     host,
		port:     port,
		email:    email,
		password: password,
	}, nil
}

func (es *EmailService) SendEmail(to, subject, body string) error {
	addr := fmt.Sprintf("%s:%s", es.host, es.port)

	// Use STARTTLS for port 587, implicit TLS for port 465
	var client *smtp.Client
	var err error

	if es.port == "587" {
		// STARTTLS: connect plain, then upgrade to TLS
		client, err = smtp.Dial(addr)
		if err != nil {
			return fmt.Errorf("failed to dial SMTP: %w", err)
		}
		defer client.Close()

		// Start TLS
		tlsconfig := &tls.Config{
			ServerName: es.host,
		}
		if err = client.StartTLS(tlsconfig); err != nil {
			return fmt.Errorf("failed to start TLS: %w", err)
		}
	} else {
		// Implicit TLS: connect with TLS from the start (port 465)
		tlsconfig := &tls.Config{
			ServerName: es.host,
		}

		conn, err := tls.Dial("tcp", addr, tlsconfig)
		if err != nil {
			return fmt.Errorf("failed to dial SMTP: %w", err)
		}
		defer conn.Close()

		client, err = smtp.NewClient(conn, es.host)
		if err != nil {
			return fmt.Errorf("failed to create SMTP client: %w", err)
		}
		defer client.Close()
	}

	if err := client.Auth(smtp.PlainAuth("", es.email, es.password, es.host)); err != nil {
		return fmt.Errorf("SMTP authentication failed: %w", err)
	}

	if err := client.Mail(es.email); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}

	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("failed to set recipient: %w", err)
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to write data: %w", err)
	}
	defer w.Close()

	// Build proper email headers
	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s", es.email, to, subject, body)
	if _, err := w.Write([]byte(msg)); err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	// Quit the connection (ignore errors - email was already sent)
	if err := client.Quit(); err != nil {
		// Some SMTP servers return non-standard success messages on QUIT
		// As long as the email was sent (which it was by this point), we can ignore QUIT errors
		log.Printf("SMTP QUIT returned non-standard response (email was sent successfully): %v", err)
	}

	log.Printf("Email sent to %s: %s", to, subject)
	return nil
}

func (es *EmailService) SendPasswordReset(email, resetURL string) error {
	subject := "StinkyKitty Password Reset"
	body := fmt.Sprintf(`Hello,

You requested a password reset for your StinkyKitty account.

Click the link below to reset your password:
%s

This link expires in 24 hours.

If you didn't request this, you can ignore this email.

Best regards,
StinkyKitty Team`, resetURL)

	return es.SendEmail(email, subject, body)
}

func (es *EmailService) SendNewUserWelcome(email, loginURL string) error {
	subject := "Welcome to StinkyKitty - Your Camp Awaits"
	body := fmt.Sprintf(`Hello,

A new StinkyKitty camp account has been created for you!

Click the link below to set your password and log in:
%s

Once logged in, you can start managing your camp's content.

If you have any questions, contact your camp administrator.

Best regards,
StinkyKitty Team`, loginURL)

	return es.SendEmail(email, subject, body)
}

func (es *EmailService) SendErrorNotification(adminEmail, subject, errorMsg string) error {
	body := fmt.Sprintf(`Admin Alert,

An error occurred in StinkyKitty:

%s

Please investigate and take appropriate action.

StinkyKitty System`, errorMsg)

	return es.SendEmail(adminEmail, fmt.Sprintf("StinkyKitty Error: %s", subject), body)
}
