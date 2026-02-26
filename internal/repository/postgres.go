package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"Go-Microservice-Template/internal/model"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Common errors for repository operations.
var (
	ErrNotFound     = errors.New("record not found")
	ErrDuplicate    = errors.New("record already exists")
	ErrInvalidInput = errors.New("invalid input")
)

// UserRepository defines the interface for user data access.
type UserRepository interface {
	Create(ctx context.Context, user *model.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.User, error)
	GetByEmail(ctx context.Context, email string) (*model.User, error)
	Update(ctx context.Context, user *model.User) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, params model.ListParams) ([]model.User, int64, error)
}

// postgresUserRepo implements UserRepository using PostgreSQL.
type postgresUserRepo struct {
	pool *pgxpool.Pool
}

// NewPostgresPool creates a connection pool with production-ready settings.
func NewPostgresPool(ctx context.Context, connStr string) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return nil, fmt.Errorf("parse connection string: %w", err)
	}

	// Production-ready pool settings
	config.MaxConns = 25
	config.MinConns = 5
	config.MaxConnLifetime = 30 * time.Minute
	config.MaxConnIdleTime = 5 * time.Minute
	config.HealthCheckPeriod = 30 * time.Second

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}

	// Verify connectivity
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return pool, nil
}

// NewUserRepository creates a new PostgreSQL-backed user repository.
func NewUserRepository(pool *pgxpool.Pool) UserRepository {
	return &postgresUserRepo{pool: pool}
}
func (r *postgresUserRepo) Create(ctx context.Context, user *model.User) error {
	user.ID = uuid.New()
	user.CreatedAt = time.Now().UTC()
	user.UpdatedAt = user.CreatedAt

	query := `
		INSERT INTO users (id, email, name, password_hash, role, active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := r.pool.Exec(ctx, query,
		user.ID, user.Email, user.Name, user.Password,
		user.Role, user.Active, user.CreatedAt, user.UpdatedAt,
	)
	if err != nil {
		// Check for unique constraint violation
		if isDuplicateError(err) {
			return ErrDuplicate
		}
		return fmt.Errorf("insert user: %w", err)
	}

	return nil
}

func (r *postgresUserRepo) Delete(ctx context.Context, id uuid.UUID) error {
	// Soft delete — set active to false
	query := `UPDATE users SET active = false, updated_at = $2 WHERE id = $1 AND active = true`

	result, err := r.pool.Exec(ctx, query, id, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

func (r *postgresUserRepo) Update(ctx context.Context, user *model.User) error {
	user.UpdatedAt = time.Now().UTC()

	query := `
		UPDATE users
		SET email = $2, name = $3, role = $4, active = $5, updated_at = $6
		WHERE id = $1
	`

	result, err := r.pool.Exec(ctx, query,
		user.ID, user.Email, user.Name, user.Role, user.Active, user.UpdatedAt,
	)
	if err != nil {
		if isDuplicateError(err) {
			return ErrDuplicate
		}
		return fmt.Errorf("update user: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

// isDuplicateError checks if the error is a PostgreSQL unique violation (code 23505).
func isDuplicateError(err error) bool {
	return err != nil && contains(err.Error(), "23505")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func (r *postgresUserRepo) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	query := `
		SELECT id, email, name, password_hash, role, active, created_at, updated_at
		FROM users
		WHERE email = $1 AND active = true
	`

	var user model.User
	err := r.pool.QueryRow(ctx, query, email).Scan(
		&user.ID, &user.Email, &user.Name, &user.Password,
		&user.Role, &user.Active, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get user by email: %w", err)
	}

	return &user, nil
}

func (r *postgresUserRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	query := `
		SELECT id, email, name, password_hash, role, active, created_at, updated_at
		FROM users
		WHERE id = $1 AND active = true
	`

	var user model.User
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&user.ID, &user.Email, &user.Name, &user.Password,
		&user.Role, &user.Active, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get user by id: %w", err)
	}

	return &user, nil
}

func (r *postgresUserRepo) List(ctx context.Context, params model.ListParams) ([]model.User, int64, error) {
	// Count total matching records
	countQuery := `SELECT COUNT(*) FROM users WHERE active = true`
	args := []interface{}{}
	argIndex := 1

	if params.Search != "" {
		countQuery += fmt.Sprintf(` AND (name ILIKE $%d OR email ILIKE $%d)`, argIndex, argIndex)
		args = append(args, "%"+params.Search+"%")
		argIndex++
	}

	var total int64
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count users: %w", err)
	}

	// Fetch page
	query := `SELECT id, email, name, password_hash, role, active, created_at, updated_at FROM users WHERE active = true`

	fetchArgs := []interface{}{}
	fetchIndex := 1

	if params.Search != "" {
		query += fmt.Sprintf(` AND (name ILIKE $%d OR email ILIKE $%d)`, fetchIndex, fetchIndex)
		fetchArgs = append(fetchArgs, "%"+params.Search+"%")
		fetchIndex++
	}

	// Validate sort column to prevent SQL injection
	sortCol := "created_at"
	switch params.SortBy {
	case "name", "email", "created_at", "updated_at":
		sortCol = params.SortBy
	}

	sortDir := "DESC"
	if params.SortDir == "asc" {
		sortDir = "ASC"
	}

	query += fmt.Sprintf(` ORDER BY %s %s`, sortCol, sortDir)
	query += fmt.Sprintf(` LIMIT $%d OFFSET $%d`, fetchIndex, fetchIndex+1)
	fetchArgs = append(fetchArgs, params.PageSize, (params.Page-1)*params.PageSize)

	rows, err := r.pool.Query(ctx, query, fetchArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	var users []model.User
	for rows.Next() {
		var u model.User
		if err := rows.Scan(&u.ID, &u.Email, &u.Name, &u.Password, &u.Role, &u.Active, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan user: %w", err)
		}
		users = append(users, u)
	}

	return users, total, nil
}
