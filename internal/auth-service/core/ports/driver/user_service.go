package driver

import (
	"context"

	"ride-hail/internal/auth-service/core/domain/dto"
)

type IAuthService interface {
	Register(ctx context.Context, regReq dto.UserRegistrationRequest) (string, string, error)
	Login(ctx context.Context, authReq dto.UserAuthRequest) (string, error)
}
