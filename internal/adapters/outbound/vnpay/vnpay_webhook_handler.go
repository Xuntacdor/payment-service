package vnpay

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// WebhookHandler handles VNPay's IPN (Instant Payment Notification) and ReturnURL callbacks.
// Register this in main.go alongside your payment routes.
//
// VNPay flow:
//  1. User pays on VNPay portal
//  2. VNPay redirects user back to ReturnURL with query params
//  3. VNPay also sends IPN (server-to-server) to your server
//  4. Both use the same params — verify signature, then update payment status
type WebhookHandler struct {
	adapter *VNPayAdapter
	// In production: inject your use case here to update payment status
	// e.g. capturePaymentUseCase port.CapturePaymentUseCase
}

// NewWebhookHandler creates the VNPay webhook handler
func NewWebhookHandler(adapter *VNPayAdapter) *WebhookHandler {
	return &WebhookHandler{adapter: adapter}
}

// RegisterRoutes registers the VNPay callback route
func (h *WebhookHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/payments/vnpay/callback", h.HandleCallback)
	rg.POST("/payments/vnpay/ipn", h.HandleIPN)
}

// HandleCallback processes the ReturnURL redirect from VNPay (user-facing).
// VNPay appends all params as query string: /callback?vnp_ResponseCode=00&vnp_TxnRef=...
func (h *WebhookHandler) HandleCallback(c *gin.Context) {
	params := make(map[string]string)
	for k, v := range c.Request.URL.Query() {
		if len(v) > 0 {
			params[k] = v[0]
		}
	}

	if err := h.adapter.VerifyWebhook(params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":  "payment verification failed",
			"detail": err.Error(),
		})
		return
	}

	txnRef, transactionNo, _ := h.adapter.ExtractWebhookData(params)

	// TODO: call your use case here to finalize the payment
	// e.g.: h.capturePaymentUseCase.Execute(txnRef, transactionNo)

	c.JSON(http.StatusOK, gin.H{
		"message":        "payment successful",
		"order_id":       txnRef,
		"transaction_no": transactionNo,
	})
}

// HandleIPN processes VNPay's server-to-server IPN notification.
// Must respond with {"RspCode":"00","Message":"Confirm Success"} within 5 seconds.
func (h *WebhookHandler) HandleIPN(c *gin.Context) {
	params := make(map[string]string)
	for k, v := range c.Request.URL.Query() {
		if len(v) > 0 {
			params[k] = v[0]
		}
	}

	if err := h.adapter.VerifyWebhook(params); err != nil {
		// VNPay expects this exact format on failure
		c.JSON(http.StatusOK, gin.H{
			"RspCode": "97",
			"Message": "Invalid Checksum",
		})
		return
	}

	txnRef, transactionNo, _ := h.adapter.ExtractWebhookData(params)

	// TODO: update order/payment status in DB
	// e.g.: h.capturePaymentUseCase.Execute(txnRef, transactionNo)
	_ = txnRef
	_ = transactionNo

	// VNPay requires this exact success response
	c.JSON(http.StatusOK, gin.H{
		"RspCode": "00",
		"Message": "Confirm Success",
	})
}
