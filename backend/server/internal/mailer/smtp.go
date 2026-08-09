package mailer

import (
	"bytes"
	"fmt"
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

// SendHTML generates and dispatches a dynamically styled HTML email
func (m *Mailer) SendHTML(toEmail, subject, htmlBody string) error {
	var body bytes.Buffer

	// Construct MIME headers for HTML email
	body.WriteString(fmt.Sprintf("To: %s\r\n", toEmail))
	body.WriteString(fmt.Sprintf("From: %s\r\n", m.config.From))
	body.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	body.WriteString("MIME-version: 1.0\r\n")
	body.WriteString("Content-Type: text/html; charset=\"UTF-8\"\r\n")
	body.WriteString("\r\n")

	// Wrap the injected HTML in an email-safe container with basic typography
	fullHTML := fmt.Sprintf(`
	<!DOCTYPE html>
	<html>
	<head>
		<meta charset="UTF-8">
	</head>
	<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif; line-height: 1.6; color: #333333; background-color: #f4f4f5; margin: 0; padding: 40px 20px;">
		<table width="100%%" border="0" cellspacing="0" cellpadding="0">
			<tr>
				<td align="center">
					<div style="max-width: 600px; text-align: left; background-color: #ffffff; padding: 40px; border-radius: 8px; box-shadow: 0 4px 6px rgba(0,0,0,0.05); border: 1px solid #e4e4e7;">
						%s
					</div>
					<div style="max-width: 600px; text-align: center; margin-top: 20px; font-size: 12px; color: #a1a1aa;">
						Automated provisioning message sent by NexusIAM.
					</div>
				</td>
			</tr>
		</table>
	</body>
	</html>
	`, htmlBody)

	body.WriteString(fullHTML)

	auth := smtp.PlainAuth("", m.config.Username, m.config.Password, m.config.Host)
	addr := fmt.Sprintf("%s:%s", m.config.Host, m.config.Port)

	err := smtp.SendMail(addr, auth, m.config.From, []string{toEmail}, body.Bytes())
	if err != nil {
		return fmt.Errorf("failed to send SMTP email: %w", err)
	}

	return nil
}