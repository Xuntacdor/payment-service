package entity_test

import (
	"testing"

	"github.com/Xuntacdor/payment-service/internal/domain/entity"
)

func newTestPayment(t *testing.T) *entity.Payment {
	t.Helper()
	money := entity.Money{Amount: 100.00, Currency: "USD"}
	payment, err := entity.NewPayment("order-001", money, entity.MethodCard)
	if err != nil {
		t.Fatalf("failed to create payment: %v", err)
	}
	return payment
}

// ---- NewPayment ----

func TestNewPayment_InitialStatus(t *testing.T) {
	p := newTestPayment(t)
	if p.Status != entity.StatusCreated {
		t.Errorf("expected CREATED, got %s", p.Status)
	}
	if p.PaymentID == "" {
		t.Error("expected non-empty PaymentID")
	}
	if p.OrderID != "order-001" {
		t.Errorf("expected orderID order-001, got %s", p.OrderID)
	}
}

func TestNewPayment_EmptyOrderID(t *testing.T) {
	money := entity.Money{Amount: 50, Currency: "USD"}
	if _, err := entity.NewPayment("", money, entity.MethodCard); err == nil {
		t.Fatal("expected error for empty orderID")
	}
}

// ---- NewMoney ----

func TestNewMoney_Valid(t *testing.T) {
	m, err := entity.NewMoney(100.0, "USD")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Amount != 100.0 || m.Currency != "USD" {
		t.Errorf("unexpected money value: %+v", m)
	}
}

func TestNewMoney_NegativeAmount(t *testing.T) {
	if _, err := entity.NewMoney(-10, "USD"); err == nil {
		t.Fatal("expected error for negative amount")
	}
}

func TestNewMoney_ZeroAmount(t *testing.T) {
	if _, err := entity.NewMoney(0, "USD"); err == nil {
		t.Fatal("expected error for zero amount")
	}
}

func TestNewMoney_InvalidCurrency(t *testing.T) {
	if _, err := entity.NewMoney(100, "US"); err == nil {
		t.Fatal("expected error for 2-char currency code")
	}
}

func TestNewMoney_InvalidCurrencyTooLong(t *testing.T) {
	if _, err := entity.NewMoney(100, "USDD"); err == nil {
		t.Fatal("expected error for 4-char currency code")
	}
}

// ---- Authorize ----

func TestPayment_Authorize_FromCreated(t *testing.T) {
	p := newTestPayment(t)
	if err := p.Authorize(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Status != entity.StatusAuthorized {
		t.Errorf("expected AUTHORIZED, got %s", p.Status)
	}
}

func TestPayment_Authorize_InvalidTransition(t *testing.T) {
	p := newTestPayment(t)
	_ = p.Authorize()
	_ = p.Capture()
	if err := p.Authorize(); err == nil {
		t.Fatal("expected error on invalid state transition CAPTURED→AUTHORIZED")
	}
}

// ---- Capture ----

func TestPayment_Capture_FromAuthorized(t *testing.T) {
	p := newTestPayment(t)
	_ = p.Authorize()
	if err := p.Capture(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Status != entity.StatusCaptured {
		t.Errorf("expected CAPTURED, got %s", p.Status)
	}
}

func TestPayment_Capture_NotAuthorized_ShouldFail(t *testing.T) {
	p := newTestPayment(t)
	if err := p.Capture(); err == nil {
		t.Fatal("expected error: cannot capture a CREATED payment")
	}
}

// ---- Fail ----

func TestPayment_Fail_FromCreated(t *testing.T) {
	p := newTestPayment(t)
	if err := p.Fail(); err != nil {
		t.Fatalf("unexpected error on fail: %v", err)
	}
	if p.Status != entity.StatusFailed {
		t.Errorf("expected FAILED, got %s", p.Status)
	}
}

func TestPayment_Fail_FromAuthorized(t *testing.T) {
	p := newTestPayment(t)
	_ = p.Authorize()
	if err := p.Fail(); err != nil {
		t.Fatalf("unexpected error on fail from AUTHORIZED: %v", err)
	}
	if p.Status != entity.StatusFailed {
		t.Errorf("expected FAILED, got %s", p.Status)
	}
}

func TestPayment_Fail_AfterCapture_ShouldFail(t *testing.T) {
	p := newTestPayment(t)
	_ = p.Authorize()
	_ = p.Capture()
	if err := p.Fail(); err == nil {
		t.Fatal("expected error: cannot fail a CAPTURED payment")
	}
}

func TestPayment_Fail_AfterRefund_ShouldFail(t *testing.T) {
	p := newTestPayment(t)
	_ = p.Authorize()
	_ = p.Capture()
	_ = p.Refund()
	if err := p.Fail(); err == nil {
		t.Fatal("expected error: cannot fail a REFUNDED payment")
	}
}

// ---- Refund ----

func TestPayment_Refund_FromCaptured(t *testing.T) {
	p := newTestPayment(t)
	_ = p.Authorize()
	_ = p.Capture()
	if err := p.Refund(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Status != entity.StatusRefunded {
		t.Errorf("expected REFUNDED, got %s", p.Status)
	}
}

func TestPayment_Refund_NotCaptured_ShouldFail(t *testing.T) {
	p := newTestPayment(t)
	_ = p.Authorize()
	if err := p.Refund(); err == nil {
		t.Fatal("expected error: cannot refund an AUTHORIZED payment")
	}
}

// ---- Cancel ----

func TestPayment_Cancel_FromCreated(t *testing.T) {
	p := newTestPayment(t)
	if err := p.Cancel(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Status != entity.StatusCancelled {
		t.Errorf("expected CANCELLED, got %s", p.Status)
	}
}

func TestPayment_Cancel_FromAuthorized(t *testing.T) {
	p := newTestPayment(t)
	_ = p.Authorize()
	if err := p.Cancel(); err != nil {
		t.Fatalf("unexpected error: cancel from AUTHORIZED should work: %v", err)
	}
	if p.Status != entity.StatusCancelled {
		t.Errorf("expected CANCELLED, got %s", p.Status)
	}
}

func TestPayment_Cancel_AfterCapture_ShouldFail(t *testing.T) {
	p := newTestPayment(t)
	_ = p.Authorize()
	_ = p.Capture()
	if err := p.Cancel(); err == nil {
		t.Fatal("expected error: cannot cancel a CAPTURED payment")
	}
}

func TestPayment_Cancel_AfterRefund_ShouldFail(t *testing.T) {
	p := newTestPayment(t)
	_ = p.Authorize()
	_ = p.Capture()
	_ = p.Refund()
	if err := p.Cancel(); err == nil {
		t.Fatal("expected error: cannot cancel a REFUNDED payment")
	}
}

// ---- AddTransaction ----

func TestPayment_AddTransaction(t *testing.T) {
	p := newTestPayment(t)
	tx := entity.NewTransaction(
		p.PaymentID,
		entity.GatewayStripe,
		"ch_test_123",
		p.Amount,
	)
	p.AddTransaction(tx)

	if len(p.Transactions) != 1 {
		t.Fatalf("expected 1 transaction, got %d", len(p.Transactions))
	}
	if p.Transactions[0].GatewayTransactionID != "ch_test_123" {
		t.Errorf("unexpected gateway tx id: %s", p.Transactions[0].GatewayTransactionID)
	}
}

func TestPayment_AddTransaction_Multiple(t *testing.T) {
	p := newTestPayment(t)
	tx1 := entity.NewTransaction(p.PaymentID, entity.GatewayStripe, "ch_001", p.Amount)
	tx2 := entity.NewTransaction(p.PaymentID, entity.GatewayStripe, "ch_002", p.Amount)
	p.AddTransaction(tx1)
	p.AddTransaction(tx2)

	if len(p.Transactions) != 2 {
		t.Fatalf("expected 2 transactions, got %d", len(p.Transactions))
	}
}
