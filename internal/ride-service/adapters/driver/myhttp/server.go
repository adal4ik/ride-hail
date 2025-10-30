package myhttp

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"ride-hail/internal/config"
	"ride-hail/internal/mylogger"
	"ride-hail/internal/ride-service/adapters/driven/bm"
	"ride-hail/internal/ride-service/adapters/driven/db"
	"ride-hail/internal/ride-service/adapters/driven/notification"
	"ride-hail/internal/ride-service/adapters/driver/myhttp/handle"
	"ride-hail/internal/ride-service/adapters/driver/myhttp/middleware"
	"ride-hail/internal/ride-service/adapters/driver/myhttp/ws"
	websocketdto "ride-hail/internal/ride-service/core/domain/websocket_dto"
	"ride-hail/internal/ride-service/core/ports"
	"ride-hail/internal/ride-service/core/services"
	"sync"
	"time"
)

var ErrServerClosed = errors.New("Server closed")

const WaitTime = 10

type Server struct {
	ctx    context.Context
	appCtx context.Context
	mu     sync.Mutex
	wg     sync.WaitGroup
	mux    *http.ServeMux
	srv    *http.Server

	mylog mylogger.Logger
	cfg   *config.Config

	notify     *notification.Notification
	dispatcher *ws.Dispatcher

	db               *db.DB
	mb               ports.IRidesBroker
	rideService      ports.IRidesService
	passengerService ports.IPassengerService
}

func NewServer(ctx, appCtx context.Context, mylog mylogger.Logger, cfg *config.Config) *Server {
	s := &Server{
		ctx:    ctx,
		appCtx: appCtx,
		cfg:    cfg,
		mylog:  mylog,
		mux:    http.NewServeMux(),
		wg:     sync.WaitGroup{},
	}

	return s
}

// Run initializes routes and starts listening. It returns when the server stops.
func (s *Server) Run() error {
	mylog := s.mylog.Action("server_started").With("port", s.cfg.Srv.RideServicePort)

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

	err = s.notify.Run()
	if err != nil {
		return err
	}

	mylog.Info("server is running")
	return s.startHTTPServer()
}

// Stop provides a programmatic shutdown. Accepts a context for timeout control.
func (s *Server) Stop(ctx context.Context) error {
	log := s.mylog.Action("Stop")
	s.mu.Lock()
	defer s.mu.Unlock()

	log.Info("Shutting down HTTP server...")
	// web socket notification, that server is shutting down
	msg := websocketdto.Event{
		Type: "notify",
		Data: []byte(`{"text":"shutting server lmao XD, now it is your problem XDXDXD"}`),
	}
	log.Info("sending broadcast message...")

	s.dispatcher.BroadCast(msg)
	s.wg.Wait()
	// make rides that have status like
	//   'REQUESTED'
	//   'MATCHED'
	//   'EN_ROUTE'
	//   'ARRIVED'
	//   'IN_PROGRESS'
	//
	// to 'CANCELLED'
	log.Info("cancel everything...")
	err := s.rideService.CancelEveryPossibleRides()
	if err != nil {
		log.Error("cannot cancel", err)
	}
	log.Info("cancelled everything...")

	if s.srv != nil {
		shutdownCtx, cancel := context.WithTimeout(ctx, WaitTime*time.Second)
		defer cancel()

		if err := s.srv.Shutdown(shutdownCtx); err != nil {
			log.Error("Failed to shut down HTTP server gracefully", err)
			return fmt.Errorf("http server shutdown: %w", err)
		}
	}

	if s.db != nil {
		if err := s.db.Close(); err != nil {
			log.Error("Failed to close database", err)
			return fmt.Errorf("db close: %w", err)
		}
		log.Info("Database closed")
	}

	log.Info("HTTP server shut down gracefully")
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
	s.rideService = rideService
	s.passengerService = passengerService

	// handlers
	rideHandler := handle.NewRidesHandler(rideService, s.mylog)

	authMiddleware := middleware.NewAuthMiddleware(s.cfg.App.PublicJwtSecret)

	eventHandle := ws.NewEventHandler(s.cfg.App.PublicJwtSecret)
	dispatcher := ws.NewDispathcer(s.appCtx, s.mylog, passengerService, eventHandle, &s.wg)
	dispatcher.InitHandler()
	s.dispatcher = dispatcher

	// consumers
	notify := notification.New(s.ctx, &s.wg, s.mylog, dispatcher, s.mb, passengerService, rideService)
	s.notify = notify

	// Register routes
	s.mux.Handle("POST /rides", authMiddleware.Wrap(rideHandler.CreateRide()))
	s.mux.Handle("POST /rides/{ride_id}/cancel", authMiddleware.Wrap(rideHandler.CancelRide()))

	// websocket routes
	s.mux.Handle("/ws/passengers/{passenger_id}", dispatcher.WsHandler())
}
