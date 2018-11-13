package main

import (
	"gopkg.in/gomail.v2"
)

type Mail interface {
	Send(mailTo, host, body string, port int) error
}

type MailClient struct{}

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
