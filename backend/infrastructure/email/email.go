package email

import (
	"fmt"
	"net/smtp"
	"strings"
)

type EmailService struct {
	host     string
	port     int
	username string
	password string
	from     string
}

func NewEmailService(host string, port int, username, password, from string) *EmailService {
	return &EmailService{
		host:     host,
		port:     port,
		username: username,
		password: password,
		from:     from,
	}
}

// SendResetPasswordEmail 发送密码重置邮件
func (s *EmailService) SendResetPasswordEmail(to, resetLink string) error {
	subject := "CloudBox - 密码重置"
	body := fmt.Sprintf(`
<html>
<body style="font-family: Arial, sans-serif; padding: 20px;">
  <div style="max-width: 480px; margin: 0 auto; background: #fff; border-radius: 12px; border: 1px solid #e5e7eb; overflow: hidden;">
    <div style="background: #2F6BFF; padding: 24px; text-align: center;">
      <h1 style="color: #fff; margin: 0; font-size: 20px;">CloudBox</h1>
    </div>
    <div style="padding: 32px 24px;">
      <h2 style="margin: 0 0 12px; font-size: 18px; color: #1f2937;">重置你的密码</h2>
      <p style="margin: 0 0 24px; color: #6b7280; font-size: 14px; line-height: 1.6;">
        我们收到了你的密码重置请求。请点击下方按钮重置密码，此链接有效期为 30 分钟。
      </p>
      <a href="%s" style="display: block; width: 100%%; background: #2F6BFF; color: #fff; text-align: center; padding: 12px 0; border-radius: 8px; text-decoration: none; font-weight: 600; font-size: 15px;">重置密码</a>
      <p style="margin: 24px 0 0; color: #9ca3af; font-size: 12px;">
        如果你没有请求重置密码，请忽略此邮件。<br>
        此链接将在 30 分钟后过期。
      </p>
    </div>
  </div>
</body>
</html>`, resetLink)

	return s.send(to, subject, body)
}

func (s *EmailService) send(to, subject, body string) error {
	addr := fmt.Sprintf("%s:%d", s.host, s.port)
	auth := smtp.PlainAuth("", s.username, s.password, s.host)

	msg := []byte(fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s",
		s.from, to, subject, body))

	// 先尝试 STARTTLS（端口 587）
	if s.port == 587 {
		return s.sendWithSTARTTLS(addr, auth, s.from, []string{to}, msg)
	}

	// 端口 465 使用 SSL
	if s.port == 465 {
		return s.sendWithSSL(addr, auth, s.from, []string{to}, msg)
	}

	// 默认尝试普通 SMTP
	return smtp.SendMail(addr, auth, s.from, []string{to}, msg)
}

func (s *EmailService) sendWithSTARTTLS(addr string, auth smtp.Auth, from string, to []string, msg []byte) error {
	return smtp.SendMail(addr, auth, from, to, msg)
}

func (s *EmailService) sendWithSSL(addr string, auth smtp.Auth, from string, to []string, msg []byte) error {
	return smtp.SendMail(addr, auth, from, to, msg)
}

func buildResetLink(base, token string) string {
	if base == "" {
		base = "http://localhost:8080"
	}
	return strings.TrimRight(base, "/") + "/reset-password?token=" + token
}
