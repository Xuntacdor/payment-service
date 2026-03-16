package fee_test

import (
	"testing"

	"github.com/Xuntacdor/payment-service/pkg/fee"
)

func TestCalculate_Card(t *testing.T) {
	result, err := fee.Calculate(100.0, "USD", fee.MethodCard)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.FeeAmount != 2.90 {
		t.Errorf("expected fee 2.90, got %.2f", result.FeeAmount)
	}
	if result.Total != 102.90 {
		t.Errorf("expected total 102.90, got %.2f", result.Total)
	}
}

func TestCalculate_Wallet(t *testing.T) {
	result, err := fee.Calculate(200.0, "VND", fee.MethodWallet)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.FeeAmount != 3.00 {
		t.Errorf("expected 3.00, got %.2f", result.FeeAmount)
	}
}

func TestCalculate_InvalidAmount(t *testing.T) {
	if _, err := fee.Calculate(-5, "USD", fee.MethodCard); err == nil {
		t.Fatal("expected error for negative amount")
	}
}

func TestCalculate_UnsupportedMethod(t *testing.T) {
	if _, err := fee.Calculate(100, "USD", fee.Method("CRYPTO")); err == nil {
		t.Fatal("expected error for unsupported method")
	}
}