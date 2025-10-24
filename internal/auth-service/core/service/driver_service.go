package service

import (
	"context"
	"errors"
	"fmt"
	"ride-hail/internal/auth-service/core/domain/dto"
	"ride-hail/internal/auth-service/core/domain/models"
	"ride-hail/internal/auth-service/core/ports"
	"ride-hail/internal/config"
	"ride-hail/internal/mylogger"
	"time"

	"github.com/golang-jwt/jwt"
)

type DriverService struct {
	ctx        context.Context
	cfg        *config.Config
	driverRepo ports.IDriverRepo
	mylog      mylogger.Logger
}

func NewDriverService(
	ctx context.Context,
	cfg *config.Config,
	driverRepo ports.IDriverRepo,
	mylogger mylogger.Logger,
) *DriverService {
	return &DriverService{
		ctx:        ctx,
		cfg:        cfg,
		driverRepo: driverRepo,
		mylog:      mylogger,
	}
}

// ======================= Register =======================
func (ds *DriverService) Register(ctx context.Context, regReq dto.DriverRegistrationRequest) (string, string, error) {
	mylog := ds.mylog.Action("Register")

	if err := validateRegistration(ctx, regReq.Username, regReq.Email, regReq.Password); err != nil {
		return "", "", err
	}

	if err := validateDriverRegistration(ctx, regReq.LicenseNumber, regReq.VehicleType, *regReq.VehicleAttrs); err != nil {
		return "", "", err
	}

	hashedPassword, err := hashPassword(regReq.Password)
	if err != nil {
		return "", "", fmt.Errorf("failed to hash password: %v", err)
	}
	user := models.Driver{
		Username:      regReq.Username,
		Email:         regReq.Email,
		PasswordHash:  hashedPassword,
		LicenseNumber: regReq.LicenseNumber,
		VehicleType:   regReq.VehicleType,
		VehicleAttrs:  regReq.VehicleAttrs,
	}
	// add user to db
	id, err := ds.driverRepo.Create(ctx, user)
	if err != nil {
		if errors.Is(err, ErrUsernameTaken) {
			mylog.Warn("Failed to register, username already taken")
			return "", "", err
		}
		if errors.Is(err, ErrEmailRegistered) {
			mylog.Warn("Failed to register, email already registered")
			return "", "", err
		}
		mylog.Error("Failed to save user in db", err)
		return "", "", fmt.Errorf("cannot save user in db: %w", err)
	}

	AccessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":  id,
		"username": regReq.Username,
		"role":     "DRIVER",
		"exp":      time.Now().Add(time.Hour * 27 * 7).Unix(),
	})

	accessTokenString, err := AccessToken.SignedString([]byte(ds.cfg.App.PublicJwtSecret))
	if err != nil {
		mylog.Error("error to create jwt token", err)
		return "", "", err
	}

	mylog.Info("User registered successfully")
	return id, accessTokenString, nil
}

func (ds *DriverService) Login(ctx context.Context, authReq dto.DriverRegistrationRequest) (string, error) {
	mylog := ds.mylog.Action("Login")

	if err := validateLogin(ctx, authReq.Email, authReq.Password); err != nil {
		return "", err
	}

	user, err := ds.driverRepo.GetByEmail(ctx, authReq.Username)
	if err != nil {
		if errors.Is(err, ErrUnknownEmail) {
			mylog.Warn("Failed to login, unknown username")
			return "", err
		}
		mylog.Error("Failed to save user in db", err)
		return "", fmt.Errorf("cannot save user in db: %w", err)
	}

	// Compare password hashes
	if !checkPassword(user.PasswordHash, authReq.Password) {
		mylog.Debug("Failed to login, unknown password")
		return "", ErrPasswordUnknown
	}

	AccessTokenString := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":  user.DriverId,
		"username": authReq.Username,
		"role":     "DRIVER",
		"exp":      time.Now().Add(time.Hour * 27 * 7).Unix(),
	})

	accesssTokenString, err := AccessTokenString.SignedString([]byte(ds.cfg.App.PublicJwtSecret))
	if err != nil {
		mylog.Error("error to create jwt token", err)
		return "", err
	}

	mylog.Info("User login successfully")
	return accesssTokenString, nil
}

func (ds *DriverService) Logout(ctx context.Context, auth dto.DriverAuthRequest) error {
	return nil
}

func (ds *DriverService) Protected(ctx context.Context, auth dto.DriverAuthRequest) error {
	return nil
}
