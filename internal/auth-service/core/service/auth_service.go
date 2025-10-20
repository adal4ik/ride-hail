package service

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"ride-hail/internal/auth-service/core/domain/dto"
	"ride-hail/internal/auth-service/core/domain/models"
	"ride-hail/internal/auth-service/core/ports"
	"ride-hail/internal/config"
	"ride-hail/internal/mylogger"
	"strings"
	"time"

	"github.com/golang-jwt/jwt"
	"golang.org/x/crypto/bcrypt"
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

const (
	MinCustomerNameLen = 1
	MaxCustomerNameLen = 100

	MinEmailLen = 5
	MaxEmailLen = 100

	MinPasswordLen = 5
	MaxPasswordLen = 50

	HashFactor = 10

	TokenLen = 32
)

var AllowedRoles = map[string]bool{
	"PASSENGER": true,
	"ADMIN":     true,
	"DRIVER":    true,
}

var (
	ErrFieldIsEmpty    = errors.New("field is empty")
	ErrUsernameUnknown = errors.New("unknown username")
	ErrPasswordUnknown = errors.New("unknown password")
	ErrUsernameTaken   = errors.New("username already taken")
	ErrEmailRegistered = errors.New("email already registered")
)

func (as *AuthService) ValidateRegistration(ctx context.Context, regReq dto.RegistrationRequest) error {
	as.mylog.Action("validation_started").Info("Validating registration request")

	if err := as.validateUsername(regReq.Username); err != nil {
		return fmt.Errorf("invalid username: %v", err)
	}

	if err := as.validateEmail(regReq.Email); err != nil {
		return fmt.Errorf("invalid email: %v", err)
	}

	if err := as.validatePassword(regReq.Password); err != nil {
		return fmt.Errorf("invalid password: %v", err)
	}

	if !AllowedRoles[regReq.Role] {
		return fmt.Errorf("invalid role: %v", regReq.Role)
	}

	as.mylog.Action("validation_completed").Info("Registration successfully validated")
	return nil
}

func (as *AuthService) ValidateAuth(ctx context.Context, authReq dto.AuthRequest) error {
	as.mylog.Action("validation_started").Info("Validating authentification request")

	if err := as.validateUsername(authReq.Username); err != nil {
		return fmt.Errorf("invalid username: %v", err)
	}

	if err := as.validatePassword(authReq.Password); err != nil {
		return fmt.Errorf("invalid password: %v", err)
	}

	as.mylog.Action("validation_completed").Info("Authentification successfully validated")
	return nil
}

func (as *AuthService) validateUsername(username string) error {
	if username == "" {
		return ErrFieldIsEmpty
	}

	usernameLen := len(username)
	if usernameLen < MinCustomerNameLen || usernameLen > MaxCustomerNameLen {
		return fmt.Errorf("must be in range [%d, %d]", MinCustomerNameLen, MaxCustomerNameLen)
	}

	return nil
}

func (as *AuthService) validateEmail(email string) error {
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

func (as *AuthService) validatePassword(password string) error {
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
func (as *AuthService) Register(ctx context.Context, regReq dto.RegistrationRequest) (string, string, error) {
	mylog := as.mylog.Action("Register")

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
	// generate refresh token
	refreshToken := randSeq(255)
	// add user to db
	id, err := as.authRepo.Create(ctx, user, refreshToken)
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
		"role":     regReq.Role,
		"exp":      time.Now().Add(time.Hour * 24).Unix(),
	})
	RefreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":  id,
		"username": regReq.Username,
		"role":     regReq.Role,
		"token":    refreshToken,
		"exp":      time.Now().Add(time.Hour * 24 * 7).Unix(),
	})

	accessTokenString, err := AccessToken.SignedString([]byte(as.cfg.App.PublicJwtSecret))
	if err != nil {
		mylog.Error("error to create jwt token", err)
		return "", "", err
	}

	refreshTokenString, err := RefreshToken.SignedString([]byte(as.cfg.App.PrivateJwtSecret))
	if err != nil {
		mylog.Error("error to create jwt token", err)
		return "", "", err
	}

	mylog.Info("User registered successfully")
	return accessTokenString, refreshTokenString, nil
}

func (as *AuthService) Login(ctx context.Context, authReq dto.AuthRequest) (string, string, error) {
	mylog := as.mylog.Action("Login")

	user, refreshToken, err := as.authRepo.GetUserByUsername(ctx, authReq.Username)
	if err != nil {
		if errors.Is(err, ErrUsernameUnknown) {
			mylog.Warn("Failed to login, unknown username")
			return "", "", err
		}
		mylog.Error("Failed to save user in db", err)
		return "", "", fmt.Errorf("cannot save user in db: %w", err)
	}

	// Compare password hashes
	if !checkPassword(user.PasswordHash, authReq.Password) {
		mylog.Debug("Failed to login, unknown password")
		return "", "", ErrPasswordUnknown
	}

	AccessTokenString := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":  user.UserId,
		"username": authReq.Username,
		"role":     user.Role,
		"exp":      time.Now().Add(time.Hour * 24).Unix(),
	})

	RefreshTokenString := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":  user.UserId,
		"username": authReq.Username,
		"role":     user.Role,
		"token":    refreshToken,
		"exp":      time.Now().Add(time.Hour * 24 * 7).Unix(),
	})

	accesssTokenString, err := AccessTokenString.SignedString([]byte(as.cfg.App.PublicJwtSecret))
	if err != nil {
		mylog.Error("error to create jwt token", err)
		return "", "", err
	}

	refreshTokenString, err := RefreshTokenString.SignedString([]byte(as.cfg.App.PrivateJwtSecret))
	if err != nil {
		mylog.Error("error to create jwt token", err)
		return "", "", err
	}

	mylog.Info("User login successfully")
	return accesssTokenString, refreshTokenString, nil
}

func (as *AuthService) Logout(ctx context.Context, auth dto.AuthRequest) error {
	return nil
}

func (as *AuthService) Protected(ctx context.Context, auth dto.AuthRequest) error {
	return nil
}

func hashPassword(password string) ([]byte, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), HashFactor)
	return bytes, err
}

func checkPassword(hashed []byte, password string) bool {
	return bcrypt.CompareHashAndPassword(hashed, []byte(password)) == nil
}

func randSeq(n int) string {
	// Define the character set (lowercase, uppercase)
	charSet := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

	// Seed the random number generator for true randomness
	rand.New(rand.NewSource(time.Now().UnixNano()))

	// Pre-allocate the slice for the correlation ID
	b := make([]rune, n)

	// Create the random part of the ID
	for i := range b {
		b[i] = charSet[rand.Intn(len(charSet))]
	}
	return string(b)
}
