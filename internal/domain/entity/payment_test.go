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

func TestNewPayment_InitialStatus(t *testing.T) {
	p := newTestPayment(t)
	if p.Status != entity.StatusCreated {
		t.Errorf("expected CREATED, got %s", p.Status)
	}
	if p.PaymentID == "" {
		t.Error("expected non-empty PaymentID")
	}
}

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

func TestPayment_Cancel_FromCreated(t *testing.T) {
	p := newTestPayment(t)
	if err := p.Cancel(); err != nil {
		t.Fatalf("unexpected error: %v", err)
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

func TestPayment_Fail_AfterRefund_ShouldFail(t *testing.T) {
	p := newTestPayment(t)
	_ = p.Authorize()
	_ = p.Capture()
	_ = p.Refund()
	if err := p.Fail(); err == nil {
		t.Fatal("expected error: cannot fail a REFUNDED payment")
	}
}

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

func TestNewMoney_InvalidCurrency(t *testing.T) {
	if _, err := entity.NewMoney(100, "US"); err == nil {
		t.Fatal("expected error for 2-char currency code")
	}
}

func TestNewPayment_EmptyOrderID(t *testing.T) {
	money := entity.Money{Amount: 50, Currency: "USD"}
	if _, err := entity.NewPayment("", money, entity.MethodCard); err == nil {
		t.Fatal("expected error for empty orderID")
	}
}
