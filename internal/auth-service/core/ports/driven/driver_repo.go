package driven

import (
	"context"

	"ride-hail/internal/auth-service/core/domain/models"
)

type IDriverRepo interface {
	Create(ctx context.Context, driver models.Driver) (string, error)
	GetByEmail(ctx context.Context, email string) (models.Driver, error)
}
