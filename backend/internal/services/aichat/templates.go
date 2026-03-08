package aichat

import (
	"encoding/json"
	"log"
	"os"

	"backend/internal/models"
	"backend/internal/pkg/database"
)

// TemplateService handles seeding of builtin templates from JSON config.
// Runtime template CRUD (GetTemplates, CreateTemplate, etc.) lives on AIChatService.
type TemplateService struct{}

// NewTemplateService loads /app/config/ai-chat-templates.json and upserts all
// entries as builtin templates into the database. Call this once at startup,
// after AutoMigrate has created the session_templates table.
func NewTemplateService() *TemplateService {
	s := &TemplateService{}
	s.initBuiltinTemplates()
	return s
}

// jsonTemplateEntry mirrors the JSON structure in ai-chat-templates.json
type jsonTemplateEntry struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Icon        string         `json:"icon"`
	Category    string         `json:"category"`
	Config      models.JSONMap `json:"config"`
}

// initBuiltinTemplates reads /app/config/ai-chat-templates.json and upserts
// all entries as builtin templates. Non-builtin (user-created) templates are
// never touched.
func (s *TemplateService) initBuiltinTemplates() {
	configPath := "/app/config/ai-chat-templates.json"

	data, err := os.ReadFile(configPath)
	if err != nil {
		log.Printf("[TemplateService] WARNING: cannot read %s: %v — skipping builtin template seeding", configPath, err)
		return
	}

	var entries []jsonTemplateEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		log.Printf("[TemplateService] WARNING: failed to parse %s: %v — skipping builtin template seeding", configPath, err)
		return
	}

	log.Printf("[TemplateService] Seeding %d builtin templates from %s", len(entries), configPath)

	for _, e := range entries {
		tmpl := models.SessionTemplate{
			ID:          e.ID,
			Name:        e.Name,
			Description: e.Description,
			Icon:        e.Icon,
			Category:    e.Category,
			IsBuiltin:   true,
			Config:      e.Config,
		}

		var existing models.SessionTemplate
		if err := database.DB.First(&existing, "id = ?", tmpl.ID).Error; err != nil {
			// Does not exist — create
			if createErr := database.DB.Create(&tmpl).Error; createErr != nil {
				log.Printf("[TemplateService] Failed to create template %s: %v", tmpl.ID, createErr)
			}
		} else if existing.IsBuiltin {
			// Exists as builtin — sync with latest JSON definition
			existing.Name = tmpl.Name
			existing.Description = tmpl.Description
			existing.Icon = tmpl.Icon
			existing.Category = tmpl.Category
			existing.Config = tmpl.Config
			if saveErr := database.DB.Save(&existing).Error; saveErr != nil {
				log.Printf("[TemplateService] Failed to update template %s: %v", tmpl.ID, saveErr)
			}
		}
		// User-created templates with the same ID are intentionally left untouched.
	}
}
