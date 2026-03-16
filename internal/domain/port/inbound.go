package port

import "github.com/Xuntacdor/payment-service/internal/domain/entity"

// ---- Process Payment ----

// ProcessPaymentInput carries the data needed to initiate a payment
type ProcessPaymentInput struct {
	OrderID       string
	Amount        float64
	Currency      string
	PaymentMethod entity.PaymentMethod
}

// ProcessPaymentOutput carries the result of a processed payment
type ProcessPaymentOutput struct {
	PaymentID string
	Status    entity.PaymentStatus
	Fee       float64
	Total     float64
}

// ProcessPaymentUseCase is the inbound port for creating and capturing payments
type ProcessPaymentUseCase interface {
	Execute(input ProcessPaymentInput) (*ProcessPaymentOutput, error)
}

// ---- Refund ----

// RefundInput carries the data needed to refund a payment
type RefundInput struct {
	PaymentID string
	Reason    string
}

// RefundOutput carries the result of a refund operation
type RefundOutput struct {
	PaymentID      string
	Status         entity.PaymentStatus
	RefundedAmount float64
}

// RefundUseCase is the inbound port for refunding captured payments
type RefundUseCase interface {
	Execute(input RefundInput) (*RefundOutput, error)
}

// ---- Cancel ----

// CancelPaymentUseCase is the inbound port for cancelling a payment before capture
type CancelPaymentUseCase interface {
	Execute(paymentID string) error
}

// ---- Query ----

// GetPaymentUseCase is the inbound port for querying payment details
type GetPaymentUseCase interface {
	Execute(paymentID string) (*entity.Payment, error)
}