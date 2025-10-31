package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"ride-hail/internal/auth-service/adapters/driven/db"
	"ride-hail/internal/auth-service/core/domain/dto"
	"ride-hail/internal/auth-service/core/domain/models"
	"ride-hail/internal/auth-service/core/myerrors"
	"ride-hail/internal/config"
	"ride-hail/internal/mylogger"

	"github.com/golang-jwt/jwt"
)

type DriverService struct {
	ctx        context.Context
	cfg        *config.Config
	driverRepo *db.DriverRepo
	mylog      mylogger.Logger
}

func NewDriverService(
	ctx context.Context,
	cfg *config.Config,
	driverRepo *db.DriverRepo,
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

	r := dto.UserRegistrationRequest{
		Username:  regReq.Username,
		Email:     regReq.Email,
		Role:      "DRIVER",
		Password:  regReq.Password,
		UserAttrs: regReq.UserAttrs,
	}

	if err := validateUserRegistration(ctx, r); err != nil {
		return "", "", err
	}

	if err := validateDriverRegistration(ctx, regReq.LicenseNumber, regReq.VehicleType, regReq.VehicleAttrs); err != nil {
		return "", "", err
	}

	user := models.Driver{
		Username:      regReq.Username,
		Email:         regReq.Email,
		Password:      regReq.Password,
		LicenseNumber: regReq.LicenseNumber,
		VehicleType:   regReq.VehicleType,
		VehicleAttrs:  regReq.VehicleAttrs,
		UserAttrs:     regReq.UserAttrs,
	}
	// add user to db
	id, err := ds.driverRepo.Create(ctx, user)
	if err != nil {
		if errors.Is(err, myerrors.ErrEmailRegistered) {
			mylog.Warn("Failed to register, email already registered")
			return "", "", err
		}
		if errors.Is(err, myerrors.ErrDriverLicenseNumberRegistered) {
			mylog.Warn("Failed to register, driver licence number already registered")
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

func (ds *DriverService) Login(ctx context.Context, authReq dto.DriverAuthRequest) (string, error) {
	mylog := ds.mylog.Action("Login")

	if err := validateLogin(ctx, authReq.Email, authReq.Password); err != nil {
		return "", err
	}

	user, err := ds.driverRepo.GetByEmail(ctx, authReq.Email)
	if err != nil {
		if errors.Is(err, myerrors.ErrUnknownEmail) {
			mylog.Warn("Failed to login, unknown username")
			return "", err
		}
		mylog.Error("Failed to get driver by id", err)
		return "", fmt.Errorf("cannot get user from db: %w", err)
	}

	// Compare password hashes
	if user.Password != authReq.Password {
		mylog.Debug("Failed to login, unknown password")
		return "", myerrors.ErrPasswordUnknown
	}

	AccessTokenString := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":  user.DriverId,
		"username": user.Username,
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
