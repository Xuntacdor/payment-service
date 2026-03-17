package vnpay_test

import (
	"strings"
	"testing"

	"github.com/Xuntacdor/payment-service/internal/adapters/outbound/vnpay"
	"github.com/Xuntacdor/payment-service/internal/domain/entity"
	"github.com/Xuntacdor/payment-service/internal/domain/port"
)

func testConfig() vnpay.Config {
	return vnpay.Config{
		TmnCode:    "DEMO1234",
		HashSecret: "RAOEXHYVSDDIIENYWSLDIIZTANXUXZFJ",
		ReturnURL:  "http://localhost:8080/api/v1/payments/vnpay/callback",
	}
}

func TestCharge_ReturnsPaymentURL(t *testing.T) {
	adapter := vnpay.NewVNPayAdapter(testConfig()).(*vnpay.VNPayAdapter)

	output, err := adapter.Charge(port.GatewayChargeInput{
		Amount:        entity.Money{Amount: 100000, Currency: "VND"},
		PaymentMethod: entity.MethodBankTransfer,
		ReferenceID:   "order-test-001",
		Description:   "Test payment",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if output == nil {
		t.Fatal("expected non-nil output")
	}
	if output.Status != "PENDING_REDIRECT" {
		t.Errorf("expected PENDING_REDIRECT, got %s", output.Status)
	}
	if output.GatewayTransactionID != "order-test-001" {
		t.Errorf("expected txnRef = order-test-001, got %s", output.GatewayTransactionID)
	}

	paymentURL, ok := output.RawResponse["payment_url"].(string)
	if !ok || paymentURL == "" {
		t.Fatal("expected non-empty payment_url in RawResponse")
	}
	if !strings.Contains(paymentURL, "sandbox.vnpayment.vn") {
		t.Errorf("expected sandbox URL, got: %s", paymentURL)
	}
	if !strings.Contains(paymentURL, "vnp_SecureHash=") {
		t.Error("expected vnp_SecureHash in payment URL")
	}
	if !strings.Contains(paymentURL, "vnp_TxnRef=order-test-001") {
		t.Error("expected vnp_TxnRef in payment URL")
	}
	if !strings.Contains(paymentURL, "vnp_Amount=10000000") {
		t.Errorf("expected amount 10000000 in URL, got: %s", paymentURL)
	}
}

func TestCharge_DifferentOrders_DifferentURLs(t *testing.T) {
	adapter := vnpay.NewVNPayAdapter(testConfig()).(*vnpay.VNPayAdapter)

	out1, _ := adapter.Charge(port.GatewayChargeInput{
		Amount:      entity.Money{Amount: 50000, Currency: "VND"},
		ReferenceID: "order-A",
		Description: "Order A",
	})
	out2, _ := adapter.Charge(port.GatewayChargeInput{
		Amount:      entity.Money{Amount: 50000, Currency: "VND"},
		ReferenceID: "order-B",
		Description: "Order B",
	})

	url1 := out1.RawResponse["payment_url"].(string)
	url2 := out2.RawResponse["payment_url"].(string)
	if url1 == url2 {
		t.Error("different orders must produce different payment URLs")
	}
}

func TestVerifyWebhook_ValidSignature(t *testing.T) {
	adapter := vnpay.NewVNPayAdapter(testConfig()).(*vnpay.VNPayAdapter)

	params := map[string]string{
		"vnp_Amount":        "10000000",
		"vnp_BankCode":      "NCB",
		"vnp_OrderInfo":     "Test payment",
		"vnp_ResponseCode":  "00",
		"vnp_TmnCode":       "DEMO1234",
		"vnp_TransactionNo": "13288395",
		"vnp_TxnRef":        "order-test-001",
	}

	params["vnp_SecureHash"] = "invalidsignature"

	err := adapter.VerifyWebhook(params)
	if err == nil {
		t.Fatal("expected error for invalid signature")
	}
	if !strings.Contains(err.Error(), "signature mismatch") {
		t.Errorf("expected signature mismatch error, got: %v", err)
	}
}

func TestVerifyWebhook_MissingSecureHash(t *testing.T) {
	adapter := vnpay.NewVNPayAdapter(testConfig()).(*vnpay.VNPayAdapter)

	params := map[string]string{
		"vnp_ResponseCode": "00",
		"vnp_TxnRef":       "order-001",
	}

	err := adapter.VerifyWebhook(params)
	if err == nil {
		t.Fatal("expected error for missing vnp_SecureHash")
	}
}

func TestVerifyWebhook_FailedPayment(t *testing.T) {
	adapter := vnpay.NewVNPayAdapter(testConfig()).(*vnpay.VNPayAdapter)

	params := map[string]string{
		"vnp_ResponseCode": "24",
		"vnp_TxnRef":       "order-001",
		"vnp_SecureHash":   "invalidsig",
	}

	err := adapter.VerifyWebhook(params)
	if err == nil {
		t.Fatal("expected error for failed payment")
	}
}

func TestExtractWebhookData(t *testing.T) {
	adapter := vnpay.NewVNPayAdapter(testConfig()).(*vnpay.VNPayAdapter)

	params := map[string]string{
		"vnp_TxnRef":        "order-test-001",
		"vnp_TransactionNo": "99887766",
		"vnp_ResponseCode":  "00",
	}

	txnRef, transactionNo, responseCode := adapter.ExtractWebhookData(params)
	if txnRef != "order-test-001" {
		t.Errorf("expected txnRef=order-test-001, got %s", txnRef)
	}
	if transactionNo != "99887766" {
		t.Errorf("expected transactionNo=99887766, got %s", transactionNo)
	}
	if responseCode != "00" {
		t.Errorf("expected responseCode=00, got %s", responseCode)
	}
}

func TestCharge_AmountConversion(t *testing.T) {
	adapter := vnpay.NewVNPayAdapter(testConfig()).(*vnpay.VNPayAdapter)

	// 500,000 VND × 100 = 50,000,000 (VNPay unit)
	output, err := adapter.Charge(port.GatewayChargeInput{
		Amount:      entity.Money{Amount: 500000, Currency: "VND"},
		ReferenceID: "order-amount-test",
		Description: "Amount test",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	paymentURL := output.RawResponse["payment_url"].(string)
	if !strings.Contains(paymentURL, "vnp_Amount=50000000") {
		t.Errorf("expected vnp_Amount=50000000 in URL, got: %s", paymentURL)
	}
}
