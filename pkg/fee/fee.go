// Package fee provides a reusable, standalone fee calculation library.
// It has zero dependencies on payment-service internals and can be imported
// by order-service or any other microservice that needs to preview fees.
package fee

import (
	"errors"
	"math"
)

// Method represents a supported payment method
type Method string

const (
	MethodCard         Method = "CARD"
	MethodWallet       Method = "WALLET"
	MethodBankTransfer Method = "BANK_TRANSFER"
)

// rates maps each payment method to its percentage fee
var rates = map[Method]float64{
	MethodCard:         0.029,
	MethodWallet:       0.015,
	MethodBankTransfer: 0.005,
}

// Result holds a full fee breakdown
type Result struct {
	BaseAmount float64 `json:"base_amount"`
	FeeAmount  float64 `json:"fee_amount"`
	Total      float64 `json:"total"`
	Currency   string  `json:"currency"`
	RateUsed   float64 `json:"rate_used"`
}

// Calculate computes the fee for a given amount, currency, and payment method.
// This function is pure and stateless — safe to call from any service.
func Calculate(amount float64, currency string, method Method) (Result, error) {
	if amount <= 0 {
		return Result{}, errors.New("amount must be positive")
	}
	if currency == "" {
		return Result{}, errors.New("currency is required")
	}
	rate, ok := rates[method]
	if !ok {
		return Result{}, errors.New("unsupported payment method")
	}
	fee := round(amount * rate)
	return Result{
		BaseAmount: amount,
		FeeAmount:  fee,
		Total:      round(amount + fee),
		Currency:   currency,
		RateUsed:   rate,
	}, nil
}

func round(v float64) float64 {
	return math.Round(v*100) / 100
}