package service

import (
	"fmt"
	"net/smtp"
	"zpay/internal/pkg"

	"go.uber.org/zap"
)

type EmailClient interface {
	SendTransactionEmail(toEmail, fromEmail, toRecipient, transactionID string, amount string) error
	Close() error
}

type smtpClient struct {
	config *pkg.SMTPConfig
	logger *pkg.Logger
}

func NewEmailClient(cfg *pkg.SMTPConfig, logger *pkg.Logger) (EmailClient, error) {
	if !cfg.Enabled {
		return &noOpEmailClient{}, nil
	}

	if cfg.Host == "" || cfg.Port == 0 {
		return nil, fmt.Errorf("invalid SMTP configuration")
	}

	return &smtpClient{
		config: cfg,
		logger: logger,
	}, nil
}

func (ec *smtpClient) SendTransactionEmail(toEmail, fromEmail, toRecipient, transactionID string, amount string) error {
	if !ec.config.Enabled {
		return nil
	}

	subject := "Transaction Confirmation - ZPay"
	body := fmt.Sprintf(`
        <html>
            <body>
                <h2>Transaction Confirmation</h2>
                <p>Dear %s,</p>
                <p>Your transaction has been processed successfully.</p>
                <table border="1" cellpadding="10">
                    <tr>
                        <td><strong>Transaction ID</strong></td>
                        <td>%s</td>
                    </tr>
                    <tr>
                        <td><strong>From</strong></td>
                        <td>%s</td>
                    </tr>
                    <tr>
                        <td><strong>To</strong></td>
                        <td>%s</td>
                    </tr>
                    <tr>
                        <td><strong>Amount</strong></td>
                        <td>$%s</td>
                    </tr>
                    <tr>
                        <td><strong>Status</strong></td>
                        <td>Completed</td>
                    </tr>
                </table>
                <p>If you did not initiate this transaction, please contact support immediately.</p>
                <p>Best regards,<br>ZPay Team</p>
            </body>
        </html>
    `, fromEmail, transactionID, fromEmail, toRecipient, amount)

	return ec.sendMail(toEmail, subject, body)
}

func (ec *smtpClient) sendMail(to, subject, body string) error {
	addr := fmt.Sprintf("%s:%d", ec.config.Host, ec.config.Port)
	message := fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-version: 1.0;\r\nContent-Type: text/html; charset=\"UTF-8\";\r\n\r\n%s",
		ec.config.From, to, subject, body,
	)

	auth := smtp.PlainAuth("", ec.config.Username, ec.config.Password, ec.config.Host)
	if err := smtp.SendMail(addr, auth, ec.config.From, []string{to}, []byte(message)); err != nil {
		ec.logger.Error("failed to send email", zap.Error(fmt.Errorf("email: %s, error: %w", to, err)))
		return fmt.Errorf("failed to send email: %w", err)
	}

	ec.logger.Info("Email sent successfully")
	return nil
}

func (ec *smtpClient) Close() error {
	return nil
}

type noOpEmailClient struct{}

func (nec *noOpEmailClient) SendTransactionEmail(toEmail, fromEmail, toRecipient, transactionID string, amount string) error {
	return nil
}

func (nec *noOpEmailClient) Close() error {
	return nil
}
