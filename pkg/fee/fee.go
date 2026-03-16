package fee

import (
	"errors"
	"math"
)

type Method string

const (
	MethodCard         Method = "CARD"
	MethodWallet       Method = "WALLET"
	MethodBankTransfer Method = "BANK_TRANSFER"
)

var rates = map[Method]float64{
	MethodCard:         0.029,
	MethodWallet:       0.015,
	MethodBankTransfer: 0.005,
}

type Result struct {
	BaseAmount float64 `json:"base_amount"`
	FeeAmount  float64 `json:"fee_amount"`
	Total      float64 `json:"total"`
	Currency   string  `json:"currency"`
	RateUsed   float64 `json:"rate_used"`
}

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
