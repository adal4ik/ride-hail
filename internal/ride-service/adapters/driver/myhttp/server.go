package myhttp

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"ride-hail/internal/config"
	"ride-hail/internal/mylogger"
	"ride-hail/internal/ride-service/adapters/driven/bm"
	"ride-hail/internal/ride-service/adapters/driven/db"
	"ride-hail/internal/ride-service/adapters/driver/myhttp/handle"
	"ride-hail/internal/ride-service/adapters/driver/myhttp/middleware"
	"ride-hail/internal/ride-service/adapters/driver/myhttp/ws"
	"ride-hail/internal/ride-service/core/ports"
	"ride-hail/internal/ride-service/core/services"
)

var ErrServerClosed = errors.New("Server closed")

const WaitTime = 10

type Server struct {
	mux    *http.ServeMux
	cfg    *config.Config
	srv    *http.Server
	mylog  mylogger.Logger
	db     *db.DB
	mb     ports.IRidesBroker
	ctx    context.Context
	appCtx context.Context
	mu     sync.Mutex
	wg     sync.WaitGroup
}

func NewServer(ctx, appCtx context.Context, mylog mylogger.Logger, cfg *config.Config) *Server {
	s := &Server{
		ctx:    ctx,
		appCtx: appCtx,
		cfg:    cfg,
		mylog:  mylog,
		mux:    http.NewServeMux(),
	}

	return s
}

// Run initializes routes and starts listening. It returns when the server stops.
func (s *Server) Run() error {
	mylog := s.mylog.Action("server_started")

	// Initialize database connection
	db, err := db.New(s.ctx, s.cfg.DB, mylog)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	s.db = db
	mylog.Info("Successful database connection")

	// Initialize RabbitMQ connection
	mb, err := bm.New(s.appCtx, *s.cfg.RabbitMq, s.mylog)
	if err != nil {
		return fmt.Errorf("failed to connect to rabbitmq: %w", err)
	}
	s.mb = mb
	mylog.Info("Successful message broker connection")

	// Configure routes and handlers
	s.Configure()

	s.mu.Lock()
	s.srv = &http.Server{
		Addr:    fmt.Sprintf(":%v", s.cfg.Srv.RideServicePort),
		Handler: s.mux,
	}
	s.mu.Unlock()

	mylog = mylog.WithGroup("details").With("port", s.cfg.Srv.RideServicePort)

	mylog.Info("server is running")
	return s.startHTTPServer()
}

// Stop provides a programmatic shutdown. Accepts a context for timeout control.
func (s *Server) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.mylog.Info("Shutting down HTTP server...")

	s.wg.Wait()

	if s.srv != nil {
		shutdownCtx, cancel := context.WithTimeout(ctx, WaitTime*time.Second)
		defer cancel()

		if err := s.srv.Shutdown(shutdownCtx); err != nil {
			s.mylog.Error("Failed to shut down HTTP server gracefully", err)
			return fmt.Errorf("http server shutdown: %w", err)
		}
	}

	if s.db != nil {
		if err := s.db.Close(); err != nil {
			s.mylog.Error("Failed to close database", err)
			return fmt.Errorf("db close: %w", err)
		}
		s.mylog.Info("Database closed")
	}

	s.mylog.Info("HTTP server shut down gracefully")
	return nil
}

func (s *Server) startHTTPServer() error {
	errCh := make(chan error, 1)

	go func() {
		if err := s.srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		} else {
			errCh <- nil
		}
	}()

	select {
	case <-s.ctx.Done():
		return nil
	case err := <-errCh:
		return err
	}
}

// Configure sets up the HTTP handlers for various APIs including Market Data, Data Mode control, and Health checks.
func (s *Server) Configure() {
	// Repositories
	rideRepo := db.NewRidesRepo(s.db)
	passengerRepo := db.NewPassengerRepo(s.db)

	// services
	rideService := services.NewRidesService(s.appCtx, s.mylog, rideRepo, s.mb, nil)
	passengerService := services.NewPassengerService(s.appCtx, s.mylog, passengerRepo, nil)

	// handlers
	rideHandler := handle.NewRidesHandler(rideService, s.mylog)
	eventHander := ws.NewEventHandler(s.cfg.App.PublicJwtSecret)

	authMiddleware := middleware.NewAuthMiddleware(s.cfg.App.PublicJwtSecret)
	// Register routes

	dispatcher := ws.NewDispathcer(s.mylog, passengerService, *eventHander)
	dispatcher.InitHandler()

	// TODO: add middleware
	s.mux.Handle("POST /rides", authMiddleware.Wrap(rideHandler.CreateRide()))
	// s.mux.Handle("GET /rides/{ride_id}/cancel", nil)

	// websocket routes
	s.mux.Handle("/ws/passengers/{passenger_id}", dispatcher.WsHandler())
}
