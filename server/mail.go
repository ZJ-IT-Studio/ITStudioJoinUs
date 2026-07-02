package server

import (
	"fmt"
	"log"
	"net/smtp"
	"strings"
)

func (a *App) sendMail(to, subject, body string) error {
	if a.Config.SMTPHost == "" {
		log.Printf("DEV MAIL to=%s subject=%s body=%s", to, subject, body)
		return nil
	}
	fromAddr := a.Config.SMTPFrom
	if i := strings.Index(fromAddr, "<"); i >= 0 {
		if j := strings.Index(fromAddr, ">"); j > i {
			fromAddr = fromAddr[i+1 : j]
		}
	}
	msg := []byte(fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s", a.Config.SMTPFrom, to, subject, body))
	var auth smtp.Auth
	if a.Config.SMTPUser != "" {
		auth = smtp.PlainAuth("", a.Config.SMTPUser, a.Config.SMTPPassword, a.Config.SMTPHost)
	}
	return smtp.SendMail(fmt.Sprintf("%s:%d", a.Config.SMTPHost, a.Config.SMTPPort), auth, fromAddr, []string{to}, msg)
}
