package mocks

import (
	"fmt"

	"github.com/Xuntacdor/payment-service/internal/domain/entity"
	"github.com/Xuntacdor/payment-service/internal/domain/port"
)

// ---- Mock PaymentGatewayPort ----

// MockPaymentGateway is a controllable test double for PaymentGatewayPort.
// Set ShouldFail = true to simulate gateway errors.
type MockPaymentGateway struct {
	ShouldFail bool
}

func (m *MockPaymentGateway) Charge(input port.GatewayChargeInput) (*port.GatewayChargeOutput, error) {
	if m.ShouldFail {
		return nil, fmt.Errorf("mock gateway: charge failed")
	}
	return &port.GatewayChargeOutput{
		GatewayTransactionID: "mock_tx_" + input.ReferenceID,
		Status:               "success",
	}, nil
}

func (m *MockPaymentGateway) Refund(input port.GatewayRefundInput) (*port.GatewayChargeOutput, error) {
	if m.ShouldFail {
		return nil, fmt.Errorf("mock gateway: refund failed")
	}
	return &port.GatewayChargeOutput{
		GatewayTransactionID: "mock_refund_" + input.GatewayTransactionID,
		Status:               "refunded",
	}, nil
}

func (m *MockPaymentGateway) GetTransaction(id string) (*port.GatewayChargeOutput, error) {
	return &port.GatewayChargeOutput{GatewayTransactionID: id, Status: "success"}, nil
}

// ---- In-Memory PaymentRepositoryPort ----

// InMemoryPaymentRepository stores payments in memory for isolated unit tests.
type InMemoryPaymentRepository struct {
	store map[string]*entity.Payment
}

func NewInMemoryPaymentRepository() port.PaymentRepositoryPort {
	return &InMemoryPaymentRepository{store: make(map[string]*entity.Payment)}
}

func (r *InMemoryPaymentRepository) Save(p *entity.Payment) error {
	r.store[p.PaymentID] = p
	return nil
}

func (r *InMemoryPaymentRepository) FindByID(id string) (*entity.Payment, error) {
	p, ok := r.store[id]
	if !ok {
		return nil, fmt.Errorf("payment not found: %s", id)
	}
	return p, nil
}

func (r *InMemoryPaymentRepository) FindByOrderID(orderID string) (*entity.Payment, error) {
	for _, p := range r.store {
		if p.OrderID == orderID {
			return p, nil
		}
	}
	return nil, fmt.Errorf("no payment found for orderID: %s", orderID)
}

func (r *InMemoryPaymentRepository) Update(p *entity.Payment) error {
	if _, ok := r.store[p.PaymentID]; !ok {
		return fmt.Errorf("payment not found for update: %s", p.PaymentID)
	}
	r.store[p.PaymentID] = p
	return nil
}

// ---- In-Memory TransactionRepositoryPort ----

// InMemoryTransactionRepository stores transactions in memory for isolated unit tests.
type InMemoryTransactionRepository struct {
	store []entity.Transaction
}

func NewInMemoryTransactionRepository() port.TransactionRepositoryPort {
	return &InMemoryTransactionRepository{}
}

func (r *InMemoryTransactionRepository) Save(tx entity.Transaction) error {
	r.store = append(r.store, tx)
	return nil
}

func (r *InMemoryTransactionRepository) FindByPaymentID(paymentID string) ([]entity.Transaction, error) {
	var result []entity.Transaction
	for _, tx := range r.store {
		if tx.PaymentID == paymentID {
			result = append(result, tx)
		}
	}
	return result, nil
}

// ---- Mock EventPublisherPort ----

// MockEventPublisher records all published events for assertion in tests.
type MockEventPublisher struct {
	Published []map[string]interface{}
}

func (m *MockEventPublisher) Publish(eventName string, payload interface{}) error {
	m.Published = append(m.Published, map[string]interface{}{
		"event":   eventName,
		"payload": payload,
	})
	return nil
}

// ---- Mock EmailPort ----

// MockEmailAdapter is a no-op email adapter for tests.
type MockEmailAdapter struct {
	SentEmails []string
}

func (m *MockEmailAdapter) SendPaymentConfirmation(to string, _ *entity.Payment) error {
	m.SentEmails = append(m.SentEmails, "confirmation:"+to)
	return nil
}
func (m *MockEmailAdapter) SendPaymentFailedNotification(to string, _ *entity.Payment) error {
	m.SentEmails = append(m.SentEmails, "failed:"+to)
	return nil
}
func (m *MockEmailAdapter) SendRefundConfirmation(to string, _ *entity.Payment) error {
	m.SentEmails = append(m.SentEmails, "refund:"+to)
	return nil
}