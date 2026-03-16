package entity

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// TransactionStatus represents the status of a gateway transaction
type TransactionStatus string

const (
	TxStatusPending  TransactionStatus = "PENDING"
	TxStatusSuccess  TransactionStatus = "SUCCESS"
	TxStatusFailed   TransactionStatus = "FAILED"
	TxStatusRefunded TransactionStatus = "REFUNDED"
)

// GatewayProvider identifies the external payment gateway
type GatewayProvider string

const (
	GatewayStripe GatewayProvider = "STRIPE"
	GatewayPayPal GatewayProvider = "PAYPAL"
	GatewayVNPay  GatewayProvider = "VNPAY"
	GatewayMoMo   GatewayProvider = "MOMO"
)

// Transaction records a single interaction with a payment gateway.
// It is a child entity of the Payment aggregate.
type Transaction struct {
	TransactionID        string
	PaymentID            string
	Gateway              GatewayProvider
	GatewayTransactionID string
	Status               TransactionStatus
	Amount               Money
	CreatedAt            time.Time
}

// NewTransaction creates a new Transaction record linked to a Payment
func NewTransaction(paymentID string, gateway GatewayProvider, gatewayTxID string, amount Money) Transaction {
	return Transaction{
		TransactionID:        fmt.Sprintf("tx_%s", uuid.New().String()),
		PaymentID:            paymentID,
		Gateway:              gateway,
		GatewayTransactionID: gatewayTxID,
		Status:               TxStatusPending,
		Amount:               amount,
		CreatedAt:            time.Now(),
	}
}