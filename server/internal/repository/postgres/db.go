package postgres

import (
	"context"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

func NewDB() (*pgxpool.Pool, error) {
	databaseURL := os.Getenv("POSTGRES_URL")

	dbpool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		return nil, err
	}

	return dbpool, nil
}