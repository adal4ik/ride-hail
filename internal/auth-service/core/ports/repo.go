package ports

import (
	"context"

	"ride-hail/internal/auth-service/core/domain/models"
)

type IAuthRepo interface {
	Create(ctx context.Context, user models.User) (string, error)
	GetByEmail(ctx context.Context, email string) (models.User, error)
}

type IDriverRepo interface {
	Create(ctx context.Context, user models.Driver) (string, error)
	GetByEmail(ctx context.Context, email string) (models.Driver, error)
}
