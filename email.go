package main

import (
	"net/smtp"
	"os"

	log "github.com/sirupsen/logrus"
)

func sendmail(to string, subject string, body string) error {
	// Get the email and password from environment variables.
	email := os.Getenv("EMAIL")
	password := os.Getenv("PASSWORD")

	if email == "" || password == "" {
		log.Error("Please set the EMAIL and PASSWORD environment variables.")
		log.Error("Error sending email")
		return nil
	}

	// Set up authentication information.
	smtpServer := "smtp.gmail.com"
	auth := smtp.PlainAuth("", email, password, smtpServer)

	// Compose the message.
	msg := []byte("To: " + to + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"\r\n" +
		body + "\r\n")

	// Send the message.
	err := smtp.SendMail(smtpServer+":587", auth, email, []string{to}, msg)
	if err != nil {
		log.WithError(err).Error("Error sending email")
		return err
	}

	log.Info("Email sent successfully!")
	return nil
}
