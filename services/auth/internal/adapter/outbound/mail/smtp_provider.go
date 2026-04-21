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

const verificationTemplateHTML = `
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
		.btn { display: block; width: max-content; margin: 20px auto; padding: 14px 32px; background-color: #2e6c80; color: #ffffff; text-decoration: none; font-size: 16px; font-weight: bold; border-radius: 6px; }
		.link { word-break: break-all; color: #2e6c80; font-size: 13px; }
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
			<p>Bạn vừa đăng ký tài khoản trên hệ thống Go-Judge. Vui lòng nhấn nút bên dưới để kích hoạt tài khoản:</p>
			<a href="{{.Link}}" class="btn">Xác thực tài khoản</a>
			<p>Hoặc copy đường link sau vào trình duyệt:</p>
			<p class="link">{{.Link}}</p>
			<p>Link này sẽ hết hạn sau <strong>24 giờ</strong>. Nếu bạn không đăng ký tài khoản, hãy bỏ qua email này.</p>
		</div>
		<div class="footer">
			<p>&copy; {{.Year}} Go-Judge System. All rights reserved.</p>
		</div>
	</div>
</body>
</html>
`

const passwordResetTemplateHTML = `
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
		.btn { display: block; width: max-content; margin: 20px auto; padding: 14px 32px; background-color: #c0392b; color: #ffffff; text-decoration: none; font-size: 16px; font-weight: bold; border-radius: 6px; }
		.link { word-break: break-all; color: #c0392b; font-size: 13px; }
		.footer { text-align: center; margin-top: 30px; font-size: 12px; color: #999999; }
	</style>
</head>
<body>
	<div class="container">
		<div class="header">
			<h2>Đặt lại mật khẩu Go-Judge</h2>
		</div>
		<div class="content">
			<p>Chào bạn,</p>
			<p>Bạn vừa yêu cầu đặt lại mật khẩu. Vui lòng nhấn nút bên dưới để tiếp tục:</p>
			<a href="{{.Link}}" class="btn">Đặt lại mật khẩu</a>
			<p>Hoặc copy đường link sau vào trình duyệt:</p>
			<p class="link">{{.Link}}</p>
			<p>Link này sẽ hết hạn sau <strong>15 phút</strong>. Nếu bạn không yêu cầu, hãy bỏ qua email này.</p>
		</div>
		<div class="footer">
			<p>&copy; {{.Year}} Go-Judge System. All rights reserved.</p>
		</div>
	</div>
</body>
</html>
`

type smtpProvider struct {
	smtpCfg          config.SMTPConfig
	appCfg           config.AppConfig
	logger           *zap.Logger
	verificationTmpl *template.Template
	resetTmpl        *template.Template
}

func NewSMTPProvider(smtpCfg config.SMTPConfig, appCfg config.AppConfig, logger *zap.Logger) outbound.MailProvider {
	verifyTmpl, err := template.New("verify_email").Parse(verificationTemplateHTML)
	if err != nil {
		panic("failed to parse verification email template: " + err.Error())
	}

	resetTmpl, err := template.New("reset_password").Parse(passwordResetTemplateHTML)
	if err != nil {
		panic("failed to parse password reset email template: " + err.Error())
	}

	return &smtpProvider{
		smtpCfg:          smtpCfg,
		appCfg:           appCfg,
		logger:           logger,
		verificationTmpl: verifyTmpl,
		resetTmpl:        resetTmpl,
	}
}

func (s *smtpProvider) SendVerificationEmail(ctx context.Context, toEmail, token string) error {
	link := fmt.Sprintf("%s/verify-email?token=%s", s.appCfg.FrontendURL, token)
	data := struct {
		Link string
		Year int
	}{
		Link: link,
		Year: time.Now().Year(),
	}

	var body bytes.Buffer
	if err := s.verificationTmpl.Execute(&body, data); err != nil {
		return err
	}

	return s.sendMail(toEmail, "Xác thực tài khoản Go-Judge", body.Bytes())
}

func (s *smtpProvider) SendForgotPasswordEmail(ctx context.Context, toEmail, token string) error {
	link := fmt.Sprintf("%s/reset-password?token=%s", s.appCfg.FrontendURL, token)
	data := struct {
		Link string
		Year int
	}{
		Link: link,
		Year: time.Now().Year(),
	}

	var body bytes.Buffer
	if err := s.resetTmpl.Execute(&body, data); err != nil {
		return err
	}

	return s.sendMail(toEmail, "Đặt lại mật khẩu Go-Judge", body.Bytes())
}

func (s *smtpProvider) sendMail(toEmail, subject string, htmlBody []byte) error {
	headers := make(map[string]string)
	headers["From"] = fmt.Sprintf("%s <%s>", s.smtpCfg.FromName, s.smtpCfg.From)
	headers["To"] = toEmail
	headers["Subject"] = subject
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = `text/html; charset="utf-8"`

	var msg bytes.Buffer
	for k, v := range headers {
		msg.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
	msg.WriteString("\r\n")
	msg.Write(htmlBody)

	addr := fmt.Sprintf("%s:%d", s.smtpCfg.Host, s.smtpCfg.Port)

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server: %w", err)
	}

	client, err := smtp.NewClient(conn, s.smtpCfg.Host)
	if err != nil {
		conn.Close()
		return fmt.Errorf("failed to create SMTP client: %w", err)
	}
	defer client.Close()

	if ok, _ := client.Extension("STARTTLS"); ok {
		tlsConfig := &tls.Config{ServerName: s.smtpCfg.Host}
		if err = client.StartTLS(tlsConfig); err != nil {
			return fmt.Errorf("STARTTLS failed: %w", err)
		}
	}

	if s.smtpCfg.Username != "" && s.smtpCfg.Password != "" {
		auth := smtp.PlainAuth("", s.smtpCfg.Username, s.smtpCfg.Password, s.smtpCfg.Host)
		if err = client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP auth failed: %w", err)
		}
	}

	if err = client.Mail(s.smtpCfg.From); err != nil {
		return fmt.Errorf("MAIL FROM failed: %w", err)
	}
	if err = client.Rcpt(toEmail); err != nil {
		return fmt.Errorf("RCPT TO failed: %w", err)
	}

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
