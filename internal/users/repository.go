package users

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrUserNotFound = errors.New("user not found")

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) EnsureSchema(ctx context.Context) error {
	query := `
CREATE TABLE IF NOT EXISTS users (
  id BIGSERIAL PRIMARY KEY,
  name TEXT NOT NULL,
  email TEXT NOT NULL UNIQUE,
  phone TEXT NOT NULL,
  password_hash TEXT NOT NULL,
  totp_secret TEXT NOT NULL,
  is_two_factor_enabled BOOLEAN NOT NULL DEFAULT TRUE,
  must_change_password BOOLEAN NOT NULL DEFAULT TRUE,
  last_login_at TIMESTAMPTZ NULL,
  password_changed_at TIMESTAMPTZ NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);`
	_, err := r.pool.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("ensure users table: %w", err)
	}
	return nil
}

func (r *Repository) Create(ctx context.Context, in CreateUserInput, passwordHash, totpSecret string) (User, error) {
	query := `
INSERT INTO users (name, email, phone, password_hash, totp_secret, is_two_factor_enabled, must_change_password)
VALUES ($1, $2, $3, $4, $5, TRUE, TRUE)
RETURNING id, name, email, phone, must_change_password, last_login_at, password_changed_at, is_two_factor_enabled, created_at, updated_at`

	var u User
	err := r.pool.QueryRow(ctx, query, in.Name, in.Email, in.Phone, passwordHash, totpSecret).Scan(
		&u.ID, &u.Name, &u.Email, &u.Phone, &u.MustChangePassword, &u.LastLoginAt, &u.PasswordChangedAt, &u.TwoFactorEnabled, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return User{}, fmt.Errorf("email already exists")
		}
		return User{}, fmt.Errorf("create user: %w", err)
	}
	return u, nil
}

func (r *Repository) List(ctx context.Context) ([]User, error) {
	rows, err := r.pool.Query(ctx, `
SELECT id, name, email, phone, must_change_password, last_login_at, password_changed_at, is_two_factor_enabled, created_at, updated_at
FROM users ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	out := make([]User, 0)
	for rows.Next() {
		var u User
		if scanErr := rows.Scan(&u.ID, &u.Name, &u.Email, &u.Phone, &u.MustChangePassword, &u.LastLoginAt, &u.PasswordChangedAt, &u.TwoFactorEnabled, &u.CreatedAt, &u.UpdatedAt); scanErr != nil {
			return nil, fmt.Errorf("scan user: %w", scanErr)
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func (r *Repository) GetByID(ctx context.Context, id int64) (User, error) {
	var u User
	err := r.pool.QueryRow(ctx, `
SELECT id, name, email, phone, must_change_password, last_login_at, password_changed_at, is_two_factor_enabled, created_at, updated_at
FROM users WHERE id = $1`, id).Scan(
		&u.ID, &u.Name, &u.Email, &u.Phone, &u.MustChangePassword, &u.LastLoginAt, &u.PasswordChangedAt, &u.TwoFactorEnabled, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrUserNotFound
		}
		return User{}, fmt.Errorf("get user: %w", err)
	}
	return u, nil
}

func (r *Repository) Update(ctx context.Context, id int64, in UpdateUserInput) (User, error) {
	query := `
UPDATE users
SET name = $2, email = $3, phone = $4, updated_at = NOW()
WHERE id = $1
RETURNING id, name, email, phone, must_change_password, last_login_at, password_changed_at, is_two_factor_enabled, created_at, updated_at`
	var u User
	err := r.pool.QueryRow(ctx, query, id, in.Name, in.Email, in.Phone).Scan(
		&u.ID, &u.Name, &u.Email, &u.Phone, &u.MustChangePassword, &u.LastLoginAt, &u.PasswordChangedAt, &u.TwoFactorEnabled, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrUserNotFound
		}
		return User{}, fmt.Errorf("update user: %w", err)
	}
	return u, nil
}

func (r *Repository) Delete(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (r *Repository) GetAuthData(ctx context.Context, id int64) (string, string, string, string, error) {
	var passwordHash string
	var totpSecret string
	var email string
	var phone string
	err := r.pool.QueryRow(ctx, `SELECT password_hash, totp_secret, email, phone FROM users WHERE id = $1`, id).
		Scan(&passwordHash, &totpSecret, &email, &phone)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", "", "", "", ErrUserNotFound
		}
		return "", "", "", "", fmt.Errorf("get auth data: %w", err)
	}
	return passwordHash, totpSecret, email, phone, nil
}

func (r *Repository) UpdatePassword(ctx context.Context, id int64, hash string, changedAt time.Time) error {
	tag, err := r.pool.Exec(ctx, `
UPDATE users
SET password_hash = $2, must_change_password = FALSE, password_changed_at = $3, updated_at = NOW()
WHERE id = $1`, id, hash, changedAt)
	if err != nil {
		return fmt.Errorf("update password: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (r *Repository) MarkLogin(ctx context.Context, id int64, loginAt time.Time) error {
	tag, err := r.pool.Exec(ctx, `UPDATE users SET last_login_at = $2, updated_at = NOW() WHERE id = $1`, id, loginAt)
	if err != nil {
		return fmt.Errorf("mark login: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}
