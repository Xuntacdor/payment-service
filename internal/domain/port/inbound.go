package port

import "github.com/Xuntacdor/payment-service/internal/domain/entity"

type ProcessPaymentInput struct {
	OrderID       string
	Amount        float64
	Currency      string
	PaymentMethod entity.PaymentMethod
}

type ProcessPaymentOutput struct {
	PaymentID string
	Status    entity.PaymentStatus
	Fee       float64
	Total     float64
}

type ProcessPaymentUseCase interface {
	Execute(input ProcessPaymentInput) (*ProcessPaymentOutput, error)
}

type RefundInput struct {
	PaymentID string
	Reason    string
}

type RefundOutput struct {
	PaymentID      string
	Status         entity.PaymentStatus
	RefundedAmount float64
}

type RefundUseCase interface {
	Execute(input RefundInput) (*RefundOutput, error)
}

type CancelPaymentUseCase interface {
	Execute(paymentID string) error
}

type GetPaymentUseCase interface {
	Execute(paymentID string) (*entity.Payment, error)
}
