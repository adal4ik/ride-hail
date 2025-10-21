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

type DriverHandler struct {
	authService *service.AuthService
	mylog       mylogger.Logger
}

func NewDriverHandler(authService *service.AuthService, mylog mylogger.Logger) *DriverHandler {
	return &DriverHandler{
		authService: authService,
		mylog:       mylog,
	}
}

func (ah *DriverHandler) Register() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var regReq dto.RegistrationRequest

		mylog := ah.mylog.Action("Register")

		if err := json.NewDecoder(r.Body).Decode(&regReq); err != nil {
			mylog.Error("Failed to parse auth", err)
			jsonError(w, http.StatusBadRequest, errors.New("failed to parse JSON"))
			return
		}
		mylog.Debug("registration data successfully parsed")

		ctx, cancel := context.WithTimeout(context.Background(), WaitTime*time.Second)
		defer cancel()

		if err := ah.authService.ValidateRegistration(ctx, regReq); err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		accessToken, err := ah.authService.Register(ctx, regReq)
		if err != nil {
			if errors.Is(err, ErrEmailRegistered) || errors.Is(err, ErrUsernameTaken) {
				jsonError(w, http.StatusBadRequest, err)
				return
			}
			jsonError(w, http.StatusInternalServerError, errors.New("failed to register"))
			return
		}

		jsonResponse(w, http.StatusOK, map[string]string{
			"msg":        fmt.Sprintf("%s registered successfully!", regReq.Username),
			"jwt_access": accessToken,
		})
		mylog.Info("Successfully registered!")
	}
}

func (ah *DriverHandler) Login() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var authReq dto.AuthRequest

		mylog := ah.mylog.Action("Register")

		if err := json.NewDecoder(r.Body).Decode(&authReq); err != nil {
			mylog.Error("Failed to parse auth", err)
			jsonError(w, http.StatusBadRequest, errors.New("failed to parse JSON"))
			return
		}
		mylog.Info("registration data successfully parsed")

		ctx, cancel := context.WithTimeout(context.Background(), WaitTime*time.Second)
		defer cancel()

		if err := ah.authService.ValidateAuth(ctx, authReq); err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		accessToken, err := ah.authService.Login(ctx, authReq)
		if err != nil {
			jsonError(w, http.StatusInternalServerError, errors.New("failed to login"))
			return
		}

		jsonResponse(w, http.StatusOK, map[string]string{
			"msg":        fmt.Sprintf("%s login successfully!", authReq.Username),
			"jwt_access": accessToken,
		})
		ah.mylog.Info("Successfully login!")
	}
}

func (ah *DriverHandler) Logout() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
	}
}

func (ah *DriverHandler) Protected() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Its like function to test auth"))
	}
}
