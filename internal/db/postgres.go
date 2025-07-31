package db

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
	"gorm.io/driver/postgres"
	"github.com/bohemiyan/RBAC/internal/config"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// PostgresDB wraps both sql.DB and gorm.DB
type PostgresDB struct {
	DB     *sql.DB
	GormDB *gorm.DB
}

func NewPostgresDB(cfg *config.Config) (*PostgresDB, error) {

	// Existing sql.DB connection
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		cfg.PostgresHost, cfg.PostgresPort, cfg.PostgresUser, cfg.PostgresPassword, cfg.PostgresDB)
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping PostgreSQL: %w", err)
	}

	// Initialize GORM
	gormDB, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Error),
	})
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize GORM: %w", err)
	}


	pg := &PostgresDB{
		DB:     db,
		GormDB: gormDB,
	}

	return pg, nil
}

func (p *PostgresDB) Close() error {
	if err := p.DB.Close(); err != nil {
		return fmt.Errorf("failed to close sql.DB: %w", err)
	}

	sqlDB, err := p.GormDB.DB()
	if err != nil {
		return fmt.Errorf("failed to get sql.DB from GORM: %w", err)
	}
	if err := sqlDB.Close(); err != nil {
		return fmt.Errorf("failed to close GORM sql.DB: %w", err)
	}

	return nil
}
