package service_test

import (
	"testing"

	"github.com/Xuntacdor/payment-service/internal/domain/entity"
	"github.com/Xuntacdor/payment-service/internal/domain/service"
)

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
}

func TestCalculateFee_UnsupportedMethod(t *testing.T) {
	money := entity.Money{Amount: 100.00, Currency: "USD"}
	_, err := service.CalculateFee(money, entity.PaymentMethod("CRYPTO"))
	if err == nil {
		t.Fatal("expected error for unsupported payment method")
	}
}

func TestValidatePayment_Valid(t *testing.T) {
	money := entity.Money{Amount: 50.00, Currency: "USD"}
	err := service.ValidatePayment("order-001", money, entity.MethodCard)
	if err != nil {
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

func TestValidatePayment_BelowStripeMinimum(t *testing.T) {
	money := entity.Money{Amount: 0.30, Currency: "USD"}
	if err := service.ValidatePayment("order-001", money, entity.MethodCard); err == nil {
		t.Fatal("expected error for amount below $0.50 minimum")
	}
}

func TestValidatePayment_EmptyMethod(t *testing.T) {
	money := entity.Money{Amount: 50.00, Currency: "USD"}
	if err := service.ValidatePayment("order-001", money, ""); err == nil {
		t.Fatal("expected error for empty payment method")
	}
}
