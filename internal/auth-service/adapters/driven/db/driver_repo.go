package db

import (
	"context"
	"errors"
	"fmt"
	"ride-hail/internal/auth-service/core/domain/models"
	"ride-hail/internal/auth-service/core/ports"

	"github.com/jackc/pgx/v5"
)

type DriverRepo struct {
	ctx context.Context
	db  ports.IDB
}

func NewDriverRepo(ctx context.Context, db ports.IDB) *DriverRepo {
	return &DriverRepo{
		ctx: ctx,
		db:  db,
	}
}

func (ar *DriverRepo) Create(ctx context.Context, user models.Driver) (string, error) {
	// Start a new transaction
	tx, err := ar.db.GetConn().Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to begin transaction: %v", err)
	}

	// Ensure that we roll back in case of any error
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	// First query to insert the user
	q := `INSERT INTO drivers (username, email, password_hash, role) VALUES ($1, $2, $3, $4) RETURNING user_id`
	id := ""
	row := tx.QueryRow(ctx, q, user.Username, user.Email, user.PasswordHash, user.Role)
	if err = row.Scan(&id); err != nil {
		return "", fmt.Errorf("failed to insert user: %v", err)
	}

	// Commit the transaction
	if err = tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("failed to commit transaction: %v", err)
	}

	return id, nil
}

func (ar *DriverRepo) GetByName(ctx context.Context, name string) (models.User, error) {
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
			u.attrs,
		FROM 
			users u
		WHERE
			username = $1
	`

	var u models.User
	var refreshToken string
	err := ar.db.GetConn().QueryRow(ctx, q, name).Scan(
		&u.UserId,
		&u.CreatedAt,
		&u.UpdatedAt,
		&u.Username,
		&u.Email,
		&u.Status,
		&u.PasswordHash,
		&u.Role,
		&u.Attrs,
		&refreshToken,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.User{}, ErrUsernameUnknown
		}
		return models.User{}, err
	}

	return u, nil
}
