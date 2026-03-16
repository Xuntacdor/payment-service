package port

import "github.com/Xuntacdor/payment-service/internal/domain/entity"

type GatewayChargeInput struct {
	Amount        entity.Money
	PaymentMethod entity.PaymentMethod
	ReferenceID   string
	Description   string
}

type GatewayChargeOutput struct {
	GatewayTransactionID string
	Status               string
	RawResponse          map[string]interface{}
}

type GatewayRefundInput struct {
	GatewayTransactionID string
	Amount               entity.Money
	Reason               string
}

type PaymentGatewayPort interface {
	Charge(input GatewayChargeInput) (*GatewayChargeOutput, error)
	Refund(input GatewayRefundInput) (*GatewayChargeOutput, error)
	GetTransaction(gatewayTransactionID string) (*GatewayChargeOutput, error)
}

type PaymentRepositoryPort interface {
	Save(payment *entity.Payment) error
	FindByID(paymentID string) (*entity.Payment, error)
	FindByOrderID(orderID string) (*entity.Payment, error)
	Update(payment *entity.Payment) error
}

type TransactionRepositoryPort interface {
	Save(tx entity.Transaction) error
	FindByPaymentID(paymentID string) ([]entity.Transaction, error)
}

type EmailPort interface {
	SendPaymentConfirmation(to string, payment *entity.Payment) error
	SendPaymentFailedNotification(to string, payment *entity.Payment) error
	SendRefundConfirmation(to string, payment *entity.Payment) error
}

type EventPublisherPort interface {
	Publish(eventName string, payload interface{}) error
}
