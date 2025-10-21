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
	Create(ctx context.Context, user models.User) (string, error)
	// user model, refresh token and error
	GetByName(ctx context.Context, name string) (models.User, error)
}

type IDriverRepo interface {
	// user_id and error
	Create(ctx context.Context, user models.Driver) (string, error)
	// user model, refresh token and error
	GetByName(ctx context.Context, name string) (models.User, error)
}
