package db

import (
	"context"
	"errors"
	"fmt"

	"ride-hail/internal/auth-service/core/domain/models"
	"ride-hail/internal/auth-service/core/myerrors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type UserRepo struct {
	ctx context.Context
	db  *DB
}

func NewUserRepo(ctx context.Context, db *DB) *UserRepo {
	return &UserRepo{
		ctx: ctx,
		db:  db,
	}
}

func (ur *UserRepo) Create(ctx context.Context, user models.User) (string, error) {
	// Start a new transaction
	tx, err := ur.db.conn.Begin(ctx)
	if err != nil {
		// Check if the database is alive
		if err := ur.db.IsAlive(); err != nil {
			return "", err
		}
		return "", fmt.Errorf("failed to begin transaction: %v", err)
	}

	// Ensure that we roll back in case of any error
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	// Check if the database is alive
	if err := ur.db.IsAlive(); err != nil {
		return "", err
	}

	var userAttrs interface{}
	if user.UserAttrs != nil {
		userAttrs = *user.UserAttrs
	} else {
		userAttrs = nil
	}

	// First query to insert the user
	q := `INSERT INTO users (
	username, email, password, role, user_attrs
	) VALUES ($1, $2, $3, $4, $5) RETURNING user_id;`
	id := ""
	row := tx.QueryRow(ctx, q, user.Username, user.Email, user.Password, user.Role, userAttrs)
	if err = row.Scan(&id); err != nil {
		// Check if it's a Postgres unique violation
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == "23505" { // unique_violation
				return "", myerrors.ErrEmailRegistered
			}
		}
		return "", fmt.Errorf("failed to insert user: %v", err)
	}
	// Commit the transaction
	if err = tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("failed to commit transaction: %v", err)
	}

	return id, nil
}

func (ur *UserRepo) GetByEmail(ctx context.Context, name string) (models.User, error) {
	// Check if the database is alive
	if err := ur.db.IsAlive(); err != nil {
		return models.User{}, err
	}

	q := `
		SELECT 
			u.user_id,
			u.created_at,
			u.updated_at,
			u.username,
			u.email,
			u.status,
			u.password,
			u.role,
			u.user_attrs
		FROM 
			users u
		WHERE
			u.email = $1
	`

	var u models.User
	err := ur.db.conn.QueryRow(ctx, q, name).Scan(
		&u.UserId,
		&u.CreatedAt,
		&u.UpdatedAt,
		&u.Username,
		&u.Email,
		&u.Status,
		&u.Password,
		&u.Role,
		&u.UserAttrs,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.User{}, myerrors.ErrUnknownEmail
		}
		return models.User{}, err
	}

	return u, nil
}
