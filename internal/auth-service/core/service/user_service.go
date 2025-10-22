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

type AuthService struct {
	ctx      context.Context
	cfg      *config.Config
	authRepo ports.IAuthRepo
	mylog    mylogger.Logger
}

func NewAuthService(
	ctx context.Context,
	cfg *config.Config,
	authRepo ports.IAuthRepo,
	mylogger mylogger.Logger,
) *AuthService {
	return &AuthService{
		ctx:      ctx,
		cfg:      cfg,
		authRepo: authRepo,
		mylog:    mylogger,
	}
}

// ======================= Register =======================
func (as *AuthService) Register(ctx context.Context, regReq dto.UserRegistrationRequest) (string, string, error) {
	mylog := as.mylog.Action("Register")

	if err := validateRegistration(ctx, regReq.Username, regReq.Email, regReq.Password); err != nil {
		return "", "", err
	}

	hashedPassword, err := hashPassword(regReq.Password)
	if err != nil {
		return "", "", fmt.Errorf("failed to hash password: %v", err)
	}
	user := models.User{
		Username:     regReq.Username,
		Email:        regReq.Email,
		PasswordHash: hashedPassword,
		Role:         regReq.Role,
	}
	// add user to db
	id, err := as.authRepo.Create(ctx, user)
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
		"user_id": id,
		"email":   regReq.Email,
		"role":    regReq.Role,
		"exp":     time.Now().Add(time.Hour * 27 * 7).Unix(),
	})
	mylog.Info(as.cfg.App.PublicJwtSecret)
	accessTokenString, err := AccessToken.SignedString([]byte(as.cfg.App.PublicJwtSecret))
	if err != nil {
		mylog.Error("error to create jwt token", err)
		return "", "", err
	}

	mylog.Info("User registered successfully")
	return id, accessTokenString, nil
}

func (as *AuthService) Login(ctx context.Context, authReq dto.UserAuthRequest) (string, error) {
	mylog := as.mylog.Action("Login")

	if err := validateLogin(ctx, authReq.Email, authReq.Password); err != nil {
		return "", err
	}

	user, err := as.authRepo.GetByEmail(ctx, authReq.Email)
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
		"user_id": user.UserId,
		"email":   authReq.Email,
		"role":    user.Role,
		"exp":     time.Now().Add(time.Hour * 27 * 7).Unix(),
	})

	accesssTokenString, err := AccessTokenString.SignedString([]byte(as.cfg.App.PublicJwtSecret))
	if err != nil {
		mylog.Error("error to create jwt token", err)
		return "", err
	}

	mylog.Info("User login successfully")
	return accesssTokenString, nil
}

func (as *AuthService) Logout(ctx context.Context, auth dto.UserAuthRequest) error {
	return nil
}

func (as *AuthService) Protected(ctx context.Context, auth dto.UserAuthRequest) error {
	return nil
}
