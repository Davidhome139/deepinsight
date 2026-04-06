package rag

import (
	"backend/internal/models"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// RAGService handles document processing and retrieval-augmented generation
// Uses Aliyun DashScope Embedding API (text-embedding-v3)
type RAGService struct {
	db      *gorm.DB
	apiKey  string
	baseURL string
}

// Aliyun DashScope Embedding API structures
type embeddingRequest struct {
	Model      string           `json:"model"`
	Input      embeddingInput   `json:"input"`
	Parameters *embeddingParams `json:"parameters,omitempty"`
}

type embeddingInput struct {
	Texts []string `json:"texts"`
}

type embeddingParams struct {
	TextType string `json:"text_type,omitempty"` // query or document
}

type embeddingResponse struct {
	Output struct {
		Embeddings []struct {
			Embedding []float32 `json:"embedding"`
			TextIndex int       `json:"text_index"`
		} `json:"embeddings"`
	} `json:"output"`
	Usage struct {
		TotalTokens int `json:"total_tokens"`
	} `json:"usage"`
	RequestID string `json:"request_id"`
	Code      string `json:"code,omitempty"`
	Message   string `json:"message,omitempty"`
}

// NewRAGService creates a new RAG service using Aliyun DashScope
func NewRAGService(db *gorm.DB, apiKey string) *RAGService {
	return &RAGService{
		db:      db,
		apiKey:  apiKey,
		baseURL: "https://dashscope.aliyuncs.com/api/v1/services/embeddings/text-embedding/text-embedding",
	}
}

// SetBaseURL allows overriding the API endpoint
func (s *RAGService) SetBaseURL(url string) {
	s.baseURL = url
}

// UploadDocument uploads and processes a document
func (s *RAGService) UploadDocument(userID uint, filename, fileType, content string, fileSize int64) (*models.Document, error) {
	doc := &models.Document{
		ID:        uuid.New().String(),
		UserID:    userID,
		Title:     filename,
		Filename:  filename,
		FileType:  fileType,
		FileSize:  fileSize,
		Content:   content,
		Status:    models.DocStatusProcessing,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.db.Create(doc).Error; err != nil {
		return nil, fmt.Errorf("failed to create document: %w", err)
	}

	// Process document asynchronously
	go s.processDocument(doc)

	return doc, nil
}

// processDocument splits document into chunks and generates embeddings
func (s *RAGService) processDocument(doc *models.Document) {
	chunks := s.splitIntoChunks(doc.Content, 1000, 200) // chunk size 1000, overlap 200

	var documentChunks []models.DocumentChunk
	var chunkTexts []string

	for i, chunk := range chunks {
		chunkTexts = append(chunkTexts, chunk)
		documentChunks = append(documentChunks, models.DocumentChunk{
			ID:         uuid.New().String(),
			DocumentID: doc.ID,
			Content:    chunk,
			ChunkIndex: i,
			TokenCount: s.estimateTokens(chunk),
			CreatedAt:  time.Now(),
		})
	}

	// Generate embeddings in batches
	batchSize := 20
	for i := 0; i < len(chunkTexts); i += batchSize {
		end := i + batchSize
		if end > len(chunkTexts) {
			end = len(chunkTexts)
		}

		embeddings, err := s.generateEmbeddings(chunkTexts[i:end])
		if err != nil {
			s.updateDocumentStatus(doc.ID, models.DocStatusFailed, err.Error())
			return
		}

		for j, embedding := range embeddings {
			documentChunks[i+j].Embedding = embedding
		}
	}

	// Save chunks to database using raw SQL to properly handle vector type
	for _, chunk := range documentChunks {
		embeddingStr, _ := chunk.Embedding.Value()
		err := s.db.Exec(`
			INSERT INTO document_chunks (id, document_id, content, chunk_index, token_count, embedding, created_at)
			VALUES (?, ?, ?, ?, ?, ?::vector, ?)
		`, chunk.ID, chunk.DocumentID, chunk.Content, chunk.ChunkIndex, chunk.TokenCount, embeddingStr, chunk.CreatedAt).Error
		if err != nil {
			s.updateDocumentStatus(doc.ID, models.DocStatusFailed, err.Error())
			return
		}
	}

	// Update document status
	s.db.Model(&models.Document{}).Where("id = ?", doc.ID).Updates(map[string]interface{}{
		"status":      models.DocStatusReady,
		"chunk_count": len(documentChunks),
		"updated_at":  time.Now(),
	})
}

// splitIntoChunks splits text into overlapping chunks
func (s *RAGService) splitIntoChunks(text string, chunkSize, overlap int) []string {
	var chunks []string
	text = strings.TrimSpace(text)

	if len(text) <= chunkSize {
		return []string{text}
	}

	// Split by paragraphs first
	paragraphs := strings.Split(text, "\n\n")
	currentChunk := ""

	for _, para := range paragraphs {
		para = strings.TrimSpace(para)
		if para == "" {
			continue
		}

		if len(currentChunk)+len(para)+2 <= chunkSize {
			if currentChunk != "" {
				currentChunk += "\n\n"
			}
			currentChunk += para
		} else {
			if currentChunk != "" {
				chunks = append(chunks, currentChunk)
				// Keep overlap from current chunk
				words := strings.Fields(currentChunk)
				if len(words) > overlap/5 {
					overlapWords := words[len(words)-overlap/5:]
					currentChunk = strings.Join(overlapWords, " ")
				} else {
					currentChunk = ""
				}
			}

			// If paragraph itself is too long, split it
			if len(para) > chunkSize {
				sentences := s.splitIntoSentences(para)
				for _, sent := range sentences {
					if len(currentChunk)+len(sent)+1 <= chunkSize {
						if currentChunk != "" {
							currentChunk += " "
						}
						currentChunk += sent
					} else {
						if currentChunk != "" {
							chunks = append(chunks, currentChunk)
						}
						currentChunk = sent
					}
				}
			} else {
				if currentChunk != "" {
					currentChunk += "\n\n"
				}
				currentChunk += para
			}
		}
	}

	if currentChunk != "" {
		chunks = append(chunks, currentChunk)
	}

	return chunks
}

// splitIntoSentences splits text into sentences
func (s *RAGService) splitIntoSentences(text string) []string {
	var sentences []string
	var current strings.Builder

	runes := []rune(text)
	for i, r := range runes {
		current.WriteRune(r)

		// Check for sentence endings
		if r == '.' || r == '!' || r == '?' || r == '。' || r == '！' || r == '？' {
			// Check if next char is space or end
			if i+1 >= len(runes) || runes[i+1] == ' ' || runes[i+1] == '\n' {
				sentences = append(sentences, strings.TrimSpace(current.String()))
				current.Reset()
			}
		}
	}

	if current.Len() > 0 {
		sentences = append(sentences, strings.TrimSpace(current.String()))
	}

	return sentences
}

// estimateTokens estimates token count for text
func (s *RAGService) estimateTokens(text string) int {
	// Rough estimation: ~4 chars per token for English, ~2 chars for Chinese
	return len(text) / 3
}

// generateEmbeddings generates embeddings for texts using Aliyun DashScope API
func (s *RAGService) generateEmbeddings(texts []string) ([]models.Vector, error) {
	reqBody := embeddingRequest{
		Model:      "text-embedding-v3",
		Input:      embeddingInput{Texts: texts},
		Parameters: &embeddingParams{TextType: "document"},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", s.baseURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var apiResp embeddingResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if apiResp.Code != "" {
		return nil, fmt.Errorf("API error: %s - %s", apiResp.Code, apiResp.Message)
	}

	embeddings := make([]models.Vector, len(texts))
	for _, item := range apiResp.Output.Embeddings {
		embeddings[item.TextIndex] = models.Vector(item.Embedding)
	}

	return embeddings, nil
}

// Query searches the knowledge base for relevant chunks
func (s *RAGService) Query(userID uint, query models.RAGQuery) ([]models.RAGResult, error) {
	if query.TopK <= 0 {
		query.TopK = 5
	}
	if query.Threshold <= 0 {
		query.Threshold = 0.5
	}

	// Generate embedding for query
	embeddings, err := s.generateEmbeddings([]string{query.Query})
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	queryEmbedding := embeddings[0]
	embeddingStr := s.embeddingToString(queryEmbedding)

	// Build SQL query with pgvector using subquery to filter by score
	sql := fmt.Sprintf(`
		SELECT * FROM (
			SELECT 
				dc.id as chunk_id,
				dc.document_id,
				d.filename,
				dc.content,
				dc.chunk_index,
				1 - (dc.embedding <=> '%s'::vector) as score
			FROM document_chunks dc
			JOIN documents d ON dc.document_id = d.id
			WHERE d.user_id = ? AND d.status = 'ready'
		) sub
		WHERE score >= %f
		ORDER BY score DESC
		LIMIT %d
	`, embeddingStr, query.Threshold, query.TopK)

	args := []interface{}{userID}

	var results []models.RAGResult
	if err := s.db.Raw(sql, args...).Scan(&results).Error; err != nil {
		return nil, fmt.Errorf("failed to query knowledge base: %w", err)
	}

	return results, nil
}

// embeddingToString converts embedding to PostgreSQL vector string format
func (s *RAGService) embeddingToString(embedding models.Vector) string {
	val, _ := embedding.Value()
	if val == nil {
		return "[]"
	}
	return val.(string)
}

// GetDocuments returns user's documents
func (s *RAGService) GetDocuments(userID uint, limit, offset int) ([]models.Document, int64, error) {
	var docs []models.Document
	var total int64

	query := s.db.Model(&models.Document{}).Where("user_id = ?", userID)
	query.Count(&total)

	err := query.Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&docs).Error

	return docs, total, err
}

// GetDocument returns a single document
func (s *RAGService) GetDocument(id string, userID uint) (*models.Document, error) {
	var doc models.Document
	err := s.db.Where("id = ? AND user_id = ?", id, userID).First(&doc).Error
	if err != nil {
		return nil, err
	}
	return &doc, nil
}

// DeleteDocument deletes a document and its chunks
func (s *RAGService) DeleteDocument(id string, userID uint) error {
	// Verify ownership
	var doc models.Document
	if err := s.db.Where("id = ? AND user_id = ?", id, userID).First(&doc).Error; err != nil {
		return err
	}

	// Delete chunks first
	if err := s.db.Where("document_id = ?", id).Delete(&models.DocumentChunk{}).Error; err != nil {
		return err
	}

	// Delete document
	return s.db.Delete(&doc).Error
}

// GetDocumentChunks returns chunks for a document
func (s *RAGService) GetDocumentChunks(docID string, userID uint) ([]models.DocumentChunk, error) {
	// Verify ownership
	var doc models.Document
	if err := s.db.Where("id = ? AND user_id = ?", docID, userID).First(&doc).Error; err != nil {
		return nil, err
	}

	var chunks []models.DocumentChunk
	err := s.db.Where("document_id = ?", docID).Order("chunk_index").Find(&chunks).Error
	return chunks, err
}

// updateDocumentStatus updates document processing status
func (s *RAGService) updateDocumentStatus(docID, status, errorMsg string) {
	s.db.Model(&models.Document{}).Where("id = ?", docID).Updates(map[string]interface{}{
		"status":     status,
		"error_msg":  errorMsg,
		"updated_at": time.Now(),
	})
}

// BuildContext builds RAG context for chat prompts
func (s *RAGService) BuildContext(userID uint, query string, topK int) (*models.RAGContext, error) {
	ragQuery := models.RAGQuery{
		Query:     query,
		TopK:      topK,
		Threshold: 0.7,
	}

	results, err := s.Query(userID, ragQuery)
	if err != nil {
		return nil, err
	}

	return &models.RAGContext{
		Query:   query,
		Results: results,
	}, nil
}
