package ports

import (
	"context"

	"ride-hail/internal/admin-service/core/domain/dto"
)

type IActiveRidesRepo interface {
	GetActiveRides(ctx context.Context, page, pageSize int) (int, []dto.Ride, error)
}
