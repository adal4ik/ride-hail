package handle

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"ride-hail/internal/auth-service/core/domain/dto"
	"ride-hail/internal/auth-service/core/service"
	"ride-hail/internal/mylogger"
	"time"
)

type AuthHandler struct {
	authService *service.AuthService
	mylog       mylogger.Logger
}

func NewAuthHandler(authService *service.AuthService, mylog mylogger.Logger) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		mylog:       mylog,
	}
}

var (
	ErrEmailRegistered = errors.New("email already registered")
	ErrUsernameTaken   = errors.New("username already taken")
)

func (ah *AuthHandler) Register() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var regReq dto.UserRegistrationRequest

		mylog := ah.mylog.Action("Register")

		if err := json.NewDecoder(r.Body).Decode(&regReq); err != nil {
			mylog.Error("Failed to parse auth", err)
			jsonError(w, http.StatusBadRequest, errors.New("failed to parse JSON"))
			return
		}
		mylog.Debug("registration data successfully parsed")

		ctx, cancel := context.WithTimeout(context.Background(), WaitTime*time.Second)
		defer cancel()

		userId, accessToken, err := ah.authService.Register(ctx, regReq)
		if err != nil {
			if errors.Is(err, ErrEmailRegistered) || errors.Is(err, ErrUsernameTaken) {
				jsonError(w, http.StatusBadRequest, err)
				return
			}
			jsonError(w, http.StatusInternalServerError, err)
			return
		}

		jsonResponse(w, http.StatusOK, map[string]string{
			"msg":        fmt.Sprintf("%s registered successfully!", regReq.Username),
			"jwt_access": accessToken,
			"userId":     userId,
		})
		mylog.Info("Successfully registered!")
	}
}

func (ah *AuthHandler) Login() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var authReq dto.UserAuthRequest

		mylog := ah.mylog.Action("Register")

		if err := json.NewDecoder(r.Body).Decode(&authReq); err != nil {
			mylog.Error("Failed to parse auth", err)
			jsonError(w, http.StatusBadRequest, errors.New("failed to parse JSON"))
			return
		}
		mylog.Info("registration data successfully parsed")

		ctx, cancel := context.WithTimeout(context.Background(), WaitTime*time.Second)
		defer cancel()

		accessToken, err := ah.authService.Login(ctx, authReq)
		if err != nil {
			jsonError(w, http.StatusInternalServerError, err)
			return
		}

		jsonResponse(w, http.StatusOK, map[string]string{
			"msg":        "Successfully login!",
			"jwt_access": accessToken,
		})
		ah.mylog.Info("Successfully login!")
	}
}

func (ah *AuthHandler) Logout() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
	}
}

func (ah *AuthHandler) Protected() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Its like function to test auth"))
	}
}
