package gateway

import (
	"fmt"

	"github.com/Xuntacdor/payment-service/internal/domain/entity"
	"github.com/Xuntacdor/payment-service/internal/domain/port"
)

// GatewayName identifies a specific payment gateway provider
type GatewayName string

const (
	GatewayStripe GatewayName = "STRIPE"
	GatewayVNPay  GatewayName = "VNPAY"
	GatewayMoMo   GatewayName = "MOMO"
)

// GatewayEntry wraps a gateway implementation with its metadata
type GatewayEntry struct {
	Name     GatewayName
	Gateway  port.PaymentGatewayPort
	Priority int // lower = higher priority (1 = first tried)
	Enabled  bool

	// SupportedMethods defines which PaymentMethods this gateway can handle.
	// Empty slice means it supports ALL methods.
	SupportedMethods []entity.PaymentMethod

	// SupportedCurrencies defines which currencies this gateway accepts.
	// Empty slice means it supports ALL currencies.
	SupportedCurrencies []string
}

// Supports returns true if this gateway can handle the given method and currency
func (e *GatewayEntry) Supports(method entity.PaymentMethod, currency string) bool {
	if e.Enabled == false {
		return false
	}

	// Check payment method
	if len(e.SupportedMethods) > 0 {
		methodOK := false
		for _, m := range e.SupportedMethods {
			if m == method {
				methodOK = true
				break
			}
		}
		if !methodOK {
			return false
		}
	}

	// Check currency
	if len(e.SupportedCurrencies) > 0 {
		currOK := false
		for _, c := range e.SupportedCurrencies {
			if c == currency {
				currOK = true
				break
			}
		}
		if !currOK {
			return false
		}
	}

	return true
}

// Registry holds all registered gateways
type Registry struct {
	entries []GatewayEntry
}

// NewRegistry creates an empty gateway registry
func NewRegistry() *Registry {
	return &Registry{}
}

// Register adds a gateway to the registry
func (r *Registry) Register(entry GatewayEntry) *Registry {
	r.entries = append(r.entries, entry)
	return r
}

// Get returns a gateway by name
func (r *Registry) Get(name GatewayName) (port.PaymentGatewayPort, error) {
	for _, e := range r.entries {
		if e.Name == name && e.Enabled {
			return e.Gateway, nil
		}
	}
	return nil, fmt.Errorf("gateway %s not found or disabled", name)
}

// All returns all registered gateway entries (sorted by priority)
func (r *Registry) All() []GatewayEntry {
	return r.entries
}
