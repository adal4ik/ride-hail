package driver

import (
	"context"

	"ride-hail/internal/auth-service/core/domain/dto"
)

type IDriverService interface {
	Register(ctx context.Context, regReq dto.DriverRegistrationRequest) (string, string, error)
	Login(ctx context.Context, authReq dto.DriverAuthRequest) (string, error)
}
