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

	restadapter "github.com/Xuntacdor/payment-service/internal/adapters/inbound/rest"
	usecase     "github.com/Xuntacdor/payment-service/internal/adapters/inbound/rest"
	emailadapter "github.com/Xuntacdor/payment-service/internal/adapters/outbound/email"
	messaging   "github.com/Xuntacdor/payment-service/internal/adapters/outbound/messaging"
	repo        "github.com/Xuntacdor/payment-service/internal/adapters/outbound/repository"
	stripeadapter "github.com/Xuntacdor/payment-service/internal/adapters/outbound/stripe"
	"github.com/Xuntacdor/payment-service/internal/config"
)

func main() {
	// ── Logger ────────────────────────────────────────────────────────────
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	// ── Config ────────────────────────────────────────────────────────────
	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("failed to load config", zap.Error(err))
	}

	// ── Database ──────────────────────────────────────────────────────────
	db, err := gorm.Open(postgres.Open(cfg.Database.DSN), &gorm.Config{})
	if err != nil {
		logger.Fatal("failed to connect to database", zap.Error(err))
	}
	if err := db.AutoMigrate(&repo.PaymentModel{}, &repo.TransactionModel{}); err != nil {
		logger.Fatal("failed to run migrations", zap.Error(err))
	}
	logger.Info("database migrations complete")

	// ── Outbound Adapters ─────────────────────────────────────────────────
	paymentRepo     := repo.NewPostgresPaymentRepository(db)
	transactionRepo := repo.NewPostgresTransactionRepository(db)
	gateway         := stripeadapter.NewStripeAdapter(cfg.Stripe.SecretKey)
	emailAdapter    := emailadapter.NewSMTPEmailAdapter(
		cfg.Email.Host, cfg.Email.Port,
		cfg.Email.Username, cfg.Email.Password, cfg.Email.From,
	)
	eventPublisher := messaging.NewMockKafkaPublisher(logger)

	// ── Use Cases ─────────────────────────────────────────────────────────
	processPaymentUC := usecase.NewProcessPaymentUseCase(paymentRepo, transactionRepo, gateway, eventPublisher)
	refundUC         := usecase.NewRefundUseCase(paymentRepo, gateway, emailAdapter, eventPublisher)
	cancelUC         := usecase.NewCancelPaymentUseCase(paymentRepo, eventPublisher)
	getPaymentUC     := usecase.NewGetPaymentUseCase(paymentRepo)

	// ── Inbound Adapter ───────────────────────────────────────────────────
	handler := restadapter.NewPaymentHandler(processPaymentUC, refundUC, cancelUC, getPaymentUC)

	// ── Router ────────────────────────────────────────────────────────────
	router := gin.New()
	router.Use(gin.Recovery(), restadapter.PrometheusMiddleware())
	v1 := router.Group("/api/v1")
	handler.RegisterRoutes(router, v1)

	// ── Graceful HTTP Server ──────────────────────────────────────────────
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