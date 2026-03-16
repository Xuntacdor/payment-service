package usecase

import (
	"fmt"

	"github.com/Xuntacdor/payment-service/internal/domain/entity"
	"github.com/Xuntacdor/payment-service/internal/domain/port"
	"github.com/Xuntacdor/payment-service/internal/domain/service"
)

// processPaymentUseCase implements port.ProcessPaymentUseCase
type processPaymentUseCase struct {
	paymentRepo     port.PaymentRepositoryPort
	transactionRepo port.TransactionRepositoryPort
	gateway         port.PaymentGatewayPort
	eventPublisher  port.EventPublisherPort
}

// NewProcessPaymentUseCase constructs the use case with all required dependencies injected
func NewProcessPaymentUseCase(
	paymentRepo port.PaymentRepositoryPort,
	transactionRepo port.TransactionRepositoryPort,
	gateway port.PaymentGatewayPort,
	eventPublisher port.EventPublisherPort,
) port.ProcessPaymentUseCase {
	return &processPaymentUseCase{
		paymentRepo:     paymentRepo,
		transactionRepo: transactionRepo,
		gateway:         gateway,
		eventPublisher:  eventPublisher,
	}
}

func (uc *processPaymentUseCase) Execute(input port.ProcessPaymentInput) (*port.ProcessPaymentOutput, error) {
	// Step 1: Construct and validate Money value object
	money, err := entity.NewMoney(input.Amount, input.Currency)
	if err != nil {
		return nil, fmt.Errorf("invalid money: %w", err)
	}

	// Step 2: Run domain validation rules
	if err := service.ValidatePayment(input.OrderID, money, input.PaymentMethod); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Step 3: Idempotency check — prevent duplicate payments for the same order
	if existing, _ := uc.paymentRepo.FindByOrderID(input.OrderID); existing != nil {
		return nil, fmt.Errorf("payment already exists for orderID: %s", input.OrderID)
	}

	// Step 4: Calculate fee (pure domain logic, no side effects)
	feeResult, err := service.CalculateFee(money, input.PaymentMethod)
	if err != nil {
		return nil, fmt.Errorf("fee calculation failed: %w", err)
	}

	// Step 5: Create Payment aggregate (domain object)
	payment, err := entity.NewPayment(input.OrderID, money, input.PaymentMethod)
	if err != nil {
		return nil, fmt.Errorf("failed to create payment aggregate: %w", err)
	}

	// Step 6: Persist payment in CREATED state before calling gateway
	if err := uc.paymentRepo.Save(payment); err != nil {
		return nil, fmt.Errorf("failed to persist payment: %w", err)
	}

	// Step 7: Call payment gateway via outbound port (infra concern, not domain)
	chargeOutput, err := uc.gateway.Charge(port.GatewayChargeInput{
		Amount:        money,
		PaymentMethod: input.PaymentMethod,
		ReferenceID:   input.OrderID, // idempotency key
		Description:   fmt.Sprintf("Payment for order %s", input.OrderID),
	})
	if err != nil {
		// Gateway failed: mark payment as FAILED and publish event for Order Service
		_ = payment.Fail()
		_ = uc.paymentRepo.Update(payment)
		_ = uc.eventPublisher.Publish("PaymentFailed", map[string]interface{}{
			"paymentId": payment.PaymentID,
			"orderId":   payment.OrderID,
		})
		return nil, fmt.Errorf("gateway charge failed: %w", err)
	}

	// Step 8: Record gateway transaction
	tx := entity.NewTransaction(payment.PaymentID, entity.GatewayStripe, chargeOutput.GatewayTransactionID, money)
	_ = uc.transactionRepo.Save(tx)
	payment.AddTransaction(tx)

	// Step 9: Transition aggregate state: CREATED → AUTHORIZED → CAPTURED
	_ = payment.Authorize()
	_ = payment.Capture()
	if err := uc.paymentRepo.Update(payment); err != nil {
		return nil, fmt.Errorf("failed to update payment status: %w", err)
	}

	// Step 10: Publish PaymentCaptured event → Order Service updates status to PAID
	_ = uc.eventPublisher.Publish("PaymentCaptured", map[string]interface{}{
		"paymentId": payment.PaymentID,
		"orderId":   payment.OrderID,
		"amount":    payment.Amount.Amount,
		"currency":  payment.Amount.Currency,
	})

	return &port.ProcessPaymentOutput{
		PaymentID: payment.PaymentID,
		Status:    payment.Status,
		Fee:       feeResult.FeeAmount,
		Total:     feeResult.Total,
	}, nil
}