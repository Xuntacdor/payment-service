package repository

import (
	"fmt"
	"time"

	"github.com/Xuntacdor/payment-service/internal/domain/entity"
	"github.com/Xuntacdor/payment-service/internal/domain/port"
	"gorm.io/gorm"
)

type PaymentModel struct {
	PaymentID     string    `gorm:"primaryKey;column:payment_id"`
	OrderID       string    `gorm:"uniqueIndex;column:order_id"`
	Amount        float64   `gorm:"column:amount"`
	Currency      string    `gorm:"column:currency"`
	Status        string    `gorm:"column:status"`
	PaymentMethod string    `gorm:"column:payment_method"`
	CreatedAt     time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt     time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (PaymentModel) TableName() string { return "payments" }

type TransactionModel struct {
	TransactionID        string    `gorm:"primaryKey;column:transaction_id"`
	PaymentID            string    `gorm:"index;column:payment_id"`
	Gateway              string    `gorm:"column:gateway"`
	GatewayTransactionID string    `gorm:"column:gateway_transaction_id"`
	Status               string    `gorm:"column:status"`
	Amount               float64   `gorm:"column:amount"`
	Currency             string    `gorm:"column:currency"`
	CreatedAt            time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (TransactionModel) TableName() string { return "transactions" }

type postgresPaymentRepository struct {
	db *gorm.DB
}

func NewPostgresPaymentRepository(db *gorm.DB) port.PaymentRepositoryPort {
	return &postgresPaymentRepository{db: db}
}

func (r *postgresPaymentRepository) Save(payment *entity.Payment) error {
	model := toPaymentModel(payment)
	if err := r.db.Create(&model).Error; err != nil {
		return fmt.Errorf("failed to save payment: %w", err)
	}
	return nil
}

func (r *postgresPaymentRepository) FindByID(paymentID string) (*entity.Payment, error) {
	var model PaymentModel
	if err := r.db.Where("payment_id = ?", paymentID).First(&model).Error; err != nil {
		return nil, fmt.Errorf("payment not found [id=%s]: %w", paymentID, err)
	}
	var txModels []TransactionModel
	r.db.Where("payment_id = ?", paymentID).Find(&txModels)
	return toDomainPayment(model, txModels), nil
}

func (r *postgresPaymentRepository) FindByOrderID(orderID string) (*entity.Payment, error) {
	var model PaymentModel
	if err := r.db.Where("order_id = ?", orderID).First(&model).Error; err != nil {
		return nil, fmt.Errorf("payment not found [orderID=%s]: %w", orderID, err)
	}
	return toDomainPayment(model, nil), nil
}

func (r *postgresPaymentRepository) Update(payment *entity.Payment) error {
	model := toPaymentModel(payment)
	if err := r.db.Save(&model).Error; err != nil {
		return fmt.Errorf("failed to update payment: %w", err)
	}
	return nil
}

type postgresTransactionRepository struct {
	db *gorm.DB
}

func NewPostgresTransactionRepository(db *gorm.DB) port.TransactionRepositoryPort {
	return &postgresTransactionRepository{db: db}
}

func (r *postgresTransactionRepository) Save(tx entity.Transaction) error {
	model := TransactionModel{
		TransactionID:        tx.TransactionID,
		PaymentID:            tx.PaymentID,
		Gateway:              string(tx.Gateway),
		GatewayTransactionID: tx.GatewayTransactionID,
		Status:               string(tx.Status),
		Amount:               tx.Amount.Amount,
		Currency:             tx.Amount.Currency,
		CreatedAt:            tx.CreatedAt,
	}
	if err := r.db.Create(&model).Error; err != nil {
		return fmt.Errorf("failed to save transaction: %w", err)
	}
	return nil
}

func (r *postgresTransactionRepository) FindByPaymentID(paymentID string) ([]entity.Transaction, error) {
	var models []TransactionModel
	if err := r.db.Where("payment_id = ?", paymentID).Find(&models).Error; err != nil {
		return nil, fmt.Errorf("failed to find transactions: %w", err)
	}
	txs := make([]entity.Transaction, 0, len(models))
	for _, m := range models {
		txs = append(txs, entity.Transaction{
			TransactionID:        m.TransactionID,
			PaymentID:            m.PaymentID,
			Gateway:              entity.GatewayProvider(m.Gateway),
			GatewayTransactionID: m.GatewayTransactionID,
			Status:               entity.TransactionStatus(m.Status),
			Amount:               entity.Money{Amount: m.Amount, Currency: m.Currency},
			CreatedAt:            m.CreatedAt,
		})
	}
	return txs, nil
}

func toPaymentModel(p *entity.Payment) PaymentModel {
	return PaymentModel{
		PaymentID:     p.PaymentID,
		OrderID:       p.OrderID,
		Amount:        p.Amount.Amount,
		Currency:      p.Amount.Currency,
		Status:        string(p.Status),
		PaymentMethod: string(p.PaymentMethod),
		CreatedAt:     p.CreatedAt,
		UpdatedAt:     p.UpdatedAt,
	}
}

func toDomainPayment(m PaymentModel, txModels []TransactionModel) *entity.Payment {
	txs := make([]entity.Transaction, 0, len(txModels))
	for _, t := range txModels {
		txs = append(txs, entity.Transaction{
			TransactionID:        t.TransactionID,
			PaymentID:            t.PaymentID,
			Gateway:              entity.GatewayProvider(t.Gateway),
			GatewayTransactionID: t.GatewayTransactionID,
			Status:               entity.TransactionStatus(t.Status),
			Amount:               entity.Money{Amount: t.Amount, Currency: t.Currency},
			CreatedAt:            t.CreatedAt,
		})
	}
	return &entity.Payment{
		PaymentID:     m.PaymentID,
		OrderID:       m.OrderID,
		Amount:        entity.Money{Amount: m.Amount, Currency: m.Currency},
		Status:        entity.PaymentStatus(m.Status),
		PaymentMethod: entity.PaymentMethod(m.PaymentMethod),
		CreatedAt:     m.CreatedAt,
		UpdatedAt:     m.UpdatedAt,
		Transactions:  txs,
	}
}
