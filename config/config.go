package config

import (
	"fmt"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	Stripe   StripeConfig
	VNPay    VNPayConfig
	Kafka    KafkaConfig
	Email    EmailConfig
}

type ServerConfig struct {
	Port   string
	APIKey string
}

type DatabaseConfig struct {
	DSN string
}

type RedisConfig struct {
	Addr     string
	Password string
}

type StripeConfig struct {
	SecretKey string
}

type VNPayConfig struct {
	TmnCode    string
	HashSecret string
	ReturnURL  string
}

type KafkaConfig struct {
	Brokers []string
	Topic   string
}

type EmailConfig struct {
	Host     string
	Port     string
	Username string
	Password string
	From     string
}

func Load() (*Config, error) {
	viper.SetConfigFile(".env")
	viper.SetConfigType("env")
	viper.AutomaticEnv()
	_ = viper.ReadInConfig()

	viper.SetDefault("PORT", "8080")
	viper.SetDefault("REDIS_ADDR", "localhost:6379")
	viper.SetDefault("EMAIL_PORT", "587")
	viper.SetDefault("KAFKA_TOPIC", "payment-events")

	cfg := &Config{
		Server: ServerConfig{
			Port:   viper.GetString("PORT"),
			APIKey: viper.GetString("API_KEY"),
		},
		Database: DatabaseConfig{DSN: viper.GetString("DATABASE_DSN")},
		Redis:    RedisConfig{Addr: viper.GetString("REDIS_ADDR"), Password: viper.GetString("REDIS_PASSWORD")},
		Stripe:   StripeConfig{SecretKey: viper.GetString("STRIPE_SECRET_KEY")},
		VNPay: VNPayConfig{
			TmnCode:    viper.GetString("VNPAY_TMN_CODE"),
			HashSecret: viper.GetString("VNPAY_HASH_SECRET"),
			ReturnURL:  viper.GetString("VNPAY_RETURN_URL"),
		},
		Kafka: KafkaConfig{
			Brokers: viper.GetStringSlice("KAFKA_BROKERS"),
			Topic:   viper.GetString("KAFKA_TOPIC"),
		},
		Email: EmailConfig{
			Host:     viper.GetString("EMAIL_HOST"),
			Port:     viper.GetString("EMAIL_PORT"),
			Username: viper.GetString("EMAIL_USERNAME"),
			Password: viper.GetString("EMAIL_PASSWORD"),
			From:     viper.GetString("EMAIL_FROM"),
		},
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) validate() error {
	if c.Database.DSN == "" {
		return fmt.Errorf("DATABASE_DSN is required")
	}
	if c.Stripe.SecretKey == "" {
		return fmt.Errorf("STRIPE_SECRET_KEY is required")
	}
	return nil
}
