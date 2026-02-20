package store

import (
	"context"
	"database/sql"
	"time"

	_ "github.com/lib/pq"
)

type User struct {
	ID        string
	CasdoorID string
	Email     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type UserStore struct {
	db *sql.DB
}

func NewUserStore(databaseURL string) (*UserStore, error) {
	// Connect to the PostgreSQL database using the provided database URL
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	return &UserStore{db: db}, nil
}

// Upsert inserts a new user or updates the existing user's email based on the Casdoor ID.
func (s *UserStore) Upsert(ctx context.Context, casdoorID, email string) (*User, error) {
	query := `
		INSERT INTO users (casdoor_id, email)
		VALUES ($1, $2)
		ON CONFLICT (casdoor_id) DO UPDATE
			SET email = EXCLUDED.email,
			    updated_at = NOW()
		RETURNING id, casdoor_id, email, created_at, updated_at
	`

	// Execute the upsert query and scan the result into a User struct
	user := &User{}
	err := s.db.QueryRowContext(ctx, query, casdoorID, email).Scan(
		&user.ID,
		&user.CasdoorID,
		&user.Email,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return user, nil
}
