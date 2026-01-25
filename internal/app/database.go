package app

import (
	"context"
	"fmt"

	"github.com/avc/loyalty-system-diploma/internal/repository/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// initDatabase создает пул соединений с базой данных
func initDatabase(ctx context.Context, databaseURI string) (*pgxpool.Pool, error) {
	dbPool, err := pgxpool.New(ctx, databaseURI)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := dbPool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return dbPool, nil
}

// runMigrations выполняет миграции базы данных
func runMigrations(ctx context.Context, dbPool *pgxpool.Pool, logger *zap.Logger) error {
	if err := postgres.RunMigrations(ctx, dbPool, logger); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}
	return nil
}
