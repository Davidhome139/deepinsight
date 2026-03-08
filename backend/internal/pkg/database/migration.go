package database

import (
	"log"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"backend/internal/models"
)

// MigrateMessagesToBranches migrates existing messages to branches
// Migration principles:
// 1. Every conversation should have a default "main" branch
// 2. If a conversation doesn't have any branches, migrate all messages to "main" branch
// 3. Existing branches are preserved
func MigrateMessagesToBranches(db *gorm.DB) error {
	log.Println("Starting message-to-branch migration...")

	// Step 1: Get all conversations
	var conversations []models.Conversation
	if err := db.Find(&conversations).Error; err != nil {
		return err
	}

	log.Printf("Found %d conversations to process", len(conversations))

	for _, conv := range conversations {
		// Step 2: Check if conversation has a main branch
		var mainBranch models.Branch
		err := db.Where("conversation_id = ? AND is_main = ?", conv.ID, true).First(&mainBranch).Error

		if err == gorm.ErrRecordNotFound {
			// No main branch exists, create one
			mainBranch = models.Branch{
				ID:             uuid.New().String(),
				ConversationID: conv.ID,
				Name:           "Main",
				Description:    "Default main branch",
				IsMain:         true,
				IsActive:       true,
				Status:         "active",
				MessageCount:   0,
			}
			if err := db.Create(&mainBranch).Error; err != nil {
				log.Printf("Failed to create main branch for conversation %d: %v", conv.ID, err)
				return err
			}
			log.Printf("Created main branch %s for conversation %d", mainBranch.ID, conv.ID)
		} else if err != nil {
			return err
		}

		// Step 3: Migrate messages without branch_id to main branch
		result := db.Model(&models.Message{}).
			Where("conversation_id = ? AND branch_id IS NULL", conv.ID).
			Update("branch_id", mainBranch.ID)

		if result.Error != nil {
			log.Printf("Failed to migrate messages for conversation %d: %v", conv.ID, result.Error)
			return result.Error
		}

		if result.RowsAffected > 0 {
			log.Printf("Migrated %d messages to main branch for conversation %d", result.RowsAffected, conv.ID)
		}

		// Step 4: Update message_count for all branches of this conversation
		var branches []models.Branch
		if err := db.Where("conversation_id = ?", conv.ID).Find(&branches).Error; err != nil {
			return err
		}

		for _, branch := range branches {
			var count int64
			if err := db.Model(&models.Message{}).Where("branch_id = ?", branch.ID).Count(&count).Error; err != nil {
				return err
			}
			if err := db.Model(&branch).Update("message_count", count).Error; err != nil {
				return err
			}
		}

		// Step 5: Update branch_count for conversation
		branchCount := len(branches)
		updates := map[string]interface{}{
			"branch_count": branchCount,
		}

		// Step 6: Set ActiveBranchID to main branch if not set
		if conv.ActiveBranchID == nil {
			updates["active_branch_id"] = mainBranch.ID
		}

		if err := db.Model(&conv).Updates(updates).Error; err != nil {
			return err
		}

		log.Printf("Updated conversation %d: branch_count=%d", conv.ID, branchCount)
	}

	log.Println("Message-to-branch migration completed successfully")
	return nil
}
