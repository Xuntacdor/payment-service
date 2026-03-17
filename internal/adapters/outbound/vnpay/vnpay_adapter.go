package vnpay

import (
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Xuntacdor/payment-service/internal/domain/port"
)

// ---- VNPay Constants ----

const (
	vnpVersion   = "2.1.0"
	vnpCommand   = "pay"
	vnpCurrCode  = "VND"
	vnpLocale    = "vn"
	vnpOrderType = "other"

	// VNPay response codes
	ResponseSuccess = "00"
	ResponseRefund  = "02"

	// Sandbox URL — swap with https://pay.vnpay.vn for production
	sandboxURL = "https://sandbox.vnpayment.vn/paymentv2/vpcpay.html"
	queryURL   = "https://sandbox.vnpayment.vn/merchant_webapi/api/transaction"
	refundURL  = "https://sandbox.vnpayment.vn/merchant_webapi/api/transaction"
)

// ---- Config ----

// Config holds all VNPay merchant credentials.
// All values come from the VNPay merchant portal.
type Config struct {
	TmnCode    string // Terminal / Merchant Code
	HashSecret string // Secret key for HMAC-SHA512 signature
	ReturnURL  string // URL VNPay redirects to after payment
	APIURL     string // Override for production (optional)
}

// ---- VNPayAdapter ----

// VNPayAdapter implements PaymentGatewayPort using VNPay's payment API.
// VNPay flow is redirect-based:
//  1. Charge() → builds a payment URL → user is redirected to VNPay
//  2. VNPay POSTs result to ReturnURL (webhook)
//  3. GetTransaction() → verifies the transaction status
//  4. Refund() → calls VNPay refund API directly
type VNPayAdapter struct {
	config     Config
	httpClient *http.Client
}

// NewVNPayAdapter creates a new VNPay outbound adapter.
// Usage in main.go:
//
//	gateway := vnpay.NewVNPayAdapter(vnpay.Config{
//	    TmnCode:    cfg.VNPay.TmnCode,
//	    HashSecret: cfg.VNPay.HashSecret,
//	    ReturnURL:  cfg.VNPay.ReturnURL,
//	})
func NewVNPayAdapter(config Config) port.PaymentGatewayPort {
	return &VNPayAdapter{
		config: config,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ---- Charge ----

// Charge builds a VNPay payment URL that the frontend redirects the user to.
// VNPay does NOT charge synchronously — the result comes via webhook/ReturnURL.
// The GatewayTransactionID returned is the vnp_TxnRef (= ReferenceID / orderID).
func (a *VNPayAdapter) Charge(input port.GatewayChargeInput) (*port.GatewayChargeOutput, error) {
	// VNPay requires amount in VND (integer, no decimal), multiplied by 100
	amountVND := int64(input.Amount.Amount * 100)

	now := time.Now()
	createDate := now.Format("20060102150405") // yyyyMMddHHmmss
	expireDate := now.Add(15 * time.Minute).Format("20060102150405")

	// Build VNPay parameter map (all keys must be sorted for signature)
	params := map[string]string{
		"vnp_Version":    vnpVersion,
		"vnp_Command":    vnpCommand,
		"vnp_TmnCode":    a.config.TmnCode,
		"vnp_Amount":     strconv.FormatInt(amountVND, 10),
		"vnp_CurrCode":   vnpCurrCode,
		"vnp_TxnRef":     input.ReferenceID, // = orderID (idempotency key)
		"vnp_OrderInfo":  input.Description,
		"vnp_OrderType":  vnpOrderType,
		"vnp_Locale":     vnpLocale,
		"vnp_ReturnUrl":  a.config.ReturnURL,
		"vnp_IpAddr":     "127.0.0.1", // in production: extract from HTTP request context
		"vnp_CreateDate": createDate,
		"vnp_ExpireDate": expireDate,
	}

	// Build signed payment URL
	paymentURL, err := a.buildSignedURL(params)
	if err != nil {
		return nil, fmt.Errorf("vnpay: failed to build payment URL: %w", err)
	}

	return &port.GatewayChargeOutput{
		// For redirect-based gateways, the "transaction ID" is the TxnRef
		// The actual VNPay transaction ID (vnp_TransactionNo) arrives in the webhook
		GatewayTransactionID: input.ReferenceID,
		Status:               "PENDING_REDIRECT",
		RawResponse: map[string]interface{}{
			"payment_url": paymentURL,
			"txn_ref":     input.ReferenceID,
			"expire_date": expireDate,
		},
	}, nil
}

// ---- GetTransaction ----

// GetTransaction queries VNPay for the current status of a transaction.
// Call this from your webhook handler after VNPay posts the result.
func (a *VNPayAdapter) GetTransaction(txnRef string) (*port.GatewayChargeOutput, error) {
	now := time.Now()
	requestID := fmt.Sprintf("%d", now.UnixNano())
	transDate := now.Format("20060102150405")

	payload := map[string]string{
		"vnp_RequestId":  requestID,
		"vnp_Version":    vnpVersion,
		"vnp_Command":    "querydr",
		"vnp_TmnCode":    a.config.TmnCode,
		"vnp_TxnRef":     txnRef,
		"vnp_OrderInfo":  fmt.Sprintf("Query transaction %s", txnRef),
		"vnp_TransDate":  transDate,
		"vnp_CreateDate": transDate,
		"vnp_IpAddr":     "127.0.0.1",
	}

	// Sign and call VNPay query API
	payload["vnp_SecureHash"] = a.signPayload(payload)

	respBody, err := a.postJSON(queryURL, payload)
	if err != nil {
		return nil, fmt.Errorf("vnpay: query transaction failed: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("vnpay: failed to parse query response: %w", err)
	}

	responseCode, _ := result["vnp_ResponseCode"].(string)
	transactionNo, _ := result["vnp_TransactionNo"].(string)
	status := a.mapResponseCode(responseCode)

	return &port.GatewayChargeOutput{
		GatewayTransactionID: transactionNo,
		Status:               status,
		RawResponse:          result,
	}, nil
}

// ---- Refund ----

// Refund calls the VNPay refund API to return funds to the customer.
// Requires the original vnp_TransactionNo from the captured payment.
func (a *VNPayAdapter) Refund(input port.GatewayRefundInput) (*port.GatewayChargeOutput, error) {
	now := time.Now()
	requestID := fmt.Sprintf("%d", now.UnixNano())
	transDate := now.Format("20060102150405")
	amountVND := int64(input.Amount.Amount * 100)

	// Determine refund type: "02" = full refund, "03" = partial refund
	refundType := "02"

	payload := map[string]string{
		"vnp_RequestId":       requestID,
		"vnp_Version":         vnpVersion,
		"vnp_Command":         "refund",
		"vnp_TmnCode":         a.config.TmnCode,
		"vnp_TransactionType": refundType,
		"vnp_TxnRef":          input.GatewayTransactionID,
		"vnp_Amount":          strconv.FormatInt(amountVND, 10),
		"vnp_OrderInfo":       fmt.Sprintf("Refund for transaction %s: %s", input.GatewayTransactionID, input.Reason),
		"vnp_TransDate":       transDate,
		"vnp_CreateDate":      transDate,
		"vnp_CreateBy":        "system",
		"vnp_IpAddr":          "127.0.0.1",
	}

	payload["vnp_SecureHash"] = a.signPayload(payload)

	respBody, err := a.postJSON(refundURL, payload)
	if err != nil {
		return nil, fmt.Errorf("vnpay: refund API call failed: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("vnpay: failed to parse refund response: %w", err)
	}

	responseCode, _ := result["vnp_ResponseCode"].(string)
	if responseCode != ResponseSuccess {
		return nil, fmt.Errorf("vnpay: refund rejected, response code: %s", responseCode)
	}

	transactionNo, _ := result["vnp_TransactionNo"].(string)

	return &port.GatewayChargeOutput{
		GatewayTransactionID: transactionNo,
		Status:               "REFUNDED",
		RawResponse:          result,
	}, nil
}

// ---- Webhook Verification ----

// VerifyWebhook validates the HMAC-SHA512 signature on VNPay's IPN/ReturnURL callback.
// Call this in your webhook HTTP handler before processing the payment result.
//
// Example usage in a Gin handler:
//
//	params := c.QueryMap() // or parse from POST body
//	if err := adapter.VerifyWebhook(params); err != nil {
//	    c.JSON(400, gin.H{"error": "invalid signature"})
//	    return
//	}
//	// safe to process
func (a *VNPayAdapter) VerifyWebhook(params map[string]string) error {
	receivedHash, ok := params["vnp_SecureHash"]
	if !ok {
		return fmt.Errorf("vnpay: missing vnp_SecureHash in callback")
	}

	// Remove hash fields before re-computing signature
	filtered := make(map[string]string)
	for k, v := range params {
		if k != "vnp_SecureHash" && k != "vnp_SecureHashType" {
			filtered[k] = v
		}
	}

	expectedHash := a.signPayload(filtered)
	if !strings.EqualFold(receivedHash, expectedHash) {
		return fmt.Errorf("vnpay: signature mismatch — possible tampering")
	}

	responseCode := params["vnp_ResponseCode"]
	if responseCode != ResponseSuccess {
		return fmt.Errorf("vnpay: payment failed with response code: %s", responseCode)
	}

	return nil
}

// ExtractWebhookData extracts key fields from a verified VNPay webhook payload.
// Returns the txnRef (= orderID) and VNPay's transaction number.
func (a *VNPayAdapter) ExtractWebhookData(params map[string]string) (txnRef, transactionNo, responseCode string) {
	return params["vnp_TxnRef"], params["vnp_TransactionNo"], params["vnp_ResponseCode"]
}

// ---- Private Helpers ----

// buildSignedURL constructs the full VNPay redirect URL with HMAC-SHA512 signature.
func (a *VNPayAdapter) buildSignedURL(params map[string]string) (string, error) {
	// Sort keys alphabetically — VNPay requires this for consistent signing
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Build query string for signing (URL-encoded values)
	var queryParts []string
	for _, k := range keys {
		queryParts = append(queryParts, fmt.Sprintf("%s=%s", k, url.QueryEscape(params[k])))
	}
	queryString := strings.Join(queryParts, "&")

	// Compute HMAC-SHA512 signature
	signature := a.hmacSHA512(a.config.HashSecret, queryString)

	// Append signature to URL
	baseURL := sandboxURL
	if a.config.APIURL != "" {
		baseURL = a.config.APIURL
	}

	return fmt.Sprintf("%s?%s&vnp_SecureHash=%s", baseURL, queryString, signature), nil
}

// signPayload builds the canonical string and returns HMAC-SHA512 hex signature.
// Used for API calls (query, refund) — NOT the redirect URL.
func (a *VNPayAdapter) signPayload(params map[string]string) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		if k != "vnp_SecureHash" && k != "vnp_SecureHashType" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)

	var parts []string
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", k, params[k]))
	}
	data := strings.Join(parts, "&")
	return a.hmacSHA512(a.config.HashSecret, data)
}

// hmacSHA512 computes HMAC-SHA512 and returns the lowercase hex string.
func (a *VNPayAdapter) hmacSHA512(secret, data string) string {
	h := hmac.New(sha512.New, []byte(secret))
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

// postJSON sends a JSON POST request to VNPay's API and returns the raw response body.
func (a *VNPayAdapter) postJSON(apiURL string, payload map[string]string) ([]byte, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal failed: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, apiURL, strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("create request failed: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("vnpay API returned status %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

// mapResponseCode translates VNPay response codes to human-readable status strings.
// Full list: https://sandbox.vnpayment.vn/apis/docs/huong-dan-tich-hop
func (a *VNPayAdapter) mapResponseCode(code string) string {
	switch code {
	case "00":
		return "SUCCESS"
	case "07":
		return "SUSPICIOUS_TRANSACTION"
	case "09":
		return "CARD_NOT_REGISTERED_INTERNET"
	case "10":
		return "WRONG_AUTH_TOO_MANY_TIMES"
	case "11":
		return "PAYMENT_EXPIRED"
	case "12":
		return "CARD_LOCKED"
	case "13":
		return "WRONG_OTP"
	case "24":
		return "CANCELLED_BY_CUSTOMER"
	case "51":
		return "INSUFFICIENT_FUNDS"
	case "65":
		return "EXCEEDED_DAILY_LIMIT"
	case "75":
		return "BANK_UNDER_MAINTENANCE"
	case "79":
		return "WRONG_PIN_TOO_MANY_TIMES"
	case "99":
		return "UNKNOWN_ERROR"
	default:
		return fmt.Sprintf("UNKNOWN_%s", code)
	}
}
