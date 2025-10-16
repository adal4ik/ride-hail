package db

import (
	"context"

	"ride-hail/internal/admin-service/core/domain/dto"
	"ride-hail/internal/admin-service/core/ports"
)

type ActiveDrivesRepo struct {
	db ports.IDB
}

func NewActiveDrivesRepo(db ports.IDB) *ActiveDrivesRepo {
	return &ActiveDrivesRepo{db: db}
}

func (ar *ActiveDrivesRepo) GetActiveRides(ctx context.Context) ([]dto.ActiveDrives, error) {
	return nil, nil
}
