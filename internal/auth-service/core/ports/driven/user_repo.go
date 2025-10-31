package driven

import (
	"context"

	"ride-hail/internal/auth-service/core/domain/models"
)

type IUserRepo interface {
	Create(ctx context.Context, user models.User) (string, error)
	GetByEmail(ctx context.Context, name string) (models.User, error)
}
