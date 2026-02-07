package notifications

import (
	"fmt"
	"log"
	"net/smtp"
	"os"
	"strconv"
	"strings"
)

type EmailSender interface {
	Send(to, subject, textBody, htmlBody string) error
}

type SMTPSender struct {
	host    string
	port    int
	user    string
	pass    string
	from    string
	enabled bool
}

func NewSMTPSenderFromEnv() *SMTPSender {
	host := os.Getenv("SMTP_HOST")
	portStr := os.Getenv("SMTP_PORT")
	user := os.Getenv("SMTP_USER")
	pass := os.Getenv("SMTP_PASSWORD")
	from := os.Getenv("FROM_EMAIL")

	port, _ := strconv.Atoi(portStr)
	enabled := host != "" && port > 0 && user != "" && pass != "" && from != ""

	return &SMTPSender{
		host:    host,
		port:    port,
		user:    user,
		pass:    pass,
		from:    from,
		enabled: enabled,
	}
}

func (s *SMTPSender) Send(to, subject, textBody, htmlBody string) error {
	if !s.enabled {
		log.Printf("SMTP not configured. Would send to=%s subject=%q", to, subject)
		return nil
	}

	boundary := "mixed-alt-boundary"
	var msg strings.Builder
	msg.WriteString(fmt.Sprintf("From: %s\r\n", s.from))
	msg.WriteString(fmt.Sprintf("To: %s\r\n", to))
	msg.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	msg.WriteString("MIME-Version: 1.0\r\n")
	msg.WriteString(fmt.Sprintf("Content-Type: multipart/alternative; boundary=%s\r\n\r\n", boundary))

	msg.WriteString(fmt.Sprintf("--%s\r\n", boundary))
	msg.WriteString("Content-Type: text/plain; charset=utf-8\r\n\r\n")
	msg.WriteString(textBody + "\r\n\r\n")

	msg.WriteString(fmt.Sprintf("--%s\r\n", boundary))
	msg.WriteString("Content-Type: text/html; charset=utf-8\r\n\r\n")
	msg.WriteString(htmlBody + "\r\n\r\n")
	msg.WriteString(fmt.Sprintf("--%s--\r\n", boundary))

	auth := smtp.PlainAuth("", s.user, s.pass, s.host)
	addr := fmt.Sprintf("%s:%d", s.host, s.port)
	return smtp.SendMail(addr, auth, s.from, []string{to}, []byte(msg.String()))
}
