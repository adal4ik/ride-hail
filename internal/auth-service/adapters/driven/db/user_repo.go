package db

import (
	"context"
	"errors"
	"fmt"
	"ride-hail/internal/auth-service/core/domain/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type AuthRepo struct {
	ctx context.Context
	db  *DB
}

func NewAuthRepo(ctx context.Context, db *DB) *AuthRepo {
	return &AuthRepo{
		ctx: ctx,
		db:  db,
	}
}

func (ar *AuthRepo) Create(ctx context.Context, user models.User) (string, error) {
	// Start a new transaction
	tx, err := ar.db.conn.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to begin transaction: %v", err)
	}

	// Ensure that we roll back in case of any error
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	var userAttrs interface{}
	if user.UserAttrs != nil {
		userAttrs = *user.UserAttrs
	} else {
		userAttrs = nil
	}

	// First query to insert the user
	q := `INSERT INTO users (
	username, email, password_hash, role, user_attrs
	) VALUES ($1, $2, $3, $4, $5) RETURNING user_id;`
	id := ""
	row := tx.QueryRow(ctx, q, user.Username, user.Email, user.PasswordHash, user.Role, userAttrs)
	if err = row.Scan(&id); err != nil {
		// Check if it's a Postgres unique violation
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == "23505" { // unique_violation
				return "", ErrEmailRegistered
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

func (ar *AuthRepo) GetByEmail(ctx context.Context, name string) (models.User, error) {
	q := `
		SELECT 
			u.user_id,
			u.created_at,
			u.updated_at,
			u.username,
			u.email,
			u.status,
			u.password_hash,
			u.role,
			u.user_attrs
		FROM 
			users u
		WHERE
			u.email = $1
	`

	var u models.User
	err := ar.db.conn.QueryRow(ctx, q, name).Scan(
		&u.UserId,
		&u.CreatedAt,
		&u.UpdatedAt,
		&u.Username,
		&u.Email,
		&u.Status,
		&u.PasswordHash,
		&u.Role,
		&u.UserAttrs,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.User{}, ErrUnknownEmail
		}
		return models.User{}, err
	}

	return u, nil
}
