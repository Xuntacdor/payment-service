package service

import (
	"errors"
	"math"

	"github.com/Xuntacdor/payment-service/internal/domain/entity"
)

// FeeRate defines the percentage fee charged per payment method
var FeeRate = map[entity.PaymentMethod]float64{
	entity.MethodCard:         0.029, // 2.9%
	entity.MethodWallet:       0.015, // 1.5%
	entity.MethodBankTransfer: 0.005, // 0.5%
}

// FeeResult holds the full fee breakdown for a payment
type FeeResult struct {
	BaseAmount float64
	FeeAmount  float64
	Total      float64
	Currency   string
}

// CalculateFee computes the transaction fee for a given payment amount and method.
// This is pure domain logic — no external dependencies.
func CalculateFee(money entity.Money, method entity.PaymentMethod) (FeeResult, error) {
	rate, ok := FeeRate[method]
	if !ok {
		return FeeResult{}, errors.New("unsupported payment method for fee calculation")
	}
	fee := round(money.Amount * rate)
	return FeeResult{
		BaseAmount: money.Amount,
		FeeAmount:  fee,
		Total:      round(money.Amount + fee),
		Currency:   money.Currency,
	}, nil
}

// ValidatePayment enforces domain business rules before a payment is initiated.
// Returns a descriptive error if any rule is violated.
func ValidatePayment(orderID string, money entity.Money, method entity.PaymentMethod) error {
	if orderID == "" {
		return errors.New("orderID cannot be empty")
	}
	if money.Amount <= 0 {
		return errors.New("amount must be greater than zero")
	}
	if money.Currency == "" {
		return errors.New("currency is required")
	}
	if method == "" {
		return errors.New("payment method is required")
	}
	// Stripe minimum charge amount
	if money.Currency == "USD" && money.Amount < 0.50 {
		return errors.New("minimum payment amount is $0.50 USD")
	}
	return nil
}

// round rounds a float64 to 2 decimal places
func round(val float64) float64 {
	return math.Round(val*100) / 100
}