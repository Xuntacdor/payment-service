package service

import (
	"errors"
	"math"

	"github.com/Xuntacdor/payment-service/internal/domain/entity"
)

var FeeRate = map[entity.PaymentMethod]float64{
	entity.MethodCard:         0.029,
	entity.MethodWallet:       0.015,
	entity.MethodBankTransfer: 0.005,
}

type FeeResult struct {
	BaseAmount float64
	FeeAmount  float64
	Total      float64
	Currency   string
}

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

	if money.Currency == "USD" && money.Amount < 0.50 {
		return errors.New("minimum payment amount is $0.50 USD")
	}
	return nil
}

func round(val float64) float64 {
	return math.Round(val*100) / 100
}
