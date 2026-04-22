package users // Implements database access logic for the users module.

import (
	"context" // Uses context.Context to propagate cancellation/timeouts to DB calls.
	"errors"  // Uses errors.New, errors.Is, and errors.As for error creation/matching.
	"fmt"     // Uses fmt.Errorf to wrap low-level DB errors with repository context.
	"time"    // Uses time.Time type for password/login timestamp updates.

	"github.com/jackc/pgx/v5"         // Uses pgx.ErrNoRows sentinel for "record missing" checks.
	"github.com/jackc/pgx/v5/pgconn"  // Uses pgconn.PgError to inspect Postgres error codes.
	"github.com/jackc/pgx/v5/pgxpool" // Uses pgxpool.Pool as shared Postgres connection pool.
)

var ErrUserNotFound = errors.New("user not found") // Shared domain error used by service/handler for 404 responses.

type Repository struct { // Holds infrastructure dependency required by repository methods.
	pool *pgxpool.Pool // Shared DB pool injected from main.go during startup wiring.
}

func NewRepository(pool *pgxpool.Pool) *Repository { // Constructor that builds repository with injected pool dependency.
	return &Repository{pool: pool} // Returns pointer so methods use same repository instance across requests.
}

func (r *Repository) EnsureSchema(ctx context.Context) error { // Creates table if missing so app can run on fresh DB.
	query := `CREATE TABLE IF NOT EXISTS users (
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
);` // SQL schema string executed once during startup.
	_, err := r.pool.Exec(ctx, query) // Executes schema SQL using connection pool with request/startup context.
	if err != nil {                   // Checks whether SQL execution failed.
		return fmt.Errorf("ensure users table: %w", err) // Wraps error with operation name for easier debugging.
	}
	return nil // Signals schema setup completed successfully.
}

func (r *Repository) Create(ctx context.Context, in CreateUserInput, passwordHash, totpSecret string) (User, error) { // Inserts a new user and returns complete created row.
	query := `INSERT INTO users (name, email, phone, password_hash, totp_secret, is_two_factor_enabled, must_change_password)
VALUES ($1, $2, $3, $4, $5, TRUE, TRUE)
RETURNING id, name, email, phone, must_change_password, last_login_at, password_changed_at, is_two_factor_enabled, created_at, updated_at` // SQL insert with RETURNING so API can respond with saved data.

	var u User                                                                                      // Allocates output variable that Scan fills from RETURNING values.
	err := r.pool.QueryRow(ctx, query, in.Name, in.Email, in.Phone, passwordHash, totpSecret).Scan( // Runs query and scans single returned row.
		&u.ID, &u.Name, &u.Email, &u.Phone, &u.MustChangePassword, &u.LastLoginAt, &u.PasswordChangedAt, &u.TwoFactorEnabled, &u.CreatedAt, &u.UpdatedAt, // Field-by-field column mapping to struct.
	)
	if err != nil { // Handles DB or scan failure.
		var pgErr *pgconn.PgError                            // Holds typed Postgres error when available.
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // Checks unique violation code for duplicate email.
			return User{}, fmt.Errorf("email already exists") // Returns friendly validation-style message to upper layers.
		}
		return User{}, fmt.Errorf("create user: %w", err) // Wraps unknown insert error with operation label.
	}
	return u, nil // Returns created user model to service/handler.
}

func (r *Repository) List(ctx context.Context) ([]User, error) { // Reads all users ordered by ID for listing endpoint.
	rows, err := r.pool.Query(ctx, `
SELECT id, name, email, phone, must_change_password, last_login_at, password_changed_at, is_two_factor_enabled, created_at, updated_at
FROM users ORDER BY id`) // Executes SELECT returning multiple rows with deterministic ordering.
	if err != nil { // Handles query execution failure.
		return nil, fmt.Errorf("list users: %w", err) // Wraps error with operation context.
	}
	defer rows.Close() // Ensures DB cursor/resources are released after iteration.

	out := make([]User, 0) // Creates empty slice to collect scanned users.
	for rows.Next() {      // Iterates through each returned row.
		var u User                                                                                                                                                                                  // Temporary struct per row before appending.
		if scanErr := rows.Scan(&u.ID, &u.Name, &u.Email, &u.Phone, &u.MustChangePassword, &u.LastLoginAt, &u.PasswordChangedAt, &u.TwoFactorEnabled, &u.CreatedAt, &u.UpdatedAt); scanErr != nil { // Maps current row into struct fields.
			return nil, fmt.Errorf("scan user: %w", scanErr) // Stops and reports scan/data conversion error.
		}
		out = append(out, u) // Adds scanned user to output slice.
	}
	return out, rows.Err() // Returns collected users and any deferred iterator error.
}

func (r *Repository) GetByID(ctx context.Context, id int64) (User, error) { // Reads one user by primary key ID.
	var u User // Destination struct for scanned row.
	err := r.pool.QueryRow(ctx, `
SELECT id, name, email, phone, must_change_password, last_login_at, password_changed_at, is_two_factor_enabled, created_at, updated_at
FROM users WHERE id = $1`, id).Scan( // Executes single-row SELECT filtered by provided ID.
		&u.ID, &u.Name, &u.Email, &u.Phone, &u.MustChangePassword, &u.LastLoginAt, &u.PasswordChangedAt, &u.TwoFactorEnabled, &u.CreatedAt, &u.UpdatedAt, // Maps columns to struct.
	)
	if err != nil { // Handles missing row or other query/scan errors.
		if errors.Is(err, pgx.ErrNoRows) { // Detects when ID does not exist.
			return User{}, ErrUserNotFound // Returns shared domain error for service/handler mapping.
		}
		return User{}, fmt.Errorf("get user: %w", err) // Wraps non-not-found errors.
	}
	return u, nil // Returns found user.
}

func (r *Repository) Update(ctx context.Context, id int64, in UpdateUserInput) (User, error) { // Updates editable profile fields and returns updated row.
	query := `UPDATE users
SET name = $2, email = $3, phone = $4, updated_at = NOW()
WHERE id = $1
RETURNING id, name, email, phone, must_change_password, last_login_at, password_changed_at, is_two_factor_enabled, created_at, updated_at` // SQL update statement with RETURNING for fresh values.
	var u User                                                                // Destination struct for returned updated row.
	err := r.pool.QueryRow(ctx, query, id, in.Name, in.Email, in.Phone).Scan( // Executes update and scans returned row.
		&u.ID, &u.Name, &u.Email, &u.Phone, &u.MustChangePassword, &u.LastLoginAt, &u.PasswordChangedAt, &u.TwoFactorEnabled, &u.CreatedAt, &u.UpdatedAt, // Column-to-struct mapping.
	)
	if err != nil { // Handles missing row or SQL errors.
		if errors.Is(err, pgx.ErrNoRows) { // No row returned means ID was not found.
			return User{}, ErrUserNotFound // Propagates domain not-found error.
		}
		return User{}, fmt.Errorf("update user: %w", err) // Wraps unexpected update error.
	}
	return u, nil // Returns updated user.
}

func (r *Repository) Delete(ctx context.Context, id int64) error { // Deletes user row by ID.
	tag, err := r.pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, id) // Executes delete and gets command tag metadata.
	if err != nil {                                                     // Checks SQL execution failure.
		return fmt.Errorf("delete user: %w", err) // Wraps delete failure details.
	}
	if tag.RowsAffected() == 0 { // Verifies a row was actually removed.
		return ErrUserNotFound // Returns not-found when no row matched the ID.
	}
	return nil // Signals successful deletion.
}

func (r *Repository) GetAuthData(ctx context.Context, id int64) (string, string, string, string, error) { // Reads auth-related fields used by login/password/2FA service methods.
	var passwordHash string                                                                                      // Holds stored bcrypt hash for password verification.
	var totpSecret string                                                                                        // Holds stored TOTP secret for code generation/validation.
	var email string                                                                                             // Holds email mainly for logging TOTP demo output.
	var phone string                                                                                             // Holds phone mainly for logging TOTP demo output.
	err := r.pool.QueryRow(ctx, `SELECT password_hash, totp_secret, email, phone FROM users WHERE id = $1`, id). // Executes lookup for one user auth record.
															Scan(&passwordHash, &totpSecret, &email, &phone) // Scans selected columns into local variables.
	if err != nil { // Handles missing row or other DB errors.
		if errors.Is(err, pgx.ErrNoRows) { // Maps empty result to domain-level not-found.
			return "", "", "", "", ErrUserNotFound // Returns zero values plus not-found error.
		}
		return "", "", "", "", fmt.Errorf("get auth data: %w", err) // Wraps unexpected error with context.
	}
	return passwordHash, totpSecret, email, phone, nil // Returns all required auth data to service layer.
}

func (r *Repository) GetAuthDataByEmail(ctx context.Context, email string) (int64, string, error) { // Reads user ID and password hash by email for login endpoint.
	var id int64                                                                                                        // Holds user ID mapped from database row.
	var passwordHash string                                                                                             // Holds stored bcrypt hash for password verification.
	err := r.pool.QueryRow(ctx, `SELECT id, password_hash FROM users WHERE email = $1`, email).Scan(&id, &passwordHash) // Queries login credentials by unique email.
	if err != nil {                                                                                                     // Handles not-found and query errors.
		if errors.Is(err, pgx.ErrNoRows) { // Maps missing email to domain not-found error.
			return 0, "", ErrUserNotFound // Returns not-found for unknown email address.
		}
		return 0, "", fmt.Errorf("get auth data by email: %w", err) // Wraps unexpected query errors.
	}
	return id, passwordHash, nil // Returns user ID and password hash for login verification.
}

func (r *Repository) UpdatePassword(ctx context.Context, id int64, hash string, changedAt time.Time) error { // Persists new password hash and related security flags/timestamps.
	tag, err := r.pool.Exec(ctx, `
UPDATE users
SET password_hash = $2, must_change_password = FALSE, password_changed_at = $3, updated_at = NOW()
WHERE id = $1`, id, hash, changedAt) // Executes password maintenance update statement.
	if err != nil { // Handles SQL execution failure.
		return fmt.Errorf("update password: %w", err) // Wraps low-level error with repository context.
	}
	if tag.RowsAffected() == 0 { // Detects missing target row.
		return ErrUserNotFound // Returns standard not-found error when ID absent.
	}
	return nil // Signals password update success.
}

func (r *Repository) MarkLogin(ctx context.Context, id int64, loginAt time.Time) error { // Updates last successful login timestamp for audit/security.
	tag, err := r.pool.Exec(ctx, `UPDATE users SET last_login_at = $2, updated_at = NOW() WHERE id = $1`, id, loginAt) // Executes timestamp update for matching user.
	if err != nil {                                                                                                    // Handles DB failure while updating login marker.
		return fmt.Errorf("mark login: %w", err) // Wraps error with operation label.
	}
	if tag.RowsAffected() == 0 { // Verifies user ID existed.
		return ErrUserNotFound // Returns not-found when no row matched.
	}
	return nil // Signals login timestamp update success.
}
