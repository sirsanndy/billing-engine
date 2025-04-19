package config

import (
	"log"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Logger   LoggerConfig   `mapstructure:"logger"`
	Metrics  MetricsConfig  `mapstructure:"metrics"`
	RabbitMQ RabbitMQConfig `mapstructure:"rabbitmq"`
}

type ServerConfig struct {
	Port         int             `mapstructure:"port"`
	ReadTimeout  time.Duration   `mapstructure:"readTimeout"`
	WriteTimeout time.Duration   `mapstructure:"writeTimeout"`
	IdleTimeout  time.Duration   `mapstructure:"idleTimeout"`
	RateLimit    RateLimitConfig `mapstructure:"rateLimit"`
	Auth         AuthConfig      `mapstructure:"auth"`
}

type RateLimitConfig struct {
	Enabled bool    `mapstructure:"enabled"`
	RPS     float64 `mapstructure:"rps"`
	Burst   int     `mapstructure:"burst"`
}

type AuthConfig struct {
	Enabled   bool   `mapstructure:"enabled"`
	JWTSecret string `mapstructure:"jwtSecret"`
}

type DatabaseConfig struct {
	URL string `mapstructure:"url"`
}

type LoggerConfig struct {
	Level    string `mapstructure:"level"`
	Encoding string `mapstructure:"encoding"`
}

type MetricsConfig struct {
	Port int    `mapstructure:"port"`
	Path string `mapstructure:"path"`
}

type RabbitMQConfig struct {
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	Username     string `mapstructure:"username"`
	Password     string `mapstructure:"password"`
	QueueName    string `mapstructure:"queueName"`
	ExchangeName string `mapstructure:"exchangeName"`
	ConsumerTag  string `mapstructure:"consumerTag"`
}

func LoadConfig(path string) (*Config, error) {
	viper.AddConfigPath(path)
	viper.SetConfigName("config")
	viper.SetConfigType("yml")

	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.readTimeout", 15*time.Second)
	viper.SetDefault("server.writeTimeout", 15*time.Second)
	viper.SetDefault("server.idleTimeout", 60*time.Second)
	viper.SetDefault("server.rateLimit.enabled", true)
	viper.SetDefault("server.rateLimit.rps", 10)
	viper.SetDefault("server.rateLimit.burst", 20)
	viper.SetDefault("server.auth.enabled", true)
	viper.SetDefault("database.url", "postgres://user:password@localhost:5432/payment_db?sslmode=disable")
	viper.SetDefault("logger.level", "info")
	viper.SetDefault("logger.encoding", "json")
	viper.SetDefault("metrics.port", 9090)
	viper.SetDefault("metrics.path", "/metrics")
	viper.SetDefault("server.auth.JWTSecret", "")
	viper.SetDefault("rabbitmq.host", "localhost")
	viper.SetDefault("rabbitmq.port", 5672)
	viper.SetDefault("rabbitmq.username", "guest")
	viper.SetDefault("rabbitmq.password", "guest")
	viper.SetDefault("rabbitmq.queueName", "payment-service")
	viper.SetDefault("rabbitmq.exchangeName", "billing-engine")
	viper.SetDefault("rabbitmq.consumerTag", "payment-service-consumer")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Println("Config file not found, using defaults and environment variables.")
		} else {
			return nil, err
		}
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
