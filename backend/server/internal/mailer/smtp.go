package mailer

import (
	"bytes"
	"fmt"
	"html/template"
	"net/smtp"
)

type SMTPConfig struct {
	Host     string
	Port     string
	Username string
	Password string
	From     string
}

type Mailer struct {
	config SMTPConfig
}

func NewMailer(config SMTPConfig) *Mailer {
	return &Mailer{config: config}
}

// InviteData contains the variables injected into the HTML email template
type InviteData struct {
	Username       string
	InviteURL      string
	ExpiresInHours int
}

const inviteTemplate = `
<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333;">
	<h2>Welcome to the Server Network, {{.Username}}!</h2>
	<p>An administrator has provisioned access for you.</p>
	<p>Please click the button below to complete your onboarding, set your password, and upload your SSH public key.</p>
	<p style="margin: 30px 0;">
		<a href="{{.InviteURL}}" style="background-color: #007bff; color: white; padding: 12px 24px; text-decoration: none; border-radius: 4px;">Complete Onboarding</a>
	</p>
	<p><em>This link expires in {{.ExpiresInHours}} hours.</em></p>
</body>
</html>
`

// SendInvite generates and dispatches the HTML email
func (m *Mailer) SendInvite(toEmail, username, inviteURL string, expiresInHours int) error {
	tmpl, err := template.New("invite").Parse(inviteTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse email template: %w", err)
	}

	data := InviteData{
		Username:       username,
		InviteURL:      inviteURL,
		ExpiresInHours: expiresInHours,
	}

	var body bytes.Buffer
	// Construct MIME headers for HTML email
	body.WriteString(fmt.Sprintf("To: %s\r\n", toEmail))
	body.WriteString(fmt.Sprintf("From: %s\r\n", m.config.From))
	body.WriteString("Subject: Action Required: Complete your Server Onboarding\r\n")
	body.WriteString("MIME-version: 1.0\r\n")
	body.WriteString("Content-Type: text/html; charset=\"UTF-8\"\r\n")
	body.WriteString("\r\n")
	
	if err := tmpl.Execute(&body, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	auth := smtp.PlainAuth("", m.config.Username, m.config.Password, m.config.Host)
	addr := fmt.Sprintf("%s:%s", m.config.Host, m.config.Port)

	err = smtp.SendMail(addr, auth, m.config.From, []string{toEmail}, body.Bytes())
	if err != nil {
		return fmt.Errorf("failed to send SMTP email: %w", err)
	}

	return nil
}