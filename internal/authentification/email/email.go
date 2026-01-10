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
        <h2>Vérifiez votre adresse e-mail</h2>
        <p>Merci de vous être inscrit ! Veuillez cliquer sur le bouton ci-dessous pour vérifier votre adresse e-mail :</p>
        <p><a href="{{.URL}}" class="button">Vérifier mon e-mail</a></p>
        <p>Ou copiez ce lien dans votre navigateur :</p>
        <p>{{.URL}}</p>
        <p>Ce lien expirera dans 24 heures.</p>
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
	m.SetHeader("Subject", "Vérifiez votre adresse e-mail")
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
        <h2>Réinitialisation de mot de passe</h2>
        <p>Vous avez demandé la réinitialisation de votre mot de passe. Cliquez sur le bouton ci-dessous :</p>
        <p><a href="{{.URL}}" class="button">Réinitialiser mon mot de passe</a></p>
        <p>Ou copiez ce lien dans votre navigateur :</p>
        <p>{{.URL}}</p>
        <p>Ce lien expirera dans 1 heure.</p>
        <p>Si vous n'avez pas demandé cette réinitialisation, ignorez cet e-mail.</p>
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
	m.SetHeader("Subject", "Réinitialisation de votre mot de passe")
	m.SetBody("text/html", body.String())

	return s.dialer.DialAndSend(m)
}
