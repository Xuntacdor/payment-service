package usecase

import (
	"fmt"

	"github.com/Xuntacdor/payment-service/internal/domain/port"
)

// refundUseCase implements port.RefundUseCase
type refundUseCase struct {
	paymentRepo    port.PaymentRepositoryPort
	gateway        port.PaymentGatewayPort
	emailPort      port.EmailPort
	eventPublisher port.EventPublisherPort
}

// NewRefundUseCase constructs the refund use case with all required dependencies injected
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
	// Step 1: Load the Payment aggregate
	payment, err := uc.paymentRepo.FindByID(input.PaymentID)
	if err != nil {
		return nil, fmt.Errorf("payment not found: %w", err)
	}

	// Step 2: Retrieve the gateway transaction ID needed for the refund API call
	if len(payment.Transactions) == 0 {
		return nil, fmt.Errorf("no gateway transactions found for payment %s", input.PaymentID)
	}
	gatewayTxID := payment.Transactions[0].GatewayTransactionID

	// Step 3: Call the gateway refund API via outbound port
	_, err = uc.gateway.Refund(port.GatewayRefundInput{
		GatewayTransactionID: gatewayTxID,
		Amount:               payment.Amount,
		Reason:               input.Reason,
	})
	if err != nil {
		return nil, fmt.Errorf("gateway refund failed: %w", err)
	}

	// Step 4: Transition aggregate state to REFUNDED (domain enforces this is only valid from CAPTURED)
	if err := payment.Refund(); err != nil {
		return nil, fmt.Errorf("refund state transition failed: %w", err)
	}

	// Step 5: Persist updated aggregate state
	if err := uc.paymentRepo.Update(payment); err != nil {
		return nil, fmt.Errorf("failed to persist refund: %w", err)
	}

	// Step 6: Publish PaymentRefunded event → Order Service updates status to REFUNDED
	_ = uc.eventPublisher.Publish("PaymentRefunded", map[string]interface{}{
		"paymentId": payment.PaymentID,
		"orderId":   payment.OrderID,
		"amount":    payment.Amount.Amount,
		"currency":  payment.Amount.Currency,
	})

	// Step 7: Notify the user via email
	_ = uc.emailPort.SendRefundConfirmation("user@example.com", payment)

	return &port.RefundOutput{
		PaymentID:      payment.PaymentID,
		Status:         payment.Status,
		RefundedAmount: payment.Amount.Amount,
	}, nil
}