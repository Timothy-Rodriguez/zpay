package model

import (
	"zpay/internal/database"
	"zpay/internal/pkg"
)

type App struct {
	Logger *pkg.Logger
	DB     *database.DB
	Redis  *pkg.Redis
}
