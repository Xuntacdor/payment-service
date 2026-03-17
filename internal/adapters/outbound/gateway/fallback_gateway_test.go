package gateway_test

import (
	"fmt"
	"testing"

	"go.uber.org/zap"

	"github.com/Xuntacdor/payment-service/internal/adapters/outbound/gateway"
	"github.com/Xuntacdor/payment-service/internal/domain/entity"
	"github.com/Xuntacdor/payment-service/internal/domain/port"
)

// ---- Mock Gateway ----

type mockGateway struct {
	name       string
	shouldFail bool
	callCount  int
}

func (m *mockGateway) Charge(input port.GatewayChargeInput) (*port.GatewayChargeOutput, error) {
	m.callCount++
	if m.shouldFail {
		return nil, fmt.Errorf("%s: charge failed (simulated)", m.name)
	}
	return &port.GatewayChargeOutput{
		GatewayTransactionID: fmt.Sprintf("txn_%s_%s", m.name, input.ReferenceID),
		Status:               "success",
	}, nil
}

func (m *mockGateway) Refund(input port.GatewayRefundInput) (*port.GatewayChargeOutput, error) {
	m.callCount++
	if m.shouldFail {
		return nil, fmt.Errorf("%s: refund failed (simulated)", m.name)
	}
	return &port.GatewayChargeOutput{
		GatewayTransactionID: fmt.Sprintf("refund_%s", input.GatewayTransactionID),
		Status:               "refunded",
	}, nil
}

func (m *mockGateway) GetTransaction(id string) (*port.GatewayChargeOutput, error) {
	if m.shouldFail {
		return nil, fmt.Errorf("%s: get transaction failed", m.name)
	}
	return &port.GatewayChargeOutput{GatewayTransactionID: id, Status: "success"}, nil
}

// ---- Test helpers ----

func testInput() port.GatewayChargeInput {
	return port.GatewayChargeInput{
		Amount:        entity.Money{Amount: 100.0, Currency: "USD"},
		PaymentMethod: entity.MethodCard,
		ReferenceID:   "order-001",
		Description:   "Test",
	}
}

func buildRegistry(entries ...gateway.GatewayEntry) *gateway.Registry {
	r := gateway.NewRegistry()
	for _, e := range entries {
		r.Register(e)
	}
	return r
}

func noopLogger() *zap.Logger {
	logger, _ := zap.NewDevelopment()
	return logger
}

// ---- Tests ----

func TestFallback_FirstGatewaySucceeds(t *testing.T) {
	stripe := &mockGateway{name: "stripe"}
	vnpay := &mockGateway{name: "vnpay"}

	registry := buildRegistry(
		gateway.GatewayEntry{
			Name: gateway.GatewayStripe, Gateway: stripe,
			Priority: 1, Enabled: true,
			SupportedMethods:    []entity.PaymentMethod{entity.MethodCard},
			SupportedCurrencies: []string{"USD"},
		},
		gateway.GatewayEntry{
			Name: gateway.GatewayVNPay, Gateway: vnpay,
			Priority: 2, Enabled: true,
			SupportedMethods:    []entity.PaymentMethod{entity.MethodCard},
			SupportedCurrencies: []string{"USD"},
		},
	)

	gw := gateway.NewFallbackGateway(registry, noopLogger())
	output, err := gw.Charge(testInput())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stripe.callCount != 1 {
		t.Errorf("expected stripe called once, got %d", stripe.callCount)
	}
	if vnpay.callCount != 0 {
		t.Error("expected vnpay NOT called when stripe succeeds")
	}
	// Output should be tagged with gateway name
	if output.RawResponse["gateway_used"] != "STRIPE" {
		t.Errorf("expected gateway_used=STRIPE, got %v", output.RawResponse["gateway_used"])
	}
	// Transaction ID should be prefixed
	if output.GatewayTransactionID[:7] != "STRIPE:" {
		t.Errorf("expected STRIPE: prefix, got %s", output.GatewayTransactionID)
	}
}

func TestFallback_FirstFailsSecondSucceeds(t *testing.T) {
	stripe := &mockGateway{name: "stripe", shouldFail: true}
	vnpay := &mockGateway{name: "vnpay", shouldFail: false}

	registry := buildRegistry(
		gateway.GatewayEntry{
			Name: gateway.GatewayStripe, Gateway: stripe,
			Priority: 1, Enabled: true,
			SupportedMethods:    []entity.PaymentMethod{entity.MethodCard},
			SupportedCurrencies: []string{"USD"},
		},
		gateway.GatewayEntry{
			Name: gateway.GatewayVNPay, Gateway: vnpay,
			Priority: 2, Enabled: true,
			SupportedMethods:    []entity.PaymentMethod{entity.MethodCard},
			SupportedCurrencies: []string{"USD"},
		},
	)

	gw := gateway.NewFallbackGateway(registry, noopLogger())
	output, err := gw.Charge(testInput())

	if err != nil {
		t.Fatalf("expected fallback to succeed, got error: %v", err)
	}
	if stripe.callCount != 1 {
		t.Errorf("expected stripe attempted once, got %d", stripe.callCount)
	}
	if vnpay.callCount != 1 {
		t.Errorf("expected vnpay attempted once as fallback, got %d", vnpay.callCount)
	}
	if output.RawResponse["gateway_used"] != "VNPAY" {
		t.Errorf("expected gateway_used=VNPAY, got %v", output.RawResponse["gateway_used"])
	}
}

func TestFallback_AllGatewaysFail(t *testing.T) {
	stripe := &mockGateway{name: "stripe", shouldFail: true}
	vnpay := &mockGateway{name: "vnpay", shouldFail: true}

	registry := buildRegistry(
		gateway.GatewayEntry{
			Name: gateway.GatewayStripe, Gateway: stripe,
			Priority: 1, Enabled: true,
			SupportedMethods:    []entity.PaymentMethod{entity.MethodCard},
			SupportedCurrencies: []string{"USD"},
		},
		gateway.GatewayEntry{
			Name: gateway.GatewayVNPay, Gateway: vnpay,
			Priority: 2, Enabled: true,
			SupportedMethods:    []entity.PaymentMethod{entity.MethodCard},
			SupportedCurrencies: []string{"USD"},
		},
	)

	gw := gateway.NewFallbackGateway(registry, noopLogger())
	_, err := gw.Charge(testInput())

	if err == nil {
		t.Fatal("expected error when all gateways fail")
	}
	if stripe.callCount != 1 || vnpay.callCount != 1 {
		t.Errorf("expected both gateways attempted, stripe=%d vnpay=%d", stripe.callCount, vnpay.callCount)
	}
}

func TestFallback_NoCompatibleGateway(t *testing.T) {
	stripe := &mockGateway{name: "stripe"}

	registry := buildRegistry(
		gateway.GatewayEntry{
			Name: gateway.GatewayStripe, Gateway: stripe,
			Priority: 1, Enabled: true,
			SupportedMethods:    []entity.PaymentMethod{entity.MethodCard},
			SupportedCurrencies: []string{"USD"}, // only USD
		},
	)

	gw := gateway.NewFallbackGateway(registry, noopLogger())

	// Try to pay in VND — no gateway supports it
	_, err := gw.Charge(port.GatewayChargeInput{
		Amount:        entity.Money{Amount: 500000, Currency: "VND"},
		PaymentMethod: entity.MethodBankTransfer,
		ReferenceID:   "order-002",
	})

	if err == nil {
		t.Fatal("expected error: no gateway supports VND BankTransfer")
	}
	if stripe.callCount != 0 {
		t.Error("stripe should not be called for unsupported currency")
	}
}

func TestFallback_DisabledGatewaySkipped(t *testing.T) {
	stripe := &mockGateway{name: "stripe"}
	vnpay := &mockGateway{name: "vnpay"}

	registry := buildRegistry(
		gateway.GatewayEntry{
			Name: gateway.GatewayStripe, Gateway: stripe,
			Priority: 1, Enabled: false, // DISABLED
			SupportedMethods:    []entity.PaymentMethod{entity.MethodCard},
			SupportedCurrencies: []string{"USD"},
		},
		gateway.GatewayEntry{
			Name: gateway.GatewayVNPay, Gateway: vnpay,
			Priority: 2, Enabled: true,
			SupportedMethods:    []entity.PaymentMethod{entity.MethodCard},
			SupportedCurrencies: []string{"USD"},
		},
	)

	gw := gateway.NewFallbackGateway(registry, noopLogger())
	_, err := gw.Charge(testInput())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stripe.callCount != 0 {
		t.Error("disabled stripe should not be called")
	}
	if vnpay.callCount != 1 {
		t.Error("enabled vnpay should be called")
	}
}

func TestFallback_RefundRoutesToCorrectGateway(t *testing.T) {
	stripe := &mockGateway{name: "stripe"}
	vnpay := &mockGateway{name: "vnpay"}

	registry := buildRegistry(
		gateway.GatewayEntry{Name: gateway.GatewayStripe, Gateway: stripe, Priority: 1, Enabled: true},
		gateway.GatewayEntry{Name: gateway.GatewayVNPay, Gateway: vnpay, Priority: 2, Enabled: true},
	)

	gw := gateway.NewFallbackGateway(registry, noopLogger())

	// Simulate refunding a VNPay transaction (prefixed txnID)
	_, err := gw.Refund(port.GatewayRefundInput{
		GatewayTransactionID: "VNPAY:13288395", // prefix routes to VNPay
		Amount:               entity.Money{Amount: 100.0, Currency: "VND"},
		Reason:               "customer request",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if vnpay.callCount != 1 {
		t.Errorf("expected vnpay called for refund, got %d", vnpay.callCount)
	}
	if stripe.callCount != 0 {
		t.Error("stripe should NOT be called for a VNPay refund")
	}
}

func TestFallback_PriorityOrder(t *testing.T) {
	calls := []string{}

	makeGateway := func(name string, fail bool) port.PaymentGatewayPort {
		return &trackingGateway{
			name:       name,
			shouldFail: fail,
			calls:      &calls,
		}
	}

	registry := buildRegistry(
		gateway.GatewayEntry{
			Name: gateway.GatewayMoMo, Gateway: makeGateway("momo", true),
			Priority: 3, Enabled: true,
			SupportedMethods:    []entity.PaymentMethod{entity.MethodCard},
			SupportedCurrencies: []string{"USD"},
		},
		gateway.GatewayEntry{
			Name: gateway.GatewayStripe, Gateway: makeGateway("stripe", true),
			Priority: 1, Enabled: true,
			SupportedMethods:    []entity.PaymentMethod{entity.MethodCard},
			SupportedCurrencies: []string{"USD"},
		},
		gateway.GatewayEntry{
			Name: gateway.GatewayVNPay, Gateway: makeGateway("vnpay", false),
			Priority: 2, Enabled: true,
			SupportedMethods:    []entity.PaymentMethod{entity.MethodCard},
			SupportedCurrencies: []string{"USD"},
		},
	)

	gw := gateway.NewFallbackGateway(registry, noopLogger())
	_, err := gw.Charge(testInput())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should be tried in order: stripe(1) → vnpay(2), stop at vnpay (success)
	if len(calls) != 2 || calls[0] != "stripe" || calls[1] != "vnpay" {
		t.Errorf("expected [stripe, vnpay], got %v", calls)
	}
}

// trackingGateway records call order for priority tests
type trackingGateway struct {
	name       string
	shouldFail bool
	calls      *[]string
}

func (g *trackingGateway) Charge(input port.GatewayChargeInput) (*port.GatewayChargeOutput, error) {
	*g.calls = append(*g.calls, g.name)
	if g.shouldFail {
		return nil, fmt.Errorf("%s failed", g.name)
	}
	return &port.GatewayChargeOutput{GatewayTransactionID: "txn_" + g.name, Status: "success"}, nil
}

func (g *trackingGateway) Refund(input port.GatewayRefundInput) (*port.GatewayChargeOutput, error) {
	return &port.GatewayChargeOutput{}, nil
}

func (g *trackingGateway) GetTransaction(id string) (*port.GatewayChargeOutput, error) {
	return &port.GatewayChargeOutput{}, nil
}
