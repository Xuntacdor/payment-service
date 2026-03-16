package rest

import (
	"net/http"
	"time"

	"github.com/Xuntacdor/payment-service/internal/domain/entity"
	"github.com/Xuntacdor/payment-service/internal/domain/port"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	httpRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "payment_http_requests_total",
		Help: "Total number of HTTP requests by method, path, and status code.",
	}, []string{"method", "path", "status"})

	httpRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "payment_http_request_duration_seconds",
		Help:    "HTTP request latency in seconds.",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "path"})

	paymentsProcessedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "payments_processed_total",
		Help: "Total payments processed by status (success/failure).",
	}, []string{"status"})
)

func PrometheusMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		duration := time.Since(start).Seconds()
		status := http.StatusText(c.Writer.Status())
		httpRequestsTotal.WithLabelValues(c.Request.Method, c.FullPath(), status).Inc()
		httpRequestDuration.WithLabelValues(c.Request.Method, c.FullPath()).Observe(duration)
	}
}

type PaymentHandler struct {
	processPayment port.ProcessPaymentUseCase
	refund         port.RefundUseCase
	cancelPayment  port.CancelPaymentUseCase
	getPayment     port.GetPaymentUseCase
}

func NewPaymentHandler(
	processPayment port.ProcessPaymentUseCase,
	refund port.RefundUseCase,
	cancelPayment port.CancelPaymentUseCase,
	getPayment port.GetPaymentUseCase,
) *PaymentHandler {
	return &PaymentHandler{
		processPayment: processPayment,
		refund:         refund,
		cancelPayment:  cancelPayment,
		getPayment:     getPayment,
	}
}

func (h *PaymentHandler) RegisterRoutes(router *gin.Engine, apiGroup *gin.RouterGroup) {
	router.GET("/health", h.HealthCheck)
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	payments := apiGroup.Group("/payments")
	payments.POST("", h.CreatePayment)
	payments.GET("/:id", h.GetPayment)
	payments.POST("/:id/refund", h.RefundPayment)
	payments.DELETE("/:id", h.CancelPayment)
}

func (h *PaymentHandler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"service": "payment-service",
	})
}

type createPaymentRequest struct {
	OrderID       string               `json:"order_id"       binding:"required"`
	Amount        float64              `json:"amount"         binding:"required,gt=0"`
	Currency      string               `json:"currency"       binding:"required,len=3"`
	PaymentMethod entity.PaymentMethod `json:"payment_method" binding:"required"`
}

func (h *PaymentHandler) CreatePayment(c *gin.Context) {
	var req createPaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	output, err := h.processPayment.Execute(port.ProcessPaymentInput{
		OrderID:       req.OrderID,
		Amount:        req.Amount,
		Currency:      req.Currency,
		PaymentMethod: req.PaymentMethod,
	})
	if err != nil {
		paymentsProcessedTotal.WithLabelValues("failure").Inc()
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}

	paymentsProcessedTotal.WithLabelValues("success").Inc()
	c.JSON(http.StatusCreated, gin.H{
		"payment_id": output.PaymentID,
		"status":     output.Status,
		"fee":        output.Fee,
		"total":      output.Total,
	})
}

func (h *PaymentHandler) GetPayment(c *gin.Context) {
	paymentID := c.Param("id")
	payment, err := h.getPayment.Execute(paymentID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "payment not found"})
		return
	}
	c.JSON(http.StatusOK, payment)
}

type refundRequest struct {
	Reason string `json:"reason"`
}

func (h *PaymentHandler) RefundPayment(c *gin.Context) {
	paymentID := c.Param("id")
	var req refundRequest
	_ = c.ShouldBindJSON(&req)

	output, err := h.refund.Execute(port.RefundInput{
		PaymentID: paymentID,
		Reason:    req.Reason,
	})
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"payment_id":      output.PaymentID,
		"status":          output.Status,
		"refunded_amount": output.RefundedAmount,
	})
}

func (h *PaymentHandler) CancelPayment(c *gin.Context) {
	paymentID := c.Param("id")
	if err := h.cancelPayment.Execute(paymentID); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "payment cancelled successfully"})
}
