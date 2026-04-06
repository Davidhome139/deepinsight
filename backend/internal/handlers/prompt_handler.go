package handlers

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

// GetPromptFiles 获取对话的提示词文件列表
func GetPromptFiles(c *gin.Context) {
	conversationID := c.Param("id")
	if conversationID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Conversation ID is required"})
		return
	}

	promptDir := filepath.Join("/app/chat", conversationID)
	files, err := os.ReadDir(promptDir)
	if err != nil {
		if os.IsNotExist(err) {
			c.JSON(http.StatusOK, gin.H{"files": []string{}})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read prompt directory"})
		return
	}

	var fileList []string
	for _, file := range files {
		if !file.IsDir() && (strings.HasSuffix(file.Name(), ".md") || strings.HasSuffix(file.Name(), ".txt")) {
			fileList = append(fileList, file.Name())
		}
	}

	c.JSON(http.StatusOK, gin.H{"files": fileList})
}

// GetPromptFile 获取单个提示词文件内容
func GetPromptFile(c *gin.Context) {
	conversationID := c.Param("id")
	fileName := c.Param("fileName")
	if conversationID == "" || fileName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Conversation ID and file name are required"})
		return
	}

	filePath := filepath.Join("/app/chat", conversationID, fileName)
	content, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Prompt file not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read prompt file"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"content": string(content)})
}
