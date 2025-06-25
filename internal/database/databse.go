package database

import (
	"fmt"
	"log"
	"time"

	"github.com/EasyPeek/EasyPeek-backend/internal/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DB is the global database instance
var DB *gorm.DB

// initialize database connection
func Initialize(cfg *config.Config) error {
	dsn := cfg.Database.DSN()

	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info), // logger.Info Warn Error
	}

	// connect to database
	var err error
	DB, err = gorm.Open(postgres.Open(dsn), gormConfig)

	if err != nil {
		log.Fatal("failed to connect database: %w", err)
		return fmt.Errorf("failed to connect database: %w", err)
	}

	// get the underlying sql.DB instance to configure the connection pool
	sqlDB, err := DB.DB()
	if err != nil {
		log.Fatal("failed to get database instance: %w", err)
		return fmt.Errorf("failed to get database instance: %w", err)
	}

	// set connection pool parameters
	sqlDB.SetMaxIdleConns(cfg.Database.MaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.Database.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// test connection
	if err := sqlDB.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	log.Println("database connected successfully")
	return nil
}

// Close close the database connection
func CloseDatabase() error {
	if DB == nil {
		log.Println("database not initialized")
		return fmt.Errorf("database not initialized")
	}

	sqlDB, err := DB.DB()
	if err != nil {
		log.Fatal("failed to get database instance: %w", err)
		return fmt.Errorf("failed to get database instance: %w", err)
	}

	if err := sqlDB.Close(); err != nil {
		log.Fatal("failed to close database: %w", err)
		return fmt.Errorf("failed to close database: %w", err)
	}

	log.Println("database closed successfully")
	return nil
}

// GetDB get the database instance
func GetDB() *gorm.DB {
	return DB
}

// Migrate

// Transaction

// // Migrate execute database migration
// func Migrate(models ...interface{}) error {
// 	if DB == nil {
// 		return fmt.Errorf("database not initialized")
// 	}

// 	for _, model := range models {
// 		if err := DB.AutoMigrate(model); err != nil {
// 			return fmt.Errorf("failed to migrate model %T: %w", model, err)
// 		}
// 	}

// 	log.Printf("database migration completed, %d models migrated", len(models))
// 	return nil
// }

// // Transaction execute transaction
// func Transaction(fn func(*gorm.DB) error) error {
// 	if DB == nil {
// 		return fmt.Errorf("database not initialized")
// 	}

// 	return DB.Transaction(fn)
// }
