package rest

import (
	"fmt"

	"github.com/Xuntacdor/payment-service/internal/domain/entity"
	"github.com/Xuntacdor/payment-service/internal/domain/port"
	"github.com/Xuntacdor/payment-service/internal/domain/service"
)

type processPaymentUseCase struct {
	paymentRepo     port.PaymentRepositoryPort
	transactionRepo port.TransactionRepositoryPort
	gateway         port.PaymentGatewayPort
	eventPublisher  port.EventPublisherPort
}

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
	money, err := entity.NewMoney(input.Amount, input.Currency)
	if err != nil {
		return nil, fmt.Errorf("invalid money: %w", err)
	}

	if err := service.ValidatePayment(input.OrderID, money, input.PaymentMethod); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	if existing, _ := uc.paymentRepo.FindByOrderID(input.OrderID); existing != nil {
		return nil, fmt.Errorf("payment already exists for orderID: %s", input.OrderID)
	}

	feeResult, err := service.CalculateFee(money, input.PaymentMethod)
	if err != nil {
		return nil, fmt.Errorf("fee calculation failed: %w", err)
	}

	payment, err := entity.NewPayment(input.OrderID, money, input.PaymentMethod)
	if err != nil {
		return nil, fmt.Errorf("failed to create payment aggregate: %w", err)
	}
	if err := uc.paymentRepo.Save(payment); err != nil {
		return nil, fmt.Errorf("failed to persist payment: %w", err)
	}

	chargeOutput, err := uc.gateway.Charge(port.GatewayChargeInput{
		Amount:        money,
		PaymentMethod: input.PaymentMethod,
		ReferenceID:   input.OrderID,
		Description:   fmt.Sprintf("Payment for order %s", input.OrderID),
	})
	if err != nil {
		_ = payment.Fail()
		_ = uc.paymentRepo.Update(payment)
		_ = uc.eventPublisher.Publish("PaymentFailed", map[string]interface{}{
			"paymentId": payment.PaymentID,
			"orderId":   payment.OrderID,
		})
		return nil, fmt.Errorf("gateway charge failed: %w", err)
	}

	tx := entity.NewTransaction(payment.PaymentID, entity.GatewayStripe, chargeOutput.GatewayTransactionID, money)
	_ = uc.transactionRepo.Save(tx)
	payment.AddTransaction(tx)
	_ = payment.Authorize()
	_ = payment.Capture()
	if err := uc.paymentRepo.Update(payment); err != nil {
		return nil, fmt.Errorf("failed to update payment status: %w", err)
	}

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
