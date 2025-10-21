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
	"strings"
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

func (ds *DriverService) ValidateRegistration(ctx context.Context, regReq dto.RegistrationRequest) error {
	ds.mylog.Action("validation_started").Info("Validating registration request")

	if err := validateName(regReq.Username); err != nil {
		return fmt.Errorf("invalid name: %v", err)
	}

	if err := ds.validateEmail(regReq.Email); err != nil {
		return fmt.Errorf("invalid email: %v", err)
	}

	if err := ds.validatePassword(regReq.Password); err != nil {
		return fmt.Errorf("invalid password: %v", err)
	}

	if !AllowedRoles[regReq.Role] {
		return fmt.Errorf("invalid role: %v", regReq.Role)
	}

	ds.mylog.Action("validation_completed").Info("Registration successfully validated")
	return nil
}

func (ds *DriverService) ValidateAuth(ctx context.Context, authReq dto.AuthRequest) error {
	ds.mylog.Action("validation_started").Info("Validating authentification request")

	if err := validateName(authReq.Username); err != nil {
		return fmt.Errorf("invalid username: %v", err)
	}

	if err := ds.validatePassword(authReq.Password); err != nil {
		return fmt.Errorf("invalid password: %v", err)
	}

	ds.mylog.Action("validation_completed").Info("Authentification successfully validated")
	return nil
}

func (ds *DriverService) validateEmail(email string) error {
	if email == "" {
		return ErrFieldIsEmpty
	}

	emailLen := len(email)
	if emailLen < MinEmailLen || emailLen > MaxEmailLen {
		return fmt.Errorf("must be in range [%d, %d]", MinEmailLen, MaxEmailLen)
	}

	if strings.Count(email, "@") != 1 {
		return fmt.Errorf("must contain only one @: %s", email)
	}
	return nil
}

func (ds *DriverService) validatePassword(password string) error {
	if password == "" {
		return ErrFieldIsEmpty
	}

	passwordLen := len(password)
	if passwordLen < MinPasswordLen || passwordLen > MaxPasswordLen {
		return fmt.Errorf("must be in range [%d, %d]", MinPasswordLen, MaxPasswordLen)
	}
	return nil
}

// ======================= Register =======================
// access token, refresh token and error /////////////
func (ds *DriverService) Register(ctx context.Context, regReq dto.RegistrationRequest) (string, error) {
	mylog := ds.mylog.Action("Register")

	hashedPassword, err := hashPassword(regReq.Password)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %v", err)
	}
	user := models.Driver{
		Username:     regReq.Username,
		Email:        regReq.Email,
		PasswordHash: hashedPassword,
		Role:         regReq.Role,
	}
	// add user to db
	id, err := ds.driverRepo.Create(ctx, user)
	if err != nil {
		if errors.Is(err, ErrUsernameTaken) {
			mylog.Warn("Failed to register, username already taken")
			return "", err
		}
		if errors.Is(err, ErrEmailRegistered) {
			mylog.Warn("Failed to register, email already registered")
			return "", err
		}
		mylog.Error("Failed to save user in db", err)
		return "", fmt.Errorf("cannot save user in db: %w", err)
	}

	AccessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":  id,
		"username": regReq.Username,
		"role":     regReq.Role,
		"exp":      time.Now().Add(time.Hour * 24).Unix(),
	})

	accessTokenString, err := AccessToken.SignedString([]byte(ds.cfg.App.PublicJwtSecret))
	if err != nil {
		mylog.Error("error to create jwt token", err)
		return "", err
	}

	mylog.Info("User registered successfully")
	return accessTokenString, nil
}

func (ds *DriverService) Login(ctx context.Context, authReq dto.AuthRequest) (string, error) {
	mylog := ds.mylog.Action("Login")

	user, err := ds.driverRepo.GetByName(ctx, authReq.Username)
	if err != nil {
		if errors.Is(err, ErrUsernameUnknown) {
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
		"user_id":  user.UserId,
		"username": authReq.Username,
		"role":     user.Role,
		"exp":      time.Now().Add(time.Hour * 24).Unix(),
	})

	accesssTokenString, err := AccessTokenString.SignedString([]byte(ds.cfg.App.PublicJwtSecret))
	if err != nil {
		mylog.Error("error to create jwt token", err)
		return "", err
	}

	mylog.Info("User login successfully")
	return accesssTokenString, nil
}

func (ds *DriverService) Logout(ctx context.Context, auth dto.AuthRequest) error {
	return nil
}

func (ds *DriverService) Protected(ctx context.Context, auth dto.AuthRequest) error {
	return nil
}
