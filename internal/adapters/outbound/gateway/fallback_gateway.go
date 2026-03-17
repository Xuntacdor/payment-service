package gateway

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/Xuntacdor/payment-service/internal/domain/entity"
	"github.com/Xuntacdor/payment-service/internal/domain/port"
	"github.com/sony/gobreaker/v2"
)

// AttemptResult records the outcome of a single gateway attempt
type AttemptResult struct {
	Gateway  GatewayName
	Error    string
	Duration time.Duration
}

// FallbackGateway implements PaymentGatewayPort.
// It tries gateways in Priority order and automatically falls back
// to the next one when the current gateway fails.
//
// Example setup in main.go:
//
//	registry := gateway.NewRegistry().
//	    Register(gateway.GatewayEntry{
//	        Name: gateway.GatewayStripe, Gateway: stripeAdapter,
//	        Priority: 1, Enabled: true,
//	        SupportedMethods:    []entity.PaymentMethod{entity.MethodCard},
//	        SupportedCurrencies: []string{"USD", "EUR"},
//	    }).
//	    Register(gateway.GatewayEntry{
//	        Name: gateway.GatewayVNPay, Gateway: vnpayAdapter,
//	        Priority: 2, Enabled: true,
//	        SupportedMethods:    []entity.PaymentMethod{entity.MethodBankTransfer, entity.MethodWallet},
//	        SupportedCurrencies: []string{"VND"},
//	    })
//
//	gw := gateway.NewFallbackGateway(registry, logger)
type FallbackGateway struct {
	registry *Registry
	logger   *zap.Logger
	breakers map[GatewayName]*gobreaker.CircuitBreaker[*port.GatewayChargeOutput]
}

// NewFallbackGateway creates a FallbackGateway that implements port.PaymentGatewayPort
func NewFallbackGateway(registry *Registry, logger *zap.Logger) port.PaymentGatewayPort {
	breakers := make(map[GatewayName]*gobreaker.CircuitBreaker[*port.GatewayChargeOutput])
	for _, entry := range registry.All() {
		st := gobreaker.Settings{
			Name:        string(entry.Name),
			MaxRequests: 3,
			Interval:    10 * time.Second,
			Timeout:     30 * time.Second,
			ReadyToTrip: func(counts gobreaker.Counts) bool {
				failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
				return counts.Requests >= 3 && failureRatio >= 0.6
			},
		}
		breakers[entry.Name] = gobreaker.NewCircuitBreaker[*port.GatewayChargeOutput](st)
	}

	return &FallbackGateway{
		registry: registry,
		logger:   logger,
		breakers: breakers,
	}
}

// Charge tries each compatible gateway in Priority order.
// On success, prefixes the transaction ID with the gateway name: "STRIPE:ch_xxx".
// This prefix is used later by Refund() and GetTransaction() to route back.
func (f *FallbackGateway) Charge(input port.GatewayChargeInput) (*port.GatewayChargeOutput, error) {
	candidates := f.rankCandidates(input.PaymentMethod, input.Amount.Currency)
	if len(candidates) == 0 {
		return nil, fmt.Errorf(
			"no gateway available for method=%s currency=%s",
			input.PaymentMethod, input.Amount.Currency,
		)
	}

	var attempts []AttemptResult

	for _, entry := range candidates {
		start := time.Now()
		f.logger.Info("attempting charge",
			zap.String("gateway", string(entry.Name)),
			zap.String("orderID", input.ReferenceID),
			zap.Float64("amount", input.Amount.Amount),
			zap.String("currency", input.Amount.Currency),
		)

		cb := f.breakers[entry.Name]
		output, err := cb.Execute(func() (*port.GatewayChargeOutput, error) {
			return entry.Gateway.Charge(input)
		})
		duration := time.Since(start)

		if err == nil {
			f.logger.Info("charge succeeded",
				zap.String("gateway", string(entry.Name)),
				zap.String("txnID", output.GatewayTransactionID),
				zap.Duration("duration", duration),
			)
			if output.RawResponse == nil {
				output.RawResponse = make(map[string]interface{})
			}
			output.RawResponse["gateway_used"] = string(entry.Name)
			// Prefix txnID so Refund/GetTransaction can route correctly
			output.GatewayTransactionID = fmt.Sprintf("%s:%s", entry.Name, output.GatewayTransactionID)
			return output, nil
		}

		f.logger.Warn("gateway charge failed — trying next",
			zap.String("gateway", string(entry.Name)),
			zap.Error(err),
			zap.Duration("duration", duration),
		)
		attempts = append(attempts, AttemptResult{
			Gateway:  entry.Name,
			Error:    err.Error(),
			Duration: duration,
		})
	}

	return nil, fmt.Errorf("all gateways failed for order %s: %s",
		input.ReferenceID, summarizeAttempts(attempts))
}

// Refund routes to the gateway that processed the original charge
// by parsing the "STRIPE:ch_xxx" prefix in GatewayTransactionID.
func (f *FallbackGateway) Refund(input port.GatewayRefundInput) (*port.GatewayChargeOutput, error) {
	gatewayName, cleanTxID := parseGatewayPrefix(input.GatewayTransactionID)
	if gatewayName != "" {
		gw, err := f.registry.Get(gatewayName)
		if err != nil {
			return nil, fmt.Errorf("refund: gateway %s not available: %w", gatewayName, err)
		}
		input.GatewayTransactionID = cleanTxID
		f.logger.Info("routing refund",
			zap.String("gateway", string(gatewayName)),
			zap.String("txnID", cleanTxID),
		)
		return gw.Refund(input)
	}

	// No prefix — try all gateways in order
	var attempts []AttemptResult
	for _, entry := range f.rankCandidates("", "") {
		output, err := entry.Gateway.Refund(input)
		if err == nil {
			return output, nil
		}
		attempts = append(attempts, AttemptResult{Gateway: entry.Name, Error: err.Error()})
	}
	return nil, fmt.Errorf("all gateways failed for refund: %s", summarizeAttempts(attempts))
}

// GetTransaction routes to the correct gateway using the txnID prefix.
func (f *FallbackGateway) GetTransaction(gatewayTransactionID string) (*port.GatewayChargeOutput, error) {
	gatewayName, cleanTxID := parseGatewayPrefix(gatewayTransactionID)
	if gatewayName != "" {
		gw, err := f.registry.Get(gatewayName)
		if err != nil {
			return nil, fmt.Errorf("get transaction: gateway %s not available: %w", gatewayName, err)
		}
		return gw.GetTransaction(cleanTxID)
	}

	for _, entry := range f.rankCandidates("", "") {
		output, err := entry.Gateway.GetTransaction(gatewayTransactionID)
		if err == nil {
			return output, nil
		}
	}
	return nil, fmt.Errorf("transaction %s not found in any gateway", gatewayTransactionID)
}

// rankCandidates returns enabled gateways that support method+currency, sorted by Priority asc.
func (f *FallbackGateway) rankCandidates(method entity.PaymentMethod, currency string) []GatewayEntry {
	var candidates []GatewayEntry
	for _, entry := range f.registry.All() {
		if !entry.Enabled {
			continue
		}
		if method == "" && currency == "" {
			candidates = append(candidates, entry)
			continue
		}
		if entry.Supports(method, currency) {
			candidates = append(candidates, entry)
		}
	}
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Priority < candidates[j].Priority
	})
	return candidates
}

// parseGatewayPrefix extracts gateway name from "STRIPE:ch_123" → (GatewayStripe, "ch_123")
func parseGatewayPrefix(txID string) (GatewayName, string) {
	parts := strings.SplitN(txID, ":", 2)
	if len(parts) != 2 {
		return "", txID
	}
	name := GatewayName(strings.ToUpper(parts[0]))
	switch name {
	case GatewayStripe, GatewayVNPay, GatewayMoMo:
		return name, parts[1]
	}
	return "", txID
}

// summarizeAttempts formats failed attempts into a readable error string
func summarizeAttempts(attempts []AttemptResult) string {
	var parts []string
	for _, a := range attempts {
		parts = append(parts, fmt.Sprintf("[%s: %s (%s)]",
			a.Gateway, a.Error, a.Duration.Round(time.Millisecond)))
	}
	return strings.Join(parts, ", ")
}
