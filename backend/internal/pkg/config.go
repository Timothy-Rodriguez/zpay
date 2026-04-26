package pkg

import (
	"fmt"
	"log"
	"time"

	"github.com/spf13/viper"
)

type ServerConfig struct {
	Port int
	Env  string
}

type JWTConfig struct {
	Expiration int
}

type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
	MaxConns int
}

type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
}

type KafkaConfig struct {
	Brokers  []string `mapstructure:"brokers"`
	ClientID string   `mapstructure:"client_id"`
	Producer struct {
		Topic   string        `mapstructure:"topic"`
		Timeout time.Duration `mapstructure:"timeout"`
	} `mapstructure:"producer"`
	Consumer struct {
		Topic   string `mapstructure:"topic"`
		GroupID string `mapstructure:"group_id"`
	} `mapstructure:"consumer"`
}

type SMTPConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	From     string `mapstructure:"from"`
}

// Config
type Config struct {
	JWT      JWTConfig
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	Kafka    KafkaConfig
	SMTP     SMTPConfig
}

func LoadConfig(configPath string) (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(configPath)

	if err := viper.ReadInConfig(); err != nil {
		log.Printf("Warning: config file not found, using defaults: %v", err)
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &config, nil
}
