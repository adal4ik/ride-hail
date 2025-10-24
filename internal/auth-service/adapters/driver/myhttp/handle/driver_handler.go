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
	driverService *service.DriverService
	mylog         mylogger.Logger
}

func NewDriverHandler(driverService *service.DriverService, mylog mylogger.Logger) *DriverHandler {
	return &DriverHandler{
		driverService: driverService,
		mylog:         mylog,
	}
}

func (ah *DriverHandler) Register() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var regReq dto.DriverRegistrationRequest

		mylog := ah.mylog.Action("Register")

		if err := json.NewDecoder(r.Body).Decode(&regReq); err != nil {
			mylog.Error("Failed to parse auth", err)
			jsonError(w, http.StatusBadRequest, errors.New("failed to parse JSON"))
			return
		}
		mylog.Debug("registration data successfully parsed")

		ctx, cancel := context.WithTimeout(context.Background(), WaitTime*time.Second)
		defer cancel()

		userId, accessToken, err := ah.driverService.Register(ctx, regReq)
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

func (ah *DriverHandler) Login() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var driverReq dto.DriverRegistrationRequest

		mylog := ah.mylog.Action("Register")

		if err := json.NewDecoder(r.Body).Decode(&driverReq); err != nil {
			mylog.Error("Failed to parse auth", err)
			jsonError(w, http.StatusBadRequest, errors.New("failed to parse JSON"))
			return
		}
		mylog.Info("registration data successfully parsed")

		ctx, cancel := context.WithTimeout(context.Background(), WaitTime*time.Second)
		defer cancel()

		accessToken, err := ah.driverService.Login(ctx, driverReq)
		if err != nil {
			jsonError(w, http.StatusInternalServerError, err)
			return
		}

		jsonResponse(w, http.StatusOK, map[string]string{
			"msg":        fmt.Sprintf("%s login successfully!", driverReq.Username),
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
