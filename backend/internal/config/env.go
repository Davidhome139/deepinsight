package config

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

var (
	envLoaded = false
)

// LoadEnvFile loads environment variables from a .env file
func LoadEnvFile(configDir string) error {
	if envLoaded {
		return nil
	}

	envPath := filepath.Join(configDir, ".env")
	if _, err := os.Stat(envPath); err == nil {
		log.Printf("Loading environment variables from: %s", envPath)
		
		// 直接读取文件内容，避免 Viper 自动转换键为小写
		content, err := os.ReadFile(envPath)
		if err != nil {
			log.Printf("Warning: Failed to read .env file: %v", err)
			return nil
		}
		
		lines := strings.Split(string(content), "\n")
		loadedCount := 0
		
		for _, line := range lines {
			line = strings.TrimSpace(line)
			// 跳过空行和注释
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			
			// 分割键值对
			parts := strings.SplitN(line, "=", 2)
			if len(parts) != 2 {
				continue
			}
			
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			
			// 移除值两端的引号
			value = strings.Trim(value, `"'`)
			
			// 设置环境变量
			if err := os.Setenv(key, value); err != nil {
				log.Printf("Warning: Failed to set env var %s: %v", key, err)
			} else {
				loadedCount++
			}
		}
		
		log.Printf("Successfully loaded %d environment variables from .env", loadedCount)
	} else {
		log.Printf("No .env file found at: %s", envPath)
	}

	envLoaded = true
	return nil
}

// SetupViperEnv sets up viper to read environment variables with common prefix
func SetupViperEnv(v *viper.Viper, prefix string) {
	if prefix != "" {
		v.SetEnvPrefix(prefix)
	}
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
}

// GetEnvWithDefault gets an environment variable or returns a default value
func GetEnvWithDefault(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists && value != "" {
		return value
	}
	return defaultValue
}

// UpdateEnvFile updates the .env file with the given key-value pairs
func UpdateEnvFile(configDir string, keyValuePairs map[string]string) error {
	envPath := filepath.Join(configDir, ".env")
	
	// Read existing content
	var content string
	if _, err := os.Stat(envPath); err == nil {
		existingContent, err := os.ReadFile(envPath)
		if err != nil {
			return err
		}
		content = string(existingContent)
	}
	
	// Update or add key-value pairs
	lines := strings.Split(content, "\n")
	updatedLines := make([]string, 0)
	updatedKeys := make(map[string]bool)
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			updatedLines = append(updatedLines, line)
			continue
		}
		
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			updatedLines = append(updatedLines, line)
			continue
		}
		
		key := strings.TrimSpace(parts[0])
		if value, exists := keyValuePairs[key]; exists {
			updatedLines = append(updatedLines, key+"="+value)
			updatedKeys[key] = true
		} else {
			updatedLines = append(updatedLines, line)
		}
	}
	
	// Add new key-value pairs that don't exist yet
	for key, value := range keyValuePairs {
		if !updatedKeys[key] {
			updatedLines = append(updatedLines, key+"="+value)
		}
	}
	
	// Write back to file
	updatedContent := strings.Join(updatedLines, "\n")
	return os.WriteFile(envPath, []byte(updatedContent), 0644)
}
