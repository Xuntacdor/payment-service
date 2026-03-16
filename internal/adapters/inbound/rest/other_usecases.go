package rest

import (
	"fmt"

	"github.com/Xuntacdor/payment-service/internal/domain/entity"
	"github.com/Xuntacdor/payment-service/internal/domain/port"
)

type cancelPaymentUseCase struct {
	paymentRepo    port.PaymentRepositoryPort
	eventPublisher port.EventPublisherPort
}

func NewCancelPaymentUseCase(
	paymentRepo port.PaymentRepositoryPort,
	eventPublisher port.EventPublisherPort,
) port.CancelPaymentUseCase {
	return &cancelPaymentUseCase{
		paymentRepo:    paymentRepo,
		eventPublisher: eventPublisher,
	}
}

func (uc *cancelPaymentUseCase) Execute(paymentID string) error {
	payment, err := uc.paymentRepo.FindByID(paymentID)
	if err != nil {
		return fmt.Errorf("payment not found: %w", err)
	}
	if err := payment.Cancel(); err != nil {
		return fmt.Errorf("cancel state transition failed: %w", err)
	}
	if err := uc.paymentRepo.Update(payment); err != nil {
		return fmt.Errorf("failed to persist cancellation: %w", err)
	}
	_ = uc.eventPublisher.Publish("PaymentCancelled", map[string]interface{}{
		"paymentId": payment.PaymentID,
		"orderId":   payment.OrderID,
	})
	return nil
}

type getPaymentUseCase struct {
	paymentRepo port.PaymentRepositoryPort
}

func NewGetPaymentUseCase(paymentRepo port.PaymentRepositoryPort) port.GetPaymentUseCase {
	return &getPaymentUseCase{paymentRepo: paymentRepo}
}

func (uc *getPaymentUseCase) Execute(paymentID string) (*entity.Payment, error) {
	payment, err := uc.paymentRepo.FindByID(paymentID)
	if err != nil {
		return nil, fmt.Errorf("payment not found: %w", err)
	}
	return payment, nil
}
