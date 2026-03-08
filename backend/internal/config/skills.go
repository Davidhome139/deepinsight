package config

import (
	"log"
	"sync"

	"github.com/spf13/viper"
)

// SkillsConfig MCP Skills configuration
// See: https://github.com/timescale/tiger-skills-mcp-server
type SkillsConfig struct {
	Sources  map[string]SkillSource `mapstructure:",remain"`
	Settings SkillSettings          `mapstructure:"settings"`
}

// SkillSource represents a skill source (local or GitHub)
type SkillSource struct {
	Type           string   `mapstructure:"type" json:"type"`           // local_collection, local, github_collection, github
	Path           string   `mapstructure:"path" json:"path,omitempty"` // Local path or GitHub path
	Repo           string   `mapstructure:"repo" json:"repo,omitempty"` // GitHub repository (owner/repo)
	IgnoredPaths   []string `mapstructure:"ignored_paths" json:"ignored_paths,omitempty"`
	DisabledSkills []string `mapstructure:"disabled_skills" json:"disabled_skills,omitempty"`
	EnabledSkills  []string `mapstructure:"enabled_skills" json:"enabled_skills,omitempty"`
	Enabled        bool     `mapstructure:"enabled" json:"enabled"`
}

// LoadedSkill represents a skill loaded from SKILL.md
type LoadedSkill struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	SourceKey   string `json:"source_key"`
	SourceType  string `json:"source_type"`
	Path        string `json:"path"`
	Content     string `json:"content,omitempty"`
	Enabled     bool   `json:"enabled"`
}

type SkillSettings struct {
	TTL             int  `mapstructure:"ttl" json:"ttl"`                           // Cache TTL in milliseconds
	SubagentEnabled bool `mapstructure:"subagent_enabled" json:"subagent_enabled"` // Enable subagent task execution
}

var (
	skillsConfig     *SkillsConfig
	skillsConfigPath string
	skillsMu         sync.RWMutex
	loadedSkills     []LoadedSkill
)

// LoadSkillsConfig loads skills configuration
func LoadSkillsConfig(path string) (*SkillsConfig, error) {
	v := viper.New()
	v.SetConfigFile(path)

	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}

	// Parse the YAML into a map first
	allSettings := v.AllSettings()

	cfg := &SkillsConfig{
		Sources: make(map[string]SkillSource),
	}

	// Extract settings if present
	if settings, ok := allSettings["settings"].(map[string]interface{}); ok {
		if ttl, ok := settings["ttl"].(int); ok {
			cfg.Settings.TTL = ttl
		}
		if subagent, ok := settings["subagent_enabled"].(bool); ok {
			cfg.Settings.SubagentEnabled = subagent
		}
		delete(allSettings, "settings")
	}

	// Parse remaining entries as skill sources
	for key, value := range allSettings {
		if sourceMap, ok := value.(map[string]interface{}); ok {
			source := SkillSource{Enabled: true} // Default enabled

			if t, ok := sourceMap["type"].(string); ok {
				source.Type = t
			}
			if p, ok := sourceMap["path"].(string); ok {
				source.Path = p
			}
			if r, ok := sourceMap["repo"].(string); ok {
				source.Repo = r
			}
			if e, ok := sourceMap["enabled"].(bool); ok {
				source.Enabled = e
			}
			if ignored, ok := sourceMap["ignored_paths"].([]interface{}); ok {
				for _, p := range ignored {
					if s, ok := p.(string); ok {
						source.IgnoredPaths = append(source.IgnoredPaths, s)
					}
				}
			}
			if disabled, ok := sourceMap["disabled_skills"].([]interface{}); ok {
				for _, s := range disabled {
					if str, ok := s.(string); ok {
						source.DisabledSkills = append(source.DisabledSkills, str)
					}
				}
			}
			if enabled, ok := sourceMap["enabled_skills"].([]interface{}); ok {
				for _, s := range enabled {
					if str, ok := s.(string); ok {
						source.EnabledSkills = append(source.EnabledSkills, str)
					}
				}
			}

			cfg.Sources[key] = source
		}
	}

	skillsMu.Lock()
	skillsConfig = cfg
	skillsConfigPath = path
	skillsMu.Unlock()

	log.Printf("Loaded skills config from %s, sources: %d", path, len(cfg.Sources))
	return cfg, nil
}

// GetSkillsConfig returns the skills configuration
func GetSkillsConfig() *SkillsConfig {
	skillsMu.RLock()
	defer skillsMu.RUnlock()
	return skillsConfig
}

// GetSkillSources returns skill sources as a list for API
func GetSkillSources() []map[string]interface{} {
	skillsMu.RLock()
	defer skillsMu.RUnlock()

	if skillsConfig == nil {
		return []map[string]interface{}{}
	}

	result := make([]map[string]interface{}, 0, len(skillsConfig.Sources))
	for key, source := range skillsConfig.Sources {
		result = append(result, map[string]interface{}{
			"key":             key,
			"name":            key,
			"type":            source.Type,
			"path":            source.Path,
			"repo":            source.Repo,
			"enabled":         source.Enabled,
			"ignored_paths":   source.IgnoredPaths,
			"disabled_skills": source.DisabledSkills,
			"enabled_skills":  source.EnabledSkills,
		})
	}
	return result
}

// GetLoadedSkills returns all loaded skills
func GetLoadedSkills() []LoadedSkill {
	skillsMu.RLock()
	defer skillsMu.RUnlock()
	return loadedSkills
}

// SetLoadedSkills sets the loaded skills
func SetLoadedSkills(skills []LoadedSkill) {
	skillsMu.Lock()
	defer skillsMu.Unlock()
	loadedSkills = skills
}

// UpdateSkillSource updates a skill source
func UpdateSkillSource(key string, source SkillSource) {
	skillsMu.Lock()
	defer skillsMu.Unlock()

	if skillsConfig == nil {
		skillsConfig = &SkillsConfig{
			Sources: make(map[string]SkillSource),
		}
	}
	skillsConfig.Sources[key] = source

	saveSkillsConfig()
}

// DeleteSkillSource deletes a skill source
func DeleteSkillSource(key string) {
	skillsMu.Lock()
	defer skillsMu.Unlock()

	if skillsConfig == nil || skillsConfig.Sources == nil {
		return
	}
	delete(skillsConfig.Sources, key)

	saveSkillsConfig()
}

// saveSkillsConfig saves the current config to file
func saveSkillsConfig() {
	v := viper.New()
	v.SetConfigFile(skillsConfigPath)

	// Write sources
	for key, source := range skillsConfig.Sources {
		v.Set(key+".type", source.Type)
		if source.Path != "" {
			v.Set(key+".path", source.Path)
		}
		if source.Repo != "" {
			v.Set(key+".repo", source.Repo)
		}
		if len(source.IgnoredPaths) > 0 {
			v.Set(key+".ignored_paths", source.IgnoredPaths)
		}
		if len(source.DisabledSkills) > 0 {
			v.Set(key+".disabled_skills", source.DisabledSkills)
		}
		if len(source.EnabledSkills) > 0 {
			v.Set(key+".enabled_skills", source.EnabledSkills)
		}
	}

	// Write settings
	v.Set("settings", skillsConfig.Settings)

	if err := v.WriteConfig(); err != nil {
		log.Printf("Failed to save skills config: %v", err)
	}
}

// Legacy compatibility - these functions maintain backward compatibility
// Skill represents legacy skill format for backward compatibility
type Skill struct {
	Name        string `mapstructure:"name" json:"name"`
	Enabled     bool   `mapstructure:"enabled" json:"enabled"`
	Description string `mapstructure:"description" json:"description"`
	Category    string `mapstructure:"category" json:"category"`
}

// UpdateSkill updates a skill (legacy compatibility)
func UpdateSkill(name string, skill Skill) {
	// Convert to new format - create a local skill source
	source := SkillSource{
		Type:    "local",
		Path:    "./skills/" + name,
		Enabled: skill.Enabled,
	}
	UpdateSkillSource(name, source)
}

// DeleteSkill deletes a skill (legacy compatibility)
func DeleteSkill(name string) {
	DeleteSkillSource(name)
}
