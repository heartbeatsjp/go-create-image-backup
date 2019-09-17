package main

import (
	"gopkg.in/gomail.v2"
)

// Mail provides methords for send mail
type Mail interface {
	Send(mailTo, host, body string, port int) error
}

// MailClient implements Mail interface
type MailClient struct{}

// Send sends email
func (m *MailClient) Send(mailFrom, mailTo, host, body string, port int) error {
	message := gomail.NewMessage()
	message.SetHeader("From", mailFrom)
	message.SetHeader("To", mailTo)
	message.SetHeader("Subject", "Backup failed")
	message.SetBody("text/plain", body)
	d := gomail.Dialer{Host: host, Port: port}

	if err := d.DialAndSend(message); err != nil {
		return err
	}
	return nil
}
