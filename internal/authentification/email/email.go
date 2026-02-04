package email

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"strconv"

	"gopkg.in/gomail.v2"
)

type EmailService struct {
	dialer *gomail.Dialer
	from   string
}

func NewEmailService() *EmailService {
	port, _ := strconv.Atoi(os.Getenv("SMTP_PORT"))

	dialer := gomail.NewDialer(
		os.Getenv("SMTP_HOST"),
		port,
		os.Getenv("SMTP_USER"),
		os.Getenv("SMTP_PASSWORD"),
	)

	return &EmailService{
		dialer: dialer,
		from:   os.Getenv("FROM_EMAIL"),
	}
}

func (s *EmailService) SendVerificationEmail(to, token string) error {
	verifyURL := fmt.Sprintf("%s/verify-email?token=%s", os.Getenv("FRONTEND_URL"), token)

	tmpl := `
<!DOCTYPE html>
<html>
<head>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .button { 
            display: inline-block; 
            padding: 12px 24px; 
            background-color: #4CAF50; 
            color: white; 
            text-decoration: none; 
            border-radius: 4px; 
        }
    </style>
</head>
<body>
    <div class="container">
        <h2>Verify your email address</h2>
        <p>Thanks for signing up! Please click the button below to verify your email address:</p>
        <p><a href="{{.URL}}" class="button">Verify my email</a></p>
        <p>Or copy this link into your browser:</p>
        <p>{{.URL}}</p>
        <p>This link will expire in 24 hours.</p>
    </div>
</body>
</html>
    `

	t, err := template.New("email").Parse(tmpl)
	if err != nil {
		return err
	}

	var body bytes.Buffer
	err = t.Execute(&body, map[string]string{"URL": verifyURL})
	if err != nil {
		return err
	}

	m := gomail.NewMessage()
	m.SetHeader("From", s.from)
	m.SetHeader("To", to)
	m.SetHeader("Subject", "Verify your email address")
	m.SetBody("text/html", body.String())

	return s.dialer.DialAndSend(m)
}

func (s *EmailService) SendPasswordResetEmail(to, token string) error {
	resetURL := fmt.Sprintf("%s/reset-password?token=%s", os.Getenv("FRONTEND_URL"), token)

	tmpl := `
<!DOCTYPE html>
<html>
<head>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .button { 
            display: inline-block; 
            padding: 12px 24px; 
            background-color: #2196F3; 
            color: white; 
            text-decoration: none; 
            border-radius: 4px; 
        }
    </style>
</head>
<body>
    <div class="container">
        <h2>Password reset</h2>
        <p>You requested a password reset. Click the button below:</p>
        <p><a href="{{.URL}}" class="button">Reset my password</a></p>
        <p>Or copy this link into your browser:</p>
        <p>{{.URL}}</p>
        <p>This link will expire in 1 hour.</p>
        <p>If you did not request this reset, ignore this email.</p>
    </div>
</body>
</html>
    `

	t, err := template.New("email").Parse(tmpl)
	if err != nil {
		return err
	}

	var body bytes.Buffer
	err = t.Execute(&body, map[string]string{"URL": resetURL})
	if err != nil {
		return err
	}

	m := gomail.NewMessage()
	m.SetHeader("From", s.from)
	m.SetHeader("To", to)
	m.SetHeader("Subject", "Password reset request")
	m.SetBody("text/html", body.String())

	return s.dialer.DialAndSend(m)
}
