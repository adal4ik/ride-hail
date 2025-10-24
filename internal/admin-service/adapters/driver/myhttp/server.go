package myhttp

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"ride-hail/internal/admin-service/adapters/driven/db"
	"ride-hail/internal/admin-service/adapters/driver/myhttp/handle"
	"ride-hail/internal/admin-service/adapters/driver/myhttp/middleware"
	"ride-hail/internal/admin-service/core/ports"
	"ride-hail/internal/admin-service/core/service"
	"ride-hail/internal/config"
	"ride-hail/internal/mylogger"
	"sync"
	"time"
)

var ErrServerClosed = errors.New("Server closed")

const WaitTime = 10

type Server struct {
	mux    *http.ServeMux
	cfg    *config.Config
	srv    *http.Server
	mylog  mylogger.Logger
	db     ports.IDB
	ctx    context.Context
	appCtx context.Context
	mu     sync.Mutex
	wg     sync.WaitGroup
}

func NewServer(ctx, appCtx context.Context, mylog mylogger.Logger, cfg *config.Config) *Server {
	return &Server{
		ctx:    ctx,
		appCtx: appCtx,
		cfg:    cfg,
		mylog:  mylog,
		mux:    http.NewServeMux(),
	}
}

// Run initializes routes and starts listening. It returns when the server stops.
func (s *Server) Run() error {
	mylog := s.mylog.Action("server_started")
	// Initialize database connection
	if err := s.initializeDatabase(); err != nil {
		mylog.Action("db_connection_failed").Error("Failed to connect to database", err)
		return err
	}
	mylog.Action("db_connected").Info("Successful database connection")

	// Configure routes and handlers
	s.Configure()

	s.mu.Lock()
	s.srv = &http.Server{
		Addr:    fmt.Sprintf(":%v", s.cfg.Srv.AdminServicePort),
		Handler: s.mux,
	}
	s.mu.Unlock()

	mylog = mylog.WithGroup("details").With("port", s.cfg.Srv.AdminServicePort)

	mylog.Info("server is running")
	// Start the HTTP server and handle graceful shutdown
	return s.startHTTPServer()
}

// Stop provides a programmatic shutdown. Accepts a context for timeout control.
func (s *Server) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.mylog.Action("graceful_shutdown_started").Info("Shutting down HTTP server...")

	s.wg.Wait()

	if s.srv != nil {
		shutdownCtx, cancel := context.WithTimeout(ctx, WaitTime*time.Second)
		defer cancel()

		if err := s.srv.Shutdown(shutdownCtx); err != nil {
			s.mylog.Action("graceful_shutdown_failed").Error("Failed to shut down HTTP server gracefully", err)
			return fmt.Errorf("http server shutdown: %w", err)
		}
	}

	if s.db != nil {
		if err := s.db.Close(); err != nil {
			s.mylog.Action("db_close_failed").Error("Failed to close database", err)
			return fmt.Errorf("db close: %w", err)
		}
		s.mylog.Action("db_closed").Info("Database closed")
	}

	s.mylog.Action("graceful_shutdown_completed").Info("HTTP server shut down gracefully")

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
	// Repositories and services
	systemOverviewRepo := db.NewSystemOverviewRepo(s.db)
	activeRidesRepo := db.NewActiveDrivesRepo(s.db)

	systemOverviewService := service.NewSystemOverviewService(s.ctx, s.mylog, systemOverviewRepo)
	activeRidesService := service.NewActiveDrivesService(s.ctx, s.mylog, activeRidesRepo)

	systemOverviewHandler := handle.NewSystemOverviewHandler(s.mylog, systemOverviewService)
	activeRidesHandler := handle.NewActiveDrivesHandler(s.mylog, activeRidesService)

	authMiddleware := middleware.NewAuthMiddleware(s.cfg.App.PublicJwtSecret)

	// Register routes
	s.mux.Handle("GET /admin/overview", authMiddleware.Wrap(systemOverviewHandler.GetSystemOverview()))
	s.mux.Handle("GET /admin/rides/active", authMiddleware.Wrap(activeRidesHandler.GetActiveRides()))
}

func (s *Server) initializeDatabase() error {
	db, err := db.Start(s.ctx, s.cfg.DB, s.mylog)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	s.db = db
	return nil
}
