package postgres

import (
	"context"
	"embed"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// RunMigrations выполняет миграции базы данных
// Автоматически находит все *.up.sql файлы и выполняет их в алфавитном порядке
func RunMigrations(ctx context.Context, pool *pgxpool.Pool, logger *zap.Logger) error {
	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	// Собираем только up миграции и сортируем
	var upMigrations []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".up.sql") {
			upMigrations = append(upMigrations, entry.Name())
		}
	}
	sort.Strings(upMigrations)

	// Выполняем миграции по порядку
	for _, name := range upMigrations {
		migrationPath := filepath.Join("migrations", name)
		content, err := migrationsFS.ReadFile(migrationPath)
		if err != nil {
			return fmt.Errorf("failed to read migration %s: %w", name, err)
		}

		logger.Info("running migration", zap.String("name", name))
		_, err = pool.Exec(ctx, string(content))
		if err != nil {
			return fmt.Errorf("failed to run migration %s: %w", name, err)
		}
		logger.Info("migration completed", zap.String("name", name))
	}

	return nil
}
