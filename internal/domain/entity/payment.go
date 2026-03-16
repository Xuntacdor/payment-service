package entity

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

type PaymentStatus string

const (
	StatusCreated         PaymentStatus = "CREATED"
	StatusAuthorized      PaymentStatus = "AUTHORIZED"
	StatusPartialCaptured PaymentStatus = "PARTIAL_CAPTURED"
	StatusCaptured        PaymentStatus = "CAPTURED"
	StatusFailed          PaymentStatus = "FAILED"
	StatusRefunded        PaymentStatus = "REFUNDED"
	StatusCancelled       PaymentStatus = "CANCELLED"
)

type PaymentMethod string

const (
	MethodCard         PaymentMethod = "CARD"
	MethodWallet       PaymentMethod = "WALLET"
	MethodBankTransfer PaymentMethod = "BANK_TRANSFER"
)

type Money struct {
	Amount   float64
	Currency string
}

func NewMoney(amount float64, currency string) (Money, error) {
	if amount <= 0 {
		return Money{}, errors.New("amount must be greater than zero")
	}
	if len(currency) != 3 {
		return Money{}, errors.New("currency must be a valid ISO 4217 code (e.g. USD, VND)")
	}
	return Money{Amount: amount, Currency: currency}, nil
}

type Payment struct {
	PaymentID     string
	OrderID       string
	Amount        Money
	Status        PaymentStatus
	PaymentMethod PaymentMethod
	CreatedAt     time.Time
	UpdatedAt     time.Time
	Transactions  []Transaction
}

func NewPayment(orderID string, amount Money, method PaymentMethod) (*Payment, error) {
	if orderID == "" {
		return nil, errors.New("orderID is required")
	}
	return &Payment{
		PaymentID:     uuid.New().String(),
		OrderID:       orderID,
		Amount:        amount,
		Status:        StatusCreated,
		PaymentMethod: method,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}, nil
}

func (p *Payment) Authorize() error {
	if p.Status != StatusCreated {
		return errors.New("only CREATED payments can be authorized")
	}
	p.Status = StatusAuthorized
	p.UpdatedAt = time.Now()
	return nil
}

func (p *Payment) Capture() error {
	if p.Status != StatusAuthorized {
		return errors.New("only AUTHORIZED payments can be captured")
	}
	p.Status = StatusCaptured
	p.UpdatedAt = time.Now()
	return nil
}

func (p *Payment) Fail() error {
	if p.Status == StatusCaptured || p.Status == StatusRefunded {
		return errors.New("cannot fail a captured or refunded payment")
	}
	p.Status = StatusFailed
	p.UpdatedAt = time.Now()
	return nil
}

func (p *Payment) Refund() error {
	if p.Status != StatusCaptured {
		return errors.New("only CAPTURED payments can be refunded")
	}
	p.Status = StatusRefunded
	p.UpdatedAt = time.Now()
	return nil
}

func (p *Payment) Cancel() error {
	if p.Status == StatusCaptured || p.Status == StatusRefunded {
		return errors.New("cannot cancel a captured or refunded payment")
	}
	p.Status = StatusCancelled
	p.UpdatedAt = time.Now()
	return nil
}

func (p *Payment) AddTransaction(tx Transaction) {
	p.Transactions = append(p.Transactions, tx)
	p.UpdatedAt = time.Now()
}
