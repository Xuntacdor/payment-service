package messaging

import (
	"encoding/json"
	"fmt"

	"github.com/Xuntacdor/payment-service/internal/domain/port"
	"go.uber.org/zap"
)

type DomainEvent struct {
	EventName string      `json:"event_name"`
	Payload   interface{} `json:"payload"`
}

type MockKafkaPublisher struct {
	logger *zap.Logger
}

func NewMockKafkaPublisher(logger *zap.Logger) port.EventPublisherPort {
	return &MockKafkaPublisher{logger: logger}
}

func (p *MockKafkaPublisher) Publish(eventName string, payload interface{}) error {
	data, err := json.Marshal(DomainEvent{EventName: eventName, Payload: payload})
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}
	p.logger.Info("[MOCK EVENT PUBLISHED]",
		zap.String("topic", "payment-events"),
		zap.String("event", eventName),
		zap.String("payload", string(data)),
	)
	return nil
}
