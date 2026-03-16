package email

import (
	"fmt"
	"net/smtp"

	"github.com/Xuntacdor/payment-service/internal/domain/entity"
	"github.com/Xuntacdor/payment-service/internal/domain/port"
)

type SMTPEmailAdapter struct {
	host     string
	port     string
	username string
	password string
	from     string
}

func NewSMTPEmailAdapter(host, smtpPort, username, password, from string) port.EmailPort {
	return &SMTPEmailAdapter{
		host:     host,
		port:     smtpPort,
		username: username,
		password: password,
		from:     from,
	}
}

func (a *SMTPEmailAdapter) SendPaymentConfirmation(to string, p *entity.Payment) error {
	return a.send(to,
		"Payment Confirmed ✓",
		fmt.Sprintf("Your payment of %.2f %s for order %s has been confirmed.\nPayment ID: %s",
			p.Amount.Amount, p.Amount.Currency, p.OrderID, p.PaymentID),
	)
}

func (a *SMTPEmailAdapter) SendPaymentFailedNotification(to string, p *entity.Payment) error {
	return a.send(to,
		"Payment Failed ✗",
		fmt.Sprintf("Your payment of %.2f %s for order %s has failed. Please try again.\nPayment ID: %s",
			p.Amount.Amount, p.Amount.Currency, p.OrderID, p.PaymentID),
	)
}

func (a *SMTPEmailAdapter) SendRefundConfirmation(to string, p *entity.Payment) error {
	return a.send(to,
		"Refund Processed ↩",
		fmt.Sprintf("Your refund of %.2f %s for order %s has been processed.\nPayment ID: %s",
			p.Amount.Amount, p.Amount.Currency, p.OrderID, p.PaymentID),
	)
}

func (a *SMTPEmailAdapter) send(to, subject, body string) error {
	auth := smtp.PlainAuth("", a.username, a.password, a.host)
	msg := []byte(fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		a.from, to, subject, body,
	))
	if err := smtp.SendMail(fmt.Sprintf("%s:%s", a.host, a.port), auth, a.from, []string{to}, msg); err != nil {
		return fmt.Errorf("smtp send failed: %w", err)
	}
	return nil
}
