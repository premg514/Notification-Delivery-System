package postgres

import (
	"context"
	"errors"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

func NewDB() (*pgxpool.Pool, error) {
	databaseURL := os.Getenv("POSTGRES_URL")
	if databaseURL == "" {
		return nil, errors.New("POSTGRES_URL is not set")
	}

	dbpool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		return nil, err
	}

	if err := dbpool.Ping(context.Background()); err != nil {
		dbpool.Close()
		return nil, err
	}

	return dbpool, nil
}
