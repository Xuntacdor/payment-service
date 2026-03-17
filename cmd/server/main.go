package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/Xuntacdor/payment-service/config"
	restadapter "github.com/Xuntacdor/payment-service/internal/adapters/inbound/rest"
	emailadapter "github.com/Xuntacdor/payment-service/internal/adapters/outbound/email"
	messaging "github.com/Xuntacdor/payment-service/internal/adapters/outbound/messaging"
	repo "github.com/Xuntacdor/payment-service/internal/adapters/outbound/repository"
	stripeadapter "github.com/Xuntacdor/payment-service/internal/adapters/outbound/stripe"
)

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("failed to load config", zap.Error(err))
	}

	db, err := gorm.Open(postgres.Open(cfg.Database.DSN), &gorm.Config{})
	if err != nil {
		logger.Fatal("failed to connect to database", zap.Error(err))
	}
	if err := db.AutoMigrate(&repo.PaymentModel{}, &repo.TransactionModel{}); err != nil {
		logger.Fatal("failed to run migrations", zap.Error(err))
	}
	logger.Info("database migrations complete")

	paymentRepo := repo.NewPostgresPaymentRepository(db)
	transactionRepo := repo.NewPostgresTransactionRepository(db)
	gateway := stripeadapter.NewStripeAdapter(cfg.Stripe.SecretKey)
	/*
			gateway := vnpay.NewVNPayAdapter(vnpay.Config{
		    TmnCode:    cfg.VNPay.TmnCode,
		    HashSecret: cfg.VNPay.HashSecret,
		    ReturnURL:  cfg.VNPay.ReturnURL,
		})
	*/
	emailAdapter := emailadapter.NewSMTPEmailAdapter(
		cfg.Email.Host, cfg.Email.Port,
		cfg.Email.Username, cfg.Email.Password, cfg.Email.From,
	)
	eventPublisher := messaging.NewMockKafkaPublisher(logger)

	processPaymentUC := restadapter.NewProcessPaymentUseCase(paymentRepo, transactionRepo, gateway, eventPublisher)
	refundUC := restadapter.NewRefundUseCase(paymentRepo, gateway, emailAdapter, eventPublisher)
	cancelUC := restadapter.NewCancelPaymentUseCase(paymentRepo, eventPublisher)
	getPaymentUC := restadapter.NewGetPaymentUseCase(paymentRepo)
	handler := restadapter.NewPaymentHandler(processPaymentUC, refundUC, cancelUC, getPaymentUC)

	router := gin.New()
	router.Use(gin.Recovery(), restadapter.PrometheusMiddleware())
	v1 := router.Group("/api/v1")
	handler.RegisterRoutes(router, v1)

	srv := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	go func() {
		logger.Info("payment service started", zap.String("port", cfg.Server.Port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("server error", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("shutdown signal received — draining connections...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("graceful shutdown failed", zap.Error(err))
	}
	logger.Info("payment service stopped cleanly")
}
