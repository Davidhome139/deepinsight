package handlers

import (
	"backend/internal/models"
	"backend/internal/services/rag"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// RAGHandler handles RAG/knowledge base endpoints
type RAGHandler struct {
	ragService *rag.RAGService
}

// NewRAGHandler creates a new RAG handler
func NewRAGHandler(ragService *rag.RAGService) *RAGHandler {
	return &RAGHandler{ragService: ragService}
}

// UploadDocument handles document upload
// POST /api/v1/rag/documents
func (h *RAGHandler) UploadDocument(c *gin.Context) {
	userID := c.GetUint("user_id")

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File is required"})
		return
	}
	defer file.Close()

	// Read file content
	content, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read file"})
		return
	}

	// Determine file type
	filename := header.Filename
	fileType := getFileType(filename)
	if fileType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported file type. Supported: txt, md, pdf, docx"})
		return
	}

	// Extract text from file
	text, err := extractText(content, fileType)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to extract text from file: " + err.Error()})
		return
	}

	if len(text) < 10 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Document content is too short"})
		return
	}

	// Upload document
	doc, err := h.ragService.UploadDocument(userID, filename, fileType, text, header.Size)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "Document uploaded and processing started",
		"document": doc,
	})
}

// ListDocuments returns user's documents
// GET /api/v1/rag/documents
func (h *RAGHandler) ListDocuments(c *gin.Context) {
	userID := c.GetUint("user_id")

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	docs, total, err := h.ragService.GetDocuments(userID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"documents": docs,
		"total":     total,
		"limit":     limit,
		"offset":    offset,
	})
}

// GetDocument returns a single document
// GET /api/v1/rag/documents/:id
func (h *RAGHandler) GetDocument(c *gin.Context) {
	userID := c.GetUint("user_id")
	docID := c.Param("id")

	doc, err := h.ragService.GetDocument(docID, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Document not found"})
		return
	}

	c.JSON(http.StatusOK, doc)
}

// DeleteDocument deletes a document
// DELETE /api/v1/rag/documents/:id
func (h *RAGHandler) DeleteDocument(c *gin.Context) {
	userID := c.GetUint("user_id")
	docID := c.Param("id")

	if err := h.ragService.DeleteDocument(docID, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Document deleted"})
}

// GetDocumentChunks returns chunks for a document
// GET /api/v1/rag/documents/:id/chunks
func (h *RAGHandler) GetDocumentChunks(c *gin.Context) {
	userID := c.GetUint("user_id")
	docID := c.Param("id")

	chunks, err := h.ragService.GetDocumentChunks(docID, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Document not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"chunks": chunks})
}

// Query searches the knowledge base
// POST /api/v1/rag/query
func (h *RAGHandler) Query(c *gin.Context) {
	userID := c.GetUint("user_id")

	var req models.RAGQuery
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	results, err := h.ragService.Query(userID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"query":   req.Query,
		"results": results,
		"count":   len(results),
	})
}

// Helper functions

func getFileType(filename string) string {
	filename = strings.ToLower(filename)
	if strings.HasSuffix(filename, ".txt") {
		return "txt"
	}
	if strings.HasSuffix(filename, ".md") {
		return "md"
	}
	if strings.HasSuffix(filename, ".pdf") {
		return "pdf"
	}
	if strings.HasSuffix(filename, ".docx") {
		return "docx"
	}
	return ""
}

func extractText(content []byte, fileType string) (string, error) {
	switch fileType {
	case "txt", "md":
		return string(content), nil
	case "pdf":
		return extractPDFText(content)
	case "docx":
		return extractDocxText(content)
	default:
		return "", nil
	}
}

// extractPDFText extracts text from PDF (simplified - in production use a proper PDF library)
func extractPDFText(content []byte) (string, error) {
	// For now, return a message that PDF parsing needs a library
	// In production, use libraries like pdfcpu or call external tools
	text := string(content)

	// Basic PDF text extraction - look for text streams
	// This is a simplified version - real PDF parsing needs proper libraries
	var extracted strings.Builder

	// Try to find readable text in the PDF
	for i := 0; i < len(text); i++ {
		if text[i] >= 32 && text[i] < 127 {
			extracted.WriteByte(text[i])
		} else if text[i] == '\n' || text[i] == '\r' {
			extracted.WriteByte('\n')
		}
	}

	result := extracted.String()
	if len(result) < 100 {
		return "", nil // PDF parsing not supported without proper library
	}

	return result, nil
}

// extractDocxText extracts text from DOCX
func extractDocxText(content []byte) (string, error) {
	// DOCX is a ZIP containing XML files
	// For production, use a proper DOCX library
	// This is a placeholder - real implementation needs zip + xml parsing
	return "", nil
}
