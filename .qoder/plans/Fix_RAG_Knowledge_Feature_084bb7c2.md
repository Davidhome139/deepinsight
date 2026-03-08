# Fix RAG/Knowledge Base Feature

## Problem Analysis

1. **White Screen on "Manage Documents"**: Route mismatch - button navigates to `/rag` but router defines `/knowledge`
2. **Document Upload Fails**: `DocumentChunk` model missing from AutoMigrate - table never created
3. **No Dark Mode Support**: KnowledgeBaseView.vue uses hardcoded light colors
4. **Route Inconsistency**: Breadcrumb uses `/rag`, router uses `/knowledge`

## Implementation Steps

### Step 1: Fix Database Migration (Critical)

**File:** `backend/internal/pkg/database/postgres.go`

Add `DocumentChunk` to AutoMigrate list (line ~58):
```go
&models.Document{},
&models.DocumentChunk{},  // ADD THIS LINE
```

### Step 2: Fix Route Mismatch

**Option A (Recommended):** Change router to use `/rag` for consistency

**File:** `frontend/src/router/index.ts` (line 57)
```typescript
// Change from:
path: '/knowledge',
// To:
path: '/rag',
```

This keeps the "Manage Documents" button working and matches the API prefix.

### Step 3: Update Breadcrumb Configuration

**File:** `frontend/src/components/Breadcrumb.vue` (line 60)

Already has `/rag` mapping - no change needed if we use Option A above.

### Step 4: Add Dark Mode Support to KnowledgeBaseView

**File:** `frontend/src/views/KnowledgeBaseView.vue`

Replace hardcoded colors with CSS variables. Key changes:

| Element | Current | Dark Mode |
|---------|---------|-----------|
| `.knowledge-base-view` | (none) | `background: var(--bg-primary)` |
| `.modal` | `background: white` | `background: var(--card-bg)` |
| `.modal h2` | (none) | `color: var(--text-primary)` |
| `.documents-section` | `background: white` | `background: var(--card-bg)` |
| `.document-card` | `background: #f8f9fa` | `background: var(--bg-secondary)` |
| `.result-item` | `background: #f8f9fa` | `background: var(--bg-secondary)` |
| `.chunk-item` | `background: #f8f9fa` | `background: var(--bg-secondary)` |
| Input borders | `border: 1px solid #e0e0e0` | `border-color: var(--border-primary)` |
| Text colors | `color: #666`, `#888` | `color: var(--text-secondary)`, `var(--text-tertiary)` |

### Step 5: Fix Text Extraction for PDF/DOCX (Optional Enhancement)

**File:** `backend/internal/api/handlers/rag.go`

Current PDF extraction is basic, DOCX returns empty. This works but could be improved later. For now, focus on TXT/MD which work correctly.

## Files to Modify

1. `backend/internal/pkg/database/postgres.go` - Add DocumentChunk to migration
2. `frontend/src/router/index.ts` - Change `/knowledge` to `/rag`
3. `frontend/src/views/KnowledgeBaseView.vue` - Add dark mode CSS variables

## Testing Checklist

- [ ] Navigate to `/rag` directly - should show KnowledgeBaseView
- [ ] Click "Manage Documents" in Chat - should navigate to Knowledge Base
- [ ] Upload a .txt or .md file - should show "processing" then "ready"
- [ ] View document chunks - should display chunk list
- [ ] Search knowledge base - should return results with scores
- [ ] Toggle dark mode - all elements should be readable
- [ ] Test in Docker container - verify database migration runs
- [ ] Test on Windows and Linux - verify file upload works

## Cross-Platform Notes

- No file system storage used - documents stored in PostgreSQL `content` column
- Vector embeddings stored in pgvector - requires `CREATE EXTENSION vector`
- All paths are database-based, no local file mapping needed
- API uses multipart form upload - platform independent