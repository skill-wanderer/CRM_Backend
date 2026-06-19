package database

import (
	"fmt"
	"log"

	"crm-backend/internal/config"
	"crm-backend/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Connect(cfg config.DatabaseConfig) (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=%s",
		cfg.Host,
		cfg.User,
		cfg.Password,
		cfg.Name,
		cfg.Port,
		cfg.SSLMode,
		cfg.TimeZone,
	)

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("connect to database: %w", err)
	}

	sqlDB, err := DB.DB()
	if err != nil {
		return nil, fmt.Errorf("open database pool: %w", err)
	}
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	log.Println("Database connection established")

	if cfg.AutoMigrate {
		if err := migrate(DB); err != nil {
			return nil, err
		}
		log.Println("Database migration completed")
	}

	return DB, nil
}

func migrate(db *gorm.DB) error {
	if err := db.Exec("CREATE EXTENSION IF NOT EXISTS pgcrypto").Error; err != nil {
		return fmt.Errorf("ensure pgcrypto extension: %w", err)
	}

	if err := db.SetupJoinTable(&models.User{}, "Tenants", &models.UserTenant{}); err != nil {
		return fmt.Errorf("setup user tenant join table: %w", err)
	}

	if err := db.AutoMigrate(
		&models.Tenant{},
		&models.User{},
		&models.UserTenant{},
		&models.LeadTemplate{},
		&models.LeadField{},
		&models.Lead{},
		&models.LeadValue{},
	); err != nil {
		return fmt.Errorf("auto migrate models: %w", err)
	}

	return nil
}
