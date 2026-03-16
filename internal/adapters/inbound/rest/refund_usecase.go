package rest

import (
	"fmt"

	"github.com/Xuntacdor/payment-service/internal/domain/port"
)

type refundUseCase struct {
	paymentRepo    port.PaymentRepositoryPort
	gateway        port.PaymentGatewayPort
	emailPort      port.EmailPort
	eventPublisher port.EventPublisherPort
}

func NewRefundUseCase(
	paymentRepo port.PaymentRepositoryPort,
	gateway port.PaymentGatewayPort,
	emailPort port.EmailPort,
	eventPublisher port.EventPublisherPort,
) port.RefundUseCase {
	return &refundUseCase{
		paymentRepo:    paymentRepo,
		gateway:        gateway,
		emailPort:      emailPort,
		eventPublisher: eventPublisher,
	}
}

func (uc *refundUseCase) Execute(input port.RefundInput) (*port.RefundOutput, error) {

	payment, err := uc.paymentRepo.FindByID(input.PaymentID)
	if err != nil {
		return nil, fmt.Errorf("payment not found: %w", err)
	}

	if len(payment.Transactions) == 0 {
		return nil, fmt.Errorf("no gateway transactions found for payment %s", input.PaymentID)
	}
	gatewayTxID := payment.Transactions[0].GatewayTransactionID

	_, err = uc.gateway.Refund(port.GatewayRefundInput{
		GatewayTransactionID: gatewayTxID,
		Amount:               payment.Amount,
		Reason:               input.Reason,
	})
	if err != nil {
		return nil, fmt.Errorf("gateway refund failed: %w", err)
	}

	if err := payment.Refund(); err != nil {
		return nil, fmt.Errorf("refund state transition failed: %w", err)
	}

	if err := uc.paymentRepo.Update(payment); err != nil {
		return nil, fmt.Errorf("failed to persist refund: %w", err)
	}

	_ = uc.eventPublisher.Publish("PaymentRefunded", map[string]interface{}{
		"paymentId": payment.PaymentID,
		"orderId":   payment.OrderID,
		"amount":    payment.Amount.Amount,
		"currency":  payment.Amount.Currency,
	})

	_ = uc.emailPort.SendRefundConfirmation("user@example.com", payment)

	return &port.RefundOutput{
		PaymentID:      payment.PaymentID,
		Status:         payment.Status,
		RefundedAmount: payment.Amount.Amount,
	}, nil
}
