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
	if result.BaseAmount != 100.0 {
		t.Errorf("expected base 100.0, got %.2f", result.BaseAmount)
	}
	if result.Currency != "USD" {
		t.Errorf("expected USD, got %s", result.Currency)
	}
	if result.RateUsed != 0.029 {
		t.Errorf("expected rate 0.029, got %f", result.RateUsed)
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
	if result.RateUsed != 0.015 {
		t.Errorf("expected rate 0.015, got %f", result.RateUsed)
	}
}

func TestCalculate_BankTransfer(t *testing.T) {
	result, err := fee.Calculate(1000.0, "USD", fee.MethodBankTransfer)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.FeeAmount != 5.00 {
		t.Errorf("expected fee 5.00, got %.2f", result.FeeAmount)
	}
	if result.RateUsed != 0.005 {
		t.Errorf("expected rate 0.005, got %f", result.RateUsed)
	}
}

func TestCalculate_InvalidAmount(t *testing.T) {
	if _, err := fee.Calculate(-5, "USD", fee.MethodCard); err == nil {
		t.Fatal("expected error for negative amount")
	}
}

func TestCalculate_ZeroAmount(t *testing.T) {
	if _, err := fee.Calculate(0, "USD", fee.MethodCard); err == nil {
		t.Fatal("expected error for zero amount")
	}
}

func TestCalculate_EmptyCurrency(t *testing.T) {
	if _, err := fee.Calculate(100, "", fee.MethodCard); err == nil {
		t.Fatal("expected error for empty currency")
	}
}

func TestCalculate_UnsupportedMethod(t *testing.T) {
	if _, err := fee.Calculate(100, "USD", fee.Method("CRYPTO")); err == nil {
		t.Fatal("expected error for unsupported method")
	}
}

func TestCalculate_TotalIsBaseAndFee(t *testing.T) {
	result, err := fee.Calculate(50.0, "USD", fee.MethodCard)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := result.BaseAmount + result.FeeAmount
	// allow tiny float rounding
	diff := result.Total - expected
	if diff > 0.001 || diff < -0.001 {
		t.Errorf("Total %.2f != BaseAmount %.2f + FeeAmount %.2f", result.Total, result.BaseAmount, result.FeeAmount)
	}
}
