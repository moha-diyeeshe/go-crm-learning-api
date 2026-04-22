package customers // Implements database operations for customer records.

import (
	"context" // Uses context.Context for cancellation/timeouts in DB queries.
	"errors"  // Uses errors.New and errors.Is for domain and DB error matching.
	"fmt"     // Uses fmt.Errorf to wrap DB errors with operation context.

	"github.com/jackc/pgx/v5"         // Uses pgx.ErrNoRows to detect missing records.
	"github.com/jackc/pgx/v5/pgconn"  // Uses pgconn.PgError to read Postgres error codes.
	"github.com/jackc/pgx/v5/pgxpool" // Uses pgxpool.Pool for shared DB access.
)

var ErrCustomerNotFound = errors.New("customer not found") // Shared not-found error used by service and handler.

type Repository struct { // Holds dependencies needed for customer persistence.
	pool *pgxpool.Pool // Shared database pool injected from main wiring.
}

func NewRepository(pool *pgxpool.Pool) *Repository { // Constructor for repository dependency injection.
	return &Repository{pool: pool} // Returns repository pointer for shared use.
}

func (r *Repository) EnsureSchema(ctx context.Context) error { // Creates customers table if it does not exist.
	query := `CREATE TABLE IF NOT EXISTS customers (
  id BIGSERIAL PRIMARY KEY,
  name TEXT NOT NULL,
  email TEXT NOT NULL UNIQUE,
  phone TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);` // SQL schema statement for customers module.
	_, err := r.pool.Exec(ctx, query) // Executes schema SQL against PostgreSQL.
	if err != nil {                   // Handles schema execution failure.
		return fmt.Errorf("ensure customers table: %w", err) // Wraps raw error with operation context.
	}
	return nil // Signals schema is ready.
}

func (r *Repository) Create(ctx context.Context, in CreateCustomerInput) (Customer, error) { // Inserts one customer and returns saved record.
	query := `INSERT INTO customers (name, email, phone)
VALUES ($1, $2, $3)
RETURNING id, name, email, phone, created_at, updated_at` // SQL insert statement with RETURNING clause.

	var c Customer                                                        // Destination struct that receives returned row.
	err := r.pool.QueryRow(ctx, query, in.Name, in.Email, in.Phone).Scan( // Executes insert and scans returned row.
		&c.ID, &c.Name, &c.Email, &c.Phone, &c.CreatedAt, &c.UpdatedAt, // Maps columns to struct fields.
	)
	if err != nil { // Handles insert/query errors.
		var pgErr *pgconn.PgError                            // Temporary typed Postgres error holder.
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // Detects unique constraint violation for email.
			return Customer{}, fmt.Errorf("email already exists") // Returns user-friendly duplicate email message.
		}
		return Customer{}, fmt.Errorf("create customer: %w", err) // Wraps unknown create failure.
	}
	return c, nil // Returns created customer.
}

func (r *Repository) List(ctx context.Context) ([]Customer, error) { // Returns all customers ordered by ID.
	rows, err := r.pool.Query(ctx, `
SELECT id, name, email, phone, created_at, updated_at
FROM customers ORDER BY id`) // Executes list query and returns cursor.
	if err != nil { // Handles query execution error.
		return nil, fmt.Errorf("list customers: %w", err) // Wraps list operation error.
	}
	defer rows.Close() // Ensures cursor is closed and resources released.

	out := make([]Customer, 0) // Initializes output slice for scanned rows.
	for rows.Next() {          // Iterates through each row in result set.
		var c Customer                                                                                            // Temporary row holder.
		if scanErr := rows.Scan(&c.ID, &c.Name, &c.Email, &c.Phone, &c.CreatedAt, &c.UpdatedAt); scanErr != nil { // Scans current row into struct.
			return nil, fmt.Errorf("scan customer: %w", scanErr) // Returns scan failure with context.
		}
		out = append(out, c) // Appends scanned customer to output slice.
	}
	return out, rows.Err() // Returns data and any deferred cursor error.
}

func (r *Repository) GetByID(ctx context.Context, id int64) (Customer, error) { // Fetches one customer by ID.
	var c Customer // Destination struct for fetched record.
	err := r.pool.QueryRow(ctx, `
SELECT id, name, email, phone, created_at, updated_at
FROM customers WHERE id = $1`, id).Scan( // Executes single-row query filtered by provided ID.
		&c.ID, &c.Name, &c.Email, &c.Phone, &c.CreatedAt, &c.UpdatedAt, // Maps columns to struct fields.
	)
	if err != nil { // Handles missing row or query failure.
		if errors.Is(err, pgx.ErrNoRows) { // Detects when no customer exists for ID.
			return Customer{}, ErrCustomerNotFound // Returns shared not-found domain error.
		}
		return Customer{}, fmt.Errorf("get customer: %w", err) // Wraps unexpected query error.
	}
	return c, nil // Returns found customer.
}

func (r *Repository) Update(ctx context.Context, id int64, in UpdateCustomerInput) (Customer, error) { // Updates customer fields and returns updated row.
	query := `UPDATE customers
SET name = $2, email = $3, phone = $4, updated_at = NOW()
WHERE id = $1
RETURNING id, name, email, phone, created_at, updated_at` // SQL update with RETURNING clause.
	var c Customer                                                            // Destination struct for updated customer.
	err := r.pool.QueryRow(ctx, query, id, in.Name, in.Email, in.Phone).Scan( // Executes update and scans result.
		&c.ID, &c.Name, &c.Email, &c.Phone, &c.CreatedAt, &c.UpdatedAt, // Maps result columns to struct.
	)
	if err != nil { // Handles update failures.
		if errors.Is(err, pgx.ErrNoRows) { // Detects missing customer ID.
			return Customer{}, ErrCustomerNotFound // Returns not-found error.
		}
		var pgErr *pgconn.PgError                            // Holds typed DB error for unique constraint detection.
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // Detects duplicate email on update.
			return Customer{}, fmt.Errorf("email already exists") // Returns user-friendly message.
		}
		return Customer{}, fmt.Errorf("update customer: %w", err) // Wraps unknown update failure.
	}
	return c, nil // Returns updated customer.
}

func (r *Repository) Delete(ctx context.Context, id int64) error { // Deletes one customer by ID.
	tag, err := r.pool.Exec(ctx, `DELETE FROM customers WHERE id = $1`, id) // Executes delete command and captures metadata.
	if err != nil {                                                         // Handles delete query failure.
		return fmt.Errorf("delete customer: %w", err) // Wraps raw DB error.
	}
	if tag.RowsAffected() == 0 { // Checks whether any row was removed.
		return ErrCustomerNotFound // Returns not-found if ID was not present.
	}
	return nil // Signals successful deletion.
}
