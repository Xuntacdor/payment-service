package stripe

import (
	"fmt"

	"github.com/Xuntacdor/payment-service/internal/domain/entity"
	"github.com/Xuntacdor/payment-service/internal/domain/port"
	stripe "github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/charge"
	"github.com/stripe/stripe-go/v76/refund"
)

type StripeAdapter struct {
	apiKey string
}

func NewStripeAdapter(apiKey string) port.PaymentGatewayPort {
	stripe.Key = apiKey
	return &StripeAdapter{apiKey: apiKey}
}

func (a *StripeAdapter) Charge(input port.GatewayChargeInput) (*port.GatewayChargeOutput, error) {
	amountInCents := int64(input.Amount.Amount * 100)

	params := &stripe.ChargeParams{
		Amount:      stripe.Int64(amountInCents),
		Currency:    stripe.String(input.Amount.Currency),
		Description: stripe.String(input.Description),
	}
	params.SetIdempotencyKey(input.ReferenceID)

	switch input.PaymentMethod {
	case entity.MethodCard:
		params.Source = &stripe.PaymentSourceSourceParams{Token: stripe.String("tok_visa")}
	default:
		return nil, fmt.Errorf("unsupported payment method for Stripe: %s", input.PaymentMethod)
	}

	ch, err := charge.New(params)
	if err != nil {
		return nil, fmt.Errorf("stripe charge failed: %w", err)
	}

	return &port.GatewayChargeOutput{
		GatewayTransactionID: ch.ID,
		Status:               string(ch.Status),
		RawResponse: map[string]interface{}{
			"stripe_charge_id": ch.ID,
			"paid":             ch.Paid,
			"amount":           ch.Amount,
		},
	}, nil
}

func (a *StripeAdapter) Refund(input port.GatewayRefundInput) (*port.GatewayChargeOutput, error) {
	params := &stripe.RefundParams{
		Charge: stripe.String(input.GatewayTransactionID),
		Amount: stripe.Int64(int64(input.Amount.Amount * 100)),
		Reason: stripe.String(input.Reason),
	}

	r, err := refund.New(params)
	if err != nil {
		return nil, fmt.Errorf("stripe refund failed: %w", err)
	}

	return &port.GatewayChargeOutput{
		GatewayTransactionID: r.ID,
		Status:               string(r.Status),
	}, nil
}

func (a *StripeAdapter) GetTransaction(gatewayTransactionID string) (*port.GatewayChargeOutput, error) {
	ch, err := charge.Get(gatewayTransactionID, nil)
	if err != nil {
		return nil, fmt.Errorf("stripe get transaction failed: %w", err)
	}
	return &port.GatewayChargeOutput{
		GatewayTransactionID: ch.ID,
		Status:               string(ch.Status),
	}, nil
}
