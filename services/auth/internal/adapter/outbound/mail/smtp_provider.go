package mail

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"html/template"
	"net"
	"net/smtp"
	"time"

	"go-judge-system/pkg/config"
	"go-judge-system/services/auth/internal/application/port/outbound"

	"go.uber.org/zap"
)

const otpTemplateHTML = `
<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<style>
		body { font-family: Arial, sans-serif; background-color: #f4f4f4; margin: 0; padding: 0; }
		.container { max-width: 600px; margin: 40px auto; background-color: #ffffff; padding: 30px; border-radius: 8px; box-shadow: 0 4px 8px rgba(0,0,0,0.1); }
		.header { text-align: center; border-bottom: 2px solid #e0e0e0; padding-bottom: 20px; margin-bottom: 20px; }
		.header h2 { color: #333333; }
		.content { font-size: 16px; color: #555555; line-height: 1.6; }
		.otp-code { display: block; width: max-content; margin: 20px auto; font-size: 32px; font-weight: bold; color: #2e6c80; background-color: #f0f7fa; padding: 15px 30px; border-radius: 6px; letter-spacing: 5px; }
		.footer { text-align: center; margin-top: 30px; font-size: 12px; color: #999999; }
	</style>
</head>
<body>
	<div class="container">
		<div class="header">
			<h2>Xác thực tài khoản Go-Judge</h2>
		</div>
		<div class="content">
			<p>Chào bạn,</p>
			<p>Bạn vừa yêu cầu mã xác thực (OTP) để đăng ký tài khoản trên hệ thống Go-Judge. Vui lòng sử dụng mã dưới đây để hoàn tất quá trình đăng ký:</p>
			<span class="otp-code">{{.OTP}}</span>
			<p>Mã này sẽ hết hạn sau <strong>5 phút</strong>. Tuyệt đối không chia sẻ mã này cho bất kỳ ai.</p>
		</div>
		<div class="footer">
			<p>&copy; {{.Year}} Go-Judge System. All rights reserved.</p>
		</div>
	</div>
</body>
</html>
`

type smtpProvider struct {
	cfg    config.SMTPConfig
	logger *zap.Logger
	tmpl   *template.Template
}

func NewSMTPProvider(cfg config.SMTPConfig, logger *zap.Logger) outbound.MailProvider {
	tmpl, err := template.New("otp_email").Parse(otpTemplateHTML)
	if err != nil {
		panic("Failed to parse email template: " + err.Error())
	}

	return &smtpProvider{
		cfg:    cfg,
		logger: logger,
		tmpl:   tmpl,
	}
}

func (s *smtpProvider) SendOTP(ctx context.Context, toEmail string, otp string) error {
	data := struct {
		OTP  string
		Year int
	}{
		OTP:  otp,
		Year: time.Now().Year(),
	}

	var body bytes.Buffer
	if err := s.tmpl.Execute(&body, data); err != nil {
		return err
	}

	headers := make(map[string]string)
	headers["From"] = fmt.Sprintf("%s <%s>", s.cfg.FromName, s.cfg.From)
	headers["To"] = toEmail
	headers["Subject"] = "Mã xác thực đăng ký tài khoản Go-Judge"
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "text/html; charset=\"utf-8\""

	var msg bytes.Buffer
	for k, v := range headers {
		msg.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
	msg.WriteString("\r\n")
	msg.Write(body.Bytes())

	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)

	// Dial the SMTP server manually
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server: %w", err)
	}

	client, err := smtp.NewClient(conn, s.cfg.Host)
	if err != nil {
		conn.Close()
		return fmt.Errorf("failed to create SMTP client: %w", err)
	}
	defer client.Close()

	// Upgrade to TLS only if the server supports STARTTLS
	if ok, _ := client.Extension("STARTTLS"); ok {
		tlsConfig := &tls.Config{ServerName: s.cfg.Host}
		if err = client.StartTLS(tlsConfig); err != nil {
			return fmt.Errorf("STARTTLS failed: %w", err)
		}
	}

	// Authenticate only if credentials are provided
	if s.cfg.Username != "" && s.cfg.Password != "" {
		auth := smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.Host)
		if err = client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP auth failed: %w", err)
		}
	}

	// Set sender and recipient
	if err = client.Mail(s.cfg.From); err != nil {
		return fmt.Errorf("MAIL FROM failed: %w", err)
	}
	if err = client.Rcpt(toEmail); err != nil {
		return fmt.Errorf("RCPT TO failed: %w", err)
	}

	// Write the email body
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("DATA command failed: %w", err)
	}
	if _, err = w.Write(msg.Bytes()); err != nil {
		return fmt.Errorf("failed to write email body: %w", err)
	}
	if err = w.Close(); err != nil {
		return fmt.Errorf("failed to close email body: %w", err)
	}

	return client.Quit()
}
