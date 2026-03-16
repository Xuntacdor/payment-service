package entity

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type TransactionStatus string

const (
	TxStatusPending  TransactionStatus = "PENDING"
	TxStatusSuccess  TransactionStatus = "SUCCESS"
	TxStatusFailed   TransactionStatus = "FAILED"
	TxStatusRefunded TransactionStatus = "REFUNDED"
)

type GatewayProvider string

const (
	GatewayStripe GatewayProvider = "STRIPE"
	GatewayPayPal GatewayProvider = "PAYPAL"
	GatewayVNPay  GatewayProvider = "VNPAY"
	GatewayMoMo   GatewayProvider = "MOMO"
)

type Transaction struct {
	TransactionID        string
	PaymentID            string
	Gateway              GatewayProvider
	GatewayTransactionID string
	Status               TransactionStatus
	Amount               Money
	CreatedAt            time.Time
}

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
