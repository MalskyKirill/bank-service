package service

import (
	"context"
	"fmt"
	"html"
)

type EmailSender interface {
	SendEmail(ctx context.Context, to string, subject string, htmlBody string) error
}

type NotificationService struct {
	emailSender EmailSender
}

func NewNotificationService(emailSender EmailSender) *NotificationService {
	return &NotificationService{
		emailSender: emailSender,
	}
}

func (s *NotificationService) SendCardPaymentEmail(
	ctx context.Context,
	to string,
	amount float64,
	description string,
) error {
	if s == nil || s.emailSender == nil {
		return nil
	}

	subject := "Платеж по карте успешно выполнен"

	body := fmt.Sprintf(`
		<h2>Платеж успешно выполнен</h2>
		<p>Сумма: <strong>%.2f RUB</strong></p>
		<p>Описание: %s</p>
		<br>
		<small>Это автоматическое уведомление банковского сервиса.</small>
	`, amount, html.EscapeString(description))

	return s.emailSender.SendEmail(ctx, to, subject, body)
}

func (s *NotificationService) SendCreditIssuedEmail(
	ctx context.Context,
	to string,
	amount float64,
	termMonths int,
	interestRate float64,
	monthlyPayment float64,
) error {
	if s == nil || s.emailSender == nil {
		return nil
	}

	subject := "Кредит успешно оформлен"

	body := fmt.Sprintf(`
		<h2>Кредит успешно оформлен</h2>
		<p>Сумма кредита: <strong>%.2f RUB</strong></p>
		<p>Срок: <strong>%d мес.</strong></p>
		<p>Процентная ставка: <strong>%.2f%%</strong></p>
		<p>Ежемесячный платеж: <strong>%.2f RUB</strong></p>
		<br>
		<small>Это автоматическое уведомление банковского сервиса.</small>
	`, amount, termMonths, interestRate, monthlyPayment)

	return s.emailSender.SendEmail(ctx, to, subject, body)
}
