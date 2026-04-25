package model

import (
	"zpay/internal/database"
	"zpay/internal/pkg"
)

type App struct {
	Logger *pkg.Logger
	JWT    pkg.JWTService
	DB     database.DatabaseClient
	Redis  *pkg.Redis
}
