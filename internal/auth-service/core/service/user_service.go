package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"ride-hail/internal/auth-service/core/domain/dto"
	"ride-hail/internal/auth-service/core/domain/models"
	"ride-hail/internal/auth-service/core/myerrors"
	"ride-hail/internal/auth-service/core/ports/driven"
	"ride-hail/internal/config"
	"ride-hail/internal/mylogger"

	"github.com/golang-jwt/jwt"
)

type UserService struct {
	ctx      context.Context
	cfg      *config.Config
	authRepo driven.IUserRepo
	mylog    mylogger.Logger
}

func NewUserService(
	ctx context.Context,
	cfg *config.Config,
	authRepo driven.IUserRepo,
	mylogger mylogger.Logger,
) *UserService {
	return &UserService{
		ctx:      ctx,
		cfg:      cfg,
		authRepo: authRepo,
		mylog:    mylogger,
	}
}

// ======================= Register =======================
func (us *UserService) Register(ctx context.Context, regReq dto.UserRegistrationRequest) (string, string, error) {
	mylog := us.mylog.Action("Register")

	if err := validateUserRegistration(ctx, regReq); err != nil {
		return "", "", err
	}

	user := models.User{
		Username:  regReq.Username,
		Email:     regReq.Email,
		Password:  regReq.Password,
		Role:      regReq.Role,
		UserAttrs: regReq.UserAttrs,
	}
	// add user to db
	id, err := us.authRepo.Create(ctx, user)
	if err != nil {
		if errors.Is(err, myerrors.ErrDBConnClosed) {
			mylog.Error("Failed to connect to connect to db", err)
			return "", "", myerrors.ErrDBConnClosedMsg
		}
		if errors.Is(err, myerrors.ErrEmailRegistered) {
			mylog.Error("Failed to register, email already registered", err)
			return "", "", err
		}
		mylog.Error("Failed to save user in db", err)
		return "", "", fmt.Errorf("cannot save user in db: %w", err)
	}

	AccessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": id,
		"email":   regReq.Email,
		"role":    regReq.Role,
		"exp":     time.Now().Add(time.Hour * 27 * 7).Unix(),
	})
	mylog.Info(us.cfg.App.PublicJwtSecret)
	accessTokenString, err := AccessToken.SignedString([]byte(us.cfg.App.PublicJwtSecret))
	if err != nil {
		mylog.Error("error to create jwt token", err)
		return "", "", err
	}

	mylog.Info("User registered successfully")
	return id, accessTokenString, nil
}

func (us *UserService) Login(ctx context.Context, authReq dto.UserAuthRequest) (string, error) {
	mylog := us.mylog.Action("Login")

	if err := validateLogin(ctx, authReq.Email, authReq.Password); err != nil {
		return "", err
	}

	user, err := us.authRepo.GetByEmail(ctx, authReq.Email)
	if err != nil {
		if errors.Is(err, myerrors.ErrDBConnClosed) {
			mylog.Error("Failed to connect to connect to db", err)
			return "", myerrors.ErrDBConnClosedMsg
		}
		if errors.Is(err, myerrors.ErrUnknownEmail) {
			mylog.Error("Failed to login, unknown email", err)
			return "", err
		}

		mylog.Error("Failed to save user in db", err)
		return "", fmt.Errorf("cannot save user in db: %w", err)
	}

	// Compare password hashes
	if user.Password != authReq.Password {
		mylog.Debug("Failed to login, unknown password")
		return "", myerrors.ErrPasswordUnknown
	}

	AccessTokenString := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.UserId,
		"email":   authReq.Email,
		"role":    user.Role,
		"exp":     time.Now().Add(time.Hour * 27 * 7).Unix(),
	})

	accessTokenString, err := AccessTokenString.SignedString([]byte(us.cfg.App.PublicJwtSecret))
	if err != nil {
		mylog.Error("error to create jwt token", err)
		return "", err
	}

	mylog.Info("User login successfully")
	return accessTokenString, nil
}
