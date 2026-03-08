package database

import (
	"fmt"
	"log"
	"time"

	"backend/internal/config"
	"backend/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func InitDB(cfg *config.PostgresConfig) (*gorm.DB, error) {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=%s",
		cfg.Host, cfg.User, cfg.Password, cfg.DBName, cfg.Port, cfg.SSLMode)

	var db *gorm.DB
	var err error
	for i := 0; i < 10; i++ {
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err == nil {
			break
		}
		log.Printf("Failed to connect to database, retrying in 2s... (%d/10)", i+1)
		time.Sleep(2 * time.Second)
	}

	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	sqlDB.SetMaxOpenConns(cfg.MaxConnections)

	DB = db
	log.Println("Database connection established")

	// Enable pgvector extension
	err = db.Exec("CREATE EXTENSION IF NOT EXISTS vector").Error
	if err != nil {
		log.Printf("Failed to create pgvector extension: %v", err)
		return nil, err
	}

	// Auto Migration
	err = db.AutoMigrate(
		&models.User{},
		&models.Conversation{},
		&models.Message{},
		&models.Document{},
		&models.DocumentChunk{},
		&models.ProviderSetting{},
		// Branch and editing models
		&models.Branch{},
		&models.MessageVersion{},
		&models.ParallelExploration{},
		// Agent System models
		&models.CustomAgent{},
		&models.AgentExecution{},
		&models.Workflow{},
		&models.WorkflowStep{},
		&models.WorkflowEdge{},
		&models.WorkflowRun{},
		&models.WorkflowStepRun{},
		&models.ToolPermission{},
		&models.ToolInvocationLog{},
		&models.MarketplaceItem{},
		&models.MarketplaceReview{},
		&models.MarketplaceStar{},
		&models.MarketplaceDownload{},
		&models.ABTest{},
		&models.ABTestRun{},
	)
	if err != nil {
		log.Printf("Failed to run auto migration: %v", err)
		return nil, err
	}
	log.Println("Database migration completed")

	// Run message-to-branch migration (idempotent)
	if err := MigrateMessagesToBranches(db); err != nil {
		log.Printf("Failed to run message-to-branch migration: %v", err)
		// Don't return error - migration is not critical for startup
	}

	return db, nil
}
