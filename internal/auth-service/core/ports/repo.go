package ports

import (
	"context"
	"ride-hail/internal/auth-service/core/domain/models"

	"github.com/jackc/pgx/v5"
)

type IDB interface {
	GetConn() *pgx.Conn
	IsAlive() error
	Close() error
}

type IAuthRepo interface {
	// user_id and error
	Create(ctx context.Context, user models.User, refreshToken string) (string, error)
	// user model, refresh token and error
	GetUserByUsername(ctx context.Context, username string) (models.User, string, error)
}
