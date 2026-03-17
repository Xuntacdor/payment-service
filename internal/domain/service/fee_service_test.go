package service_test

import (
	"testing"

	"github.com/Xuntacdor/payment-service/internal/domain/entity"
	"github.com/Xuntacdor/payment-service/internal/domain/service"
)

// ---- CalculateFee ----

func TestCalculateFee_Card_Success(t *testing.T) {
	money := entity.Money{Amount: 100.00, Currency: "USD"}
	result, err := service.CalculateFee(money, entity.MethodCard)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result.FeeAmount != 2.90 {
		t.Errorf("expected fee 2.90, got %.2f", result.FeeAmount)
	}
	if result.Total != 102.90 {
		t.Errorf("expected total 102.90, got %.2f", result.Total)
	}
	if result.BaseAmount != 100.00 {
		t.Errorf("expected base 100.00, got %.2f", result.BaseAmount)
	}
	if result.Currency != "USD" {
		t.Errorf("expected USD, got %s", result.Currency)
	}
}

func TestCalculateFee_Wallet_Success(t *testing.T) {
	money := entity.Money{Amount: 200.00, Currency: "USD"}
	result, err := service.CalculateFee(money, entity.MethodWallet)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result.FeeAmount != 3.00 {
		t.Errorf("expected fee 3.00, got %.2f", result.FeeAmount)
	}
	if result.Total != 203.00 {
		t.Errorf("expected total 203.00, got %.2f", result.Total)
	}
}

func TestCalculateFee_BankTransfer_Success(t *testing.T) {
	money := entity.Money{Amount: 1000.00, Currency: "USD"}
	result, err := service.CalculateFee(money, entity.MethodBankTransfer)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result.FeeAmount != 5.00 {
		t.Errorf("expected fee 5.00, got %.2f", result.FeeAmount)
	}
	if result.Total != 1005.00 {
		t.Errorf("expected total 1005.00, got %.2f", result.Total)
	}
}

func TestCalculateFee_UnsupportedMethod(t *testing.T) {
	money := entity.Money{Amount: 100.00, Currency: "USD"}
	_, err := service.CalculateFee(money, entity.PaymentMethod("CRYPTO"))
	if err == nil {
		t.Fatal("expected error for unsupported payment method")
	}
}

func TestCalculateFee_VND_Card(t *testing.T) {
	money := entity.Money{Amount: 500000, Currency: "VND"}
	result, err := service.CalculateFee(money, entity.MethodCard)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Currency != "VND" {
		t.Errorf("expected VND, got %s", result.Currency)
	}
	if result.FeeAmount <= 0 {
		t.Error("expected positive fee amount")
	}
}

// ---- ValidatePayment ----

func TestValidatePayment_Valid(t *testing.T) {
	money := entity.Money{Amount: 50.00, Currency: "USD"}
	if err := service.ValidatePayment("order-001", money, entity.MethodCard); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestValidatePayment_EmptyOrderID(t *testing.T) {
	money := entity.Money{Amount: 50.00, Currency: "USD"}
	if err := service.ValidatePayment("", money, entity.MethodCard); err == nil {
		t.Fatal("expected error for empty orderID")
	}
}

func TestValidatePayment_ZeroAmount(t *testing.T) {
	money := entity.Money{Amount: 0, Currency: "USD"}
	if err := service.ValidatePayment("order-001", money, entity.MethodCard); err == nil {
		t.Fatal("expected error for zero amount")
	}
}

func TestValidatePayment_NegativeAmount(t *testing.T) {
	money := entity.Money{Amount: -10, Currency: "USD"}
	if err := service.ValidatePayment("order-001", money, entity.MethodCard); err == nil {
		t.Fatal("expected error for negative amount")
	}
}

func TestValidatePayment_EmptyCurrency(t *testing.T) {
	money := entity.Money{Amount: 50.00, Currency: ""}
	if err := service.ValidatePayment("order-001", money, entity.MethodCard); err == nil {
		t.Fatal("expected error for empty currency")
	}
}

func TestValidatePayment_EmptyMethod(t *testing.T) {
	money := entity.Money{Amount: 50.00, Currency: "USD"}
	if err := service.ValidatePayment("order-001", money, ""); err == nil {
		t.Fatal("expected error for empty payment method")
	}
}

func TestValidatePayment_BelowStripeMinimum(t *testing.T) {
	money := entity.Money{Amount: 0.30, Currency: "USD"}
	if err := service.ValidatePayment("order-001", money, entity.MethodCard); err == nil {
		t.Fatal("expected error for amount below $0.50 minimum")
	}
}

func TestValidatePayment_ExactStripeMinimum(t *testing.T) {
	// $0.50 exactly should pass
	money := entity.Money{Amount: 0.50, Currency: "USD"}
	if err := service.ValidatePayment("order-001", money, entity.MethodCard); err != nil {
		t.Fatalf("expected no error for exactly $0.50: %v", err)
	}
}

func TestValidatePayment_VND_NoMinimum(t *testing.T) {
	// VND has no minimum check — small amounts should pass
	money := entity.Money{Amount: 0.10, Currency: "VND"}
	if err := service.ValidatePayment("order-001", money, entity.MethodCard); err != nil {
		t.Fatalf("expected no error for VND small amount: %v", err)
	}
}
