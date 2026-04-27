package model

import (
	"zpay/internal/database"
	"zpay/internal/pkg"

	"go.opentelemetry.io/otel/trace"
)

type App struct {
	Logger *pkg.StructuredLogger
	JWT    pkg.JWTService
	DB     database.DatabaseClient
	Redis  *pkg.Redis
	Kafka  pkg.KafkaClient
	Tracer trace.Tracer
}
