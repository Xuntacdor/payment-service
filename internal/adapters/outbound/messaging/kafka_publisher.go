package messaging

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Xuntacdor/payment-service/internal/domain/port"
	kafka "github.com/segmentio/kafka-go"
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

// ---- Real Kafka Publisher ----

type RealKafkaPublisher struct {
	writer *kafka.Writer
	logger *zap.Logger
}

func NewRealKafkaPublisher(brokers []string, topic string, logger *zap.Logger) port.EventPublisherPort {
	return &RealKafkaPublisher{
		writer: &kafka.Writer{
			Addr:                   kafka.TCP(brokers...),
			Topic:                  topic,
			AllowAutoTopicCreation: true,
			Balancer:               &kafka.LeastBytes{},
		},
		logger: logger,
	}
}

func (p *RealKafkaPublisher) Publish(eventName string, payload interface{}) error {
	data, err := json.Marshal(DomainEvent{EventName: eventName, Payload: payload})
	if err != nil {
		return fmt.Errorf("marshal failed: %w", err)
	}
	err = p.writer.WriteMessages(context.Background(),
		kafka.Message{
			Key:   []byte(eventName),
			Value: data,
		},
	)
	if err != nil {
		p.logger.Error("kafka publish failed", zap.Error(err))
		return fmt.Errorf("kafka publish failed: %w", err)
	}
	p.logger.Info("event published", zap.String("event", eventName))
	return nil
}
