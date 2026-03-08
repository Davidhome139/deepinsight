package handlers

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

// PromptTemplateHandler handles prompt template operations
type PromptTemplateHandler struct {
	templates *PromptTemplateConfig
}

// PromptTemplateConfig represents the full template configuration
type PromptTemplateConfig struct {
	Version     string                    `json:"version"`
	Description string                    `json:"description"`
	Variables   map[string]GlobalVariable `json:"variables"`
	Categories  []PromptCategory          `json:"categories"`
}

// GlobalVariable represents a global variable definition
type GlobalVariable struct {
	Label   string   `json:"label"`
	Options []string `json:"options"`
}

// PromptCategory represents a category of templates
type PromptCategory struct {
	ID     string        `json:"id"`
	Name   string        `json:"name"`
	Icon   string        `json:"icon"`
	Topics []PromptTopic `json:"topics"`
}

// PromptTopic represents a topic within a category
type PromptTopic struct {
	ID        string           `json:"id"`
	Name      string           `json:"name"`
	Templates []PromptTemplate `json:"templates"`
}

// PromptTemplate represents a single prompt template
type PromptTemplate struct {
	ID              string                    `json:"id"`
	Name            string                    `json:"name"`
	Complexity      string                    `json:"complexity"`
	Description     string                    `json:"description"`
	Template        string                    `json:"template"`
	Variables       []string                  `json:"variables"`
	CustomVariables map[string]CustomVariable `json:"customVariables"`
}

// CustomVariable represents a template-specific variable
type CustomVariable struct {
	Label    string   `json:"label"`
	Type     string   `json:"type"`
	Options  []string `json:"options,omitempty"`
	Default  string   `json:"default,omitempty"`
	Required bool     `json:"required,omitempty"`
}

// NewPromptTemplateHandler creates a new prompt template handler
func NewPromptTemplateHandler() *PromptTemplateHandler {
	h := &PromptTemplateHandler{}
	h.loadTemplates()
	return h
}

// loadTemplates loads templates from JSON file
func (h *PromptTemplateHandler) loadTemplates() {
	data, err := os.ReadFile("config/prompt-templates.json")
	if err != nil {
		// Log error but don't fail - return empty templates
		h.templates = &PromptTemplateConfig{
			Categories: []PromptCategory{},
		}
		return
	}

	var config PromptTemplateConfig
	if err := json.Unmarshal(data, &config); err != nil {
		h.templates = &PromptTemplateConfig{
			Categories: []PromptCategory{},
		}
		return
	}

	h.templates = &config
}

// GetCategories returns all template categories
func (h *PromptTemplateHandler) GetCategories(c *gin.Context) {
	type CategorySummary struct {
		ID         string `json:"id"`
		Name       string `json:"name"`
		Icon       string `json:"icon"`
		TopicCount int    `json:"topic_count"`
	}

	categories := make([]CategorySummary, len(h.templates.Categories))
	for i, cat := range h.templates.Categories {
		categories[i] = CategorySummary{
			ID:         cat.ID,
			Name:       cat.Name,
			Icon:       cat.Icon,
			TopicCount: len(cat.Topics),
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"categories":       categories,
		"global_variables": h.templates.Variables,
	})
}

// GetCategory returns a single category with its topics and templates
func (h *PromptTemplateHandler) GetCategory(c *gin.Context) {
	categoryID := c.Param("id")

	for _, cat := range h.templates.Categories {
		if cat.ID == categoryID {
			c.JSON(http.StatusOK, cat)
			return
		}
	}

	c.JSON(http.StatusNotFound, gin.H{"error": "Category not found"})
}

// GetTemplate returns a single template by ID
func (h *PromptTemplateHandler) GetTemplate(c *gin.Context) {
	templateID := c.Param("id")

	for _, cat := range h.templates.Categories {
		for _, topic := range cat.Topics {
			for _, tmpl := range topic.Templates {
				if tmpl.ID == templateID {
					c.JSON(http.StatusOK, gin.H{
						"template":         tmpl,
						"category":         cat.Name,
						"topic":            topic.Name,
						"global_variables": h.templates.Variables,
					})
					return
				}
			}
		}
	}

	c.JSON(http.StatusNotFound, gin.H{"error": "Template not found"})
}

// RenderTemplate renders a template with provided variables
func (h *PromptTemplateHandler) RenderTemplate(c *gin.Context) {
	var req struct {
		TemplateID string            `json:"template_id"`
		Variables  map[string]string `json:"variables"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Find template
	var foundTemplate *PromptTemplate
	for _, cat := range h.templates.Categories {
		for _, topic := range cat.Topics {
			for _, tmpl := range topic.Templates {
				if tmpl.ID == req.TemplateID {
					foundTemplate = &tmpl
					break
				}
			}
		}
	}

	if foundTemplate == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Template not found"})
		return
	}

	// Render template with variables
	rendered := foundTemplate.Template
	for key, value := range req.Variables {
		rendered = strings.ReplaceAll(rendered, "{"+key+"}", value)
	}

	c.JSON(http.StatusOK, gin.H{
		"rendered_prompt": rendered,
		"template_name":   foundTemplate.Name,
		"complexity":      foundTemplate.Complexity,
	})
}

// SearchTemplates searches templates by keyword
func (h *PromptTemplateHandler) SearchTemplates(c *gin.Context) {
	query := strings.ToLower(c.Query("q"))
	complexity := c.Query("complexity")

	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Query parameter 'q' is required"})
		return
	}

	type SearchResult struct {
		TemplateID   string `json:"template_id"`
		TemplateName string `json:"template_name"`
		Complexity   string `json:"complexity"`
		Description  string `json:"description"`
		CategoryName string `json:"category_name"`
		TopicName    string `json:"topic_name"`
	}

	var results []SearchResult

	for _, cat := range h.templates.Categories {
		for _, topic := range cat.Topics {
			for _, tmpl := range topic.Templates {
				// Filter by complexity if specified
				if complexity != "" && tmpl.Complexity != complexity {
					continue
				}

				// Search in name, description, and template content
				if strings.Contains(strings.ToLower(tmpl.Name), query) ||
					strings.Contains(strings.ToLower(tmpl.Description), query) ||
					strings.Contains(strings.ToLower(tmpl.Template), query) ||
					strings.Contains(strings.ToLower(cat.Name), query) ||
					strings.Contains(strings.ToLower(topic.Name), query) {
					results = append(results, SearchResult{
						TemplateID:   tmpl.ID,
						TemplateName: tmpl.Name,
						Complexity:   tmpl.Complexity,
						Description:  tmpl.Description,
						CategoryName: cat.Name,
						TopicName:    topic.Name,
					})
				}
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"results": results,
		"count":   len(results),
	})
}

// GetAllTemplates returns all templates in a flat list
func (h *PromptTemplateHandler) GetAllTemplates(c *gin.Context) {
	c.JSON(http.StatusOK, h.templates)
}
