package mail

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"html/template"
	"net"
	"net/mail"
	"net/smtp"
	"net/url"
	"strconv"
	"strings"
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
			<h2>Xác thực tài khoản AstraCode</h2>
		</div>
		<div class="content">
			<p>Chào bạn,</p>
			<p>Bạn vừa đăng ký tài khoản trên AstraCode. Vui lòng nhấn nút bên dưới để kích hoạt tài khoản:</p>
			<a href="{{.Link}}" class="btn">Xác thực tài khoản</a>
			<p>Hoặc copy đường link sau vào trình duyệt:</p>
			<p class="link">{{.Link}}</p>
			<p>Link này sẽ hết hạn sau <strong>7 ngày</strong>. Nếu bạn không đăng ký tài khoản, hãy bỏ qua email này.</p>
		</div>
		<div class="footer">
			<p>&copy; {{.Year}} AstraCode. All rights reserved.</p>
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
			<h2>Đặt lại mật khẩu AstraCode</h2>
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
			<p>&copy; {{.Year}} AstraCode. All rights reserved.</p>
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
	smtpCfg.Security = strings.ToLower(strings.TrimSpace(smtpCfg.Security))
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
	link, err := s.frontendTokenURL("/verify-email", token, true)
	if err != nil {
		return err
	}

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

	return s.sendMail(ctx, toEmail, "Xác thực tài khoản AstraCode", body.Bytes())
}

func (s *smtpProvider) SendForgotPasswordEmail(ctx context.Context, toEmail, token string) error {
	link, err := s.frontendTokenURL("/reset-password", token, false)
	if err != nil {
		return err
	}

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

	return s.sendMail(ctx, toEmail, "Đặt lại mật khẩu AstraCode", body.Bytes())
}

func (s *smtpProvider) sendMail(ctx context.Context, toEmail, subject string, htmlBody []byte) error {
	if strings.ContainsAny(s.smtpCfg.FromName, "\r\n") {
		return errors.New("smtp from_name contains a newline")
	}
	from, err := mail.ParseAddress(s.smtpCfg.From)
	if err != nil || from.Address != s.smtpCfg.From {
		return errors.New("smtp from must be a valid bare email address")
	}

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

	addr := net.JoinHostPort(s.smtpCfg.Host, strconv.Itoa(s.smtpCfg.Port))

	dialer := net.Dialer{Timeout: s.smtpCfg.Timeout}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server: %w", err)
	}
	if err := conn.SetDeadline(time.Now().Add(s.smtpCfg.Timeout)); err != nil {
		conn.Close()
		return fmt.Errorf("set SMTP connection deadline: %w", err)
	}

	if s.smtpCfg.Security == "tls" {
		conn = tls.Client(conn, &tls.Config{ServerName: s.smtpCfg.Host, MinVersion: tls.VersionTLS12})
	}

	client, err := smtp.NewClient(conn, s.smtpCfg.Host)
	if err != nil {
		conn.Close()
		return fmt.Errorf("failed to create SMTP client: %w", err)
	}
	defer client.Close()

	if s.smtpCfg.Security == "starttls" {
		if ok, _ := client.Extension("STARTTLS"); !ok {
			return errors.New("SMTP server does not support STARTTLS")
		}
		if err = client.StartTLS(&tls.Config{ServerName: s.smtpCfg.Host, MinVersion: tls.VersionTLS12}); err != nil {
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

// ValidateConfig rejects unsafe or incomplete email settings before Auth starts.
// Server release mode is the repository's production mode; debug mode retains
// the intentionally unauthenticated MailHog development configuration.
func ValidateConfig(smtpCfg config.SMTPConfig, appCfg config.AppConfig, serverCfg config.ServerConfig) error {
	security := strings.ToLower(strings.TrimSpace(smtpCfg.Security))
	if security != "none" && security != "starttls" && security != "tls" {
		return fmt.Errorf("smtp.security must be one of none, starttls, tls")
	}
	if strings.TrimSpace(smtpCfg.Host) == "" {
		return errors.New("smtp.host is required")
	}
	if smtpCfg.Port < 1 || smtpCfg.Port > 65535 {
		return errors.New("smtp.port must be between 1 and 65535")
	}
	if smtpCfg.Timeout <= 0 {
		return errors.New("smtp.timeout must be positive")
	}
	if strings.ContainsAny(smtpCfg.FromName, "\r\n") {
		return errors.New("smtp.from_name must not contain a newline")
	}
	if strings.TrimSpace(smtpCfg.FromName) == "" {
		return errors.New("smtp.from_name is required")
	}
	from, err := mail.ParseAddress(smtpCfg.From)
	if err != nil || from.Address != smtpCfg.From {
		return errors.New("smtp.from must be a valid bare email address")
	}
	if (smtpCfg.Username == "") != (smtpCfg.Password == "") {
		return errors.New("smtp.username and smtp.password must be configured together")
	}
	if security == "none" && smtpCfg.Username != "" {
		return errors.New("smtp authentication requires starttls or tls")
	}

	parsed, err := parseFrontendURL(appCfg.FrontendURL)
	if err != nil {
		return err
	}
	if serverCfg.Mode == "release" {
		if security == "none" {
			return errors.New("smtp.security must be starttls or tls in release mode")
		}
		if smtpCfg.Username == "" {
			return errors.New("smtp authentication is required in release mode")
		}
		if parsed.Scheme != "https" {
			return errors.New("app.frontend_url must use https in release mode")
		}
	}
	return nil
}

func (s *smtpProvider) frontendTokenURL(path string, token string, fragment bool) (string, error) {
	parsed, err := parseFrontendURL(s.appCfg.FrontendURL)
	if err != nil {
		return "", err
	}

	canonicalPath := "/" + strings.TrimLeft(path, "/")
	if fragment {
		return parsed.String() + canonicalPath + "#token=" + url.QueryEscape(token), nil
	}

	return parsed.String() + canonicalPath + "?token=" + url.QueryEscape(token), nil
}

func parseFrontendURL(raw string) (*url.URL, error) {
	base := strings.TrimRight(strings.TrimSpace(raw), "/")
	if base == "" {
		return nil, errors.New("frontend_url is required")
	}
	parsed, err := url.Parse(base)
	if err != nil {
		return nil, fmt.Errorf("invalid frontend_url: %w", err)
	}
	if (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || parsed.User != nil || parsed.Path != "" || parsed.RawPath != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, errors.New("frontend_url must be an absolute http or https origin without credentials, query, or fragment")
	}
	return parsed, nil
}
