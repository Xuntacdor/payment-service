package port

import "github.com/Xuntacdor/payment-service/internal/domain/entity"

// ---- Payment Gateway ----

// GatewayChargeInput represents a charge request to a payment gateway
type GatewayChargeInput struct {
	Amount        entity.Money
	PaymentMethod entity.PaymentMethod
	ReferenceID   string // idempotency key = orderID
	Description   string
}

// GatewayChargeOutput holds the gateway's response to a charge or refund
type GatewayChargeOutput struct {
	GatewayTransactionID string
	Status               string
	RawResponse          map[string]interface{}
}

// GatewayRefundInput represents a refund request sent to a payment gateway
type GatewayRefundInput struct {
	GatewayTransactionID string
	Amount               entity.Money
	Reason               string
}

// PaymentGatewayPort defines how the domain communicates with external payment gateways.
// Implemented by: StripeAdapter, VNPayAdapter, MoMoAdapter
type PaymentGatewayPort interface {
	Charge(input GatewayChargeInput) (*GatewayChargeOutput, error)
	Refund(input GatewayRefundInput) (*GatewayChargeOutput, error)
	GetTransaction(gatewayTransactionID string) (*GatewayChargeOutput, error)
}

// ---- Persistence ----

// PaymentRepositoryPort defines persistence operations for the Payment aggregate.
// Implemented by: PostgresPaymentRepository
type PaymentRepositoryPort interface {
	Save(payment *entity.Payment) error
	FindByID(paymentID string) (*entity.Payment, error)
	FindByOrderID(orderID string) (*entity.Payment, error)
	Update(payment *entity.Payment) error
}

// TransactionRepositoryPort defines persistence operations for Transaction records.
type TransactionRepositoryPort interface {
	Save(tx entity.Transaction) error
	FindByPaymentID(paymentID string) ([]entity.Transaction, error)
}

// ---- Notification ----

// EmailPort defines the notification contract for payment lifecycle events.
// Implemented by: SMTPEmailAdapter, SendGridAdapter
type EmailPort interface {
	SendPaymentConfirmation(to string, payment *entity.Payment) error
	SendPaymentFailedNotification(to string, payment *entity.Payment) error
	SendRefundConfirmation(to string, payment *entity.Payment) error
}

// ---- Messaging ----

// EventPublisherPort defines how domain events are published to a message broker.
// Implemented by: KafkaPublisher, RabbitMQPublisher
// Events: PaymentCaptured → Order Service, PaymentFailed → Order Service, PaymentRefunded → Order Service
type EventPublisherPort interface {
	Publish(eventName string, payload interface{}) error
}