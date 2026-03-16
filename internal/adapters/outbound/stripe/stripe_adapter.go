package stripeadapter

import "github.com/Xuntacdor/payment-service/internal/domain/port"

type StripeAdapter struct {
	secretKey string
}

func NewStripeAdapter(secretKey string) port.PaymentGatewayPort {
	return &StripeAdapter{secretKey: secretKey}
}

func (s *StripeAdapter) Charge(input port.GatewayChargeInput) (*port.GatewayChargeOutput, error) {
	return &port.GatewayChargeOutput{
		GatewayTransactionID: "stripe_mock_txn_id",
	}, nil
}

func (s *StripeAdapter) Refund(input port.GatewayRefundInput) (*port.GatewayChargeOutput, error) {
	return nil, nil
}

func (s *StripeAdapter) GetTransaction(transactionID string) (*port.GatewayChargeOutput, error) {
	return nil, nil
}
