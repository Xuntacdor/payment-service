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
	"github.com/Xuntacdor/payment-service/internal/adapters/inbound/rest/middleware"
	emailadapter "github.com/Xuntacdor/payment-service/internal/adapters/outbound/email"
	gatewaypkg "github.com/Xuntacdor/payment-service/internal/adapters/outbound/gateway"
	messaging "github.com/Xuntacdor/payment-service/internal/adapters/outbound/messaging"
	repo "github.com/Xuntacdor/payment-service/internal/adapters/outbound/repository"
	stripeadapter "github.com/Xuntacdor/payment-service/internal/adapters/outbound/stripe"
	vnpayadapter "github.com/Xuntacdor/payment-service/internal/adapters/outbound/vnpay"
	"github.com/Xuntacdor/payment-service/internal/domain/entity"
	"github.com/Xuntacdor/payment-service/internal/domain/port"
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
	gw := WireGateway(
		cfg.Stripe.SecretKey,
		vnpayadapter.Config{
			TmnCode:    cfg.VNPay.TmnCode,
			HashSecret: cfg.VNPay.HashSecret,
			ReturnURL:  cfg.VNPay.ReturnURL,
		},
		logger,
	)
	emailAdapter := emailadapter.NewSMTPEmailAdapter(
		cfg.Email.Host, cfg.Email.Port,
		cfg.Email.Username, cfg.Email.Password, cfg.Email.From,
	)

	var eventPublisher port.EventPublisherPort
	if len(cfg.Kafka.Brokers) > 0 {
		eventPublisher = messaging.NewRealKafkaPublisher(cfg.Kafka.Brokers, cfg.Kafka.Topic, logger)
	} else {
		eventPublisher = messaging.NewMockKafkaPublisher(logger)
	}

	processPaymentUC := restadapter.NewProcessPaymentUseCase(paymentRepo, transactionRepo, &gw, eventPublisher)
	refundUC := restadapter.NewRefundUseCase(paymentRepo, &gw, emailAdapter, eventPublisher)
	cancelUC := restadapter.NewCancelPaymentUseCase(paymentRepo, eventPublisher)
	getPaymentUC := restadapter.NewGetPaymentUseCase(paymentRepo)
	handler := restadapter.NewPaymentHandler(processPaymentUC, refundUC, cancelUC, getPaymentUC)

	router := gin.New()
	router.Use(gin.Recovery(), restadapter.PrometheusMiddleware())
	// Rate limit all routes: e.g. 10 requests per second, burst 20.
	router.Use(middleware.RateLimitMiddleware(10, 20))

	v1 := router.Group("/api/v1")
	// Require API key for v1 routes
	v1.Use(middleware.APIKeyAuthMiddleware(cfg.Server.APIKey))

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

// WireGateway builds the FallbackGateway with all providers registered.
// Flow khi user thanh toán bằng CARD + USD:
//
//	Priority 1: Stripe   → thành công → dừng
//	Priority 1: Stripe   → lỗi        → thử tiếp
//	Priority 2: VNPay    → thành công → dừng
//	Priority 2: VNPay    → lỗi        → thử tiếp
//	→ Tất cả lỗi → trả về error tổng hợp
//
// Flow khi user thanh toán bằng BANK_TRANSFER + VND:
//
//	Stripe bị bỏ qua (không support VND BankTransfer)
//	Priority 2: VNPay → thành công → dừng
func WireGateway(
	stripKey string,
	vnpayConfig vnpayadapter.Config,
	logger *zap.Logger,
) gatewaypkg.FallbackGateway {

	// ── Khởi tạo từng gateway adapter ──────────────────────────────────
	stripeGW := stripeadapter.NewStripeAdapter(stripKey)
	vnpayGW := vnpayadapter.NewVNPayAdapter(vnpayConfig)

	// ── Đăng ký vào registry với priority + routing rules ──────────────
	registry := gatewaypkg.NewRegistry().
		Register(gatewaypkg.GatewayEntry{
			Name:     gatewaypkg.GatewayStripe,
			Gateway:  stripeGW,
			Priority: 1, // thử Stripe trước
			Enabled:  true,
			// Stripe chỉ nhận CARD và tiền tệ quốc tế
			SupportedMethods:    []entity.PaymentMethod{entity.MethodCard},
			SupportedCurrencies: []string{"USD", "EUR", "SGD", "GBP"},
		}).
		Register(gatewaypkg.GatewayEntry{
			Name:     gatewaypkg.GatewayVNPay,
			Gateway:  vnpayGW,
			Priority: 2, // fallback khi Stripe fail, hoặc thanh toán VND
			Enabled:  true,
			// VNPay nhận BankTransfer + Wallet + VND
			SupportedMethods:    []entity.PaymentMethod{entity.MethodBankTransfer, entity.MethodWallet, entity.MethodCard},
			SupportedCurrencies: []string{"VND"},
		})

	return *gatewaypkg.NewFallbackGateway(registry, logger).(*gatewaypkg.FallbackGateway)
}
