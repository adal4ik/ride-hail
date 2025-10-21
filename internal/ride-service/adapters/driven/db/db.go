package db

import (
	"context"
	"fmt"
	"sync"

	"ride-hail/internal/config"
	"ride-hail/internal/mylogger"

	"github.com/jackc/pgx/v5"
)

type DB struct {
	ctx          context.Context
	cfg          *config.DBconfig
	mylog        mylogger.Logger
	conn         *pgx.Conn
	reconnecting bool
	mu           *sync.Mutex
}

// Start initializes and returns a new DB instance with a single connection
func New(ctx context.Context, dbCfg *config.DBconfig, mylog mylogger.Logger) (*DB, error) {
	d := &DB{
		cfg:          dbCfg,
		ctx:          ctx,
		mylog:        mylog,
		mu:           &sync.Mutex{},
		reconnecting: false,
	}

	if err := d.connect(); err != nil {
		return nil, err
	}

	return d, nil
}

// Close closes the connection
func (d *DB) Close() error {
	if err := d.conn.Close(d.ctx); err != nil {
		return fmt.Errorf("close database connection: %v", err)
	}
	return nil
}

// IsAlive pings the DB to verify it's responsive
func (d *DB) IsAlive() error {
	if d.conn == nil {
		return fmt.Errorf("DB is not initialized")
	}
	if err := d.conn.Ping(d.ctx); err != nil {
		if connectionErr := d.connect(); connectionErr != nil {
			return fmt.Errorf("ping failed: %w", err)
		}
	}

	return nil
}

func (d *DB) connect() error {
	// Establish connection
	conn, err := pgx.Connect(d.ctx, fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=disable",

		d.cfg.User,
		d.cfg.Password,
		d.cfg.Host,
		d.cfg.Port,
		d.cfg.Database,
	))
	if err != nil {
		return fmt.Errorf("failed to connect to database: %v", err)
	}
	d.conn = conn
	return nil
}
