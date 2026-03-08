# DeepInsight - AI Assistant Platform

A full-stack AI assistant platform with multi-model support, conversation branching, RAG knowledge base, and MCP (Model Context Protocol) integration.

## Features

### Core Chat Features
- **Multi-Model Support** - Integrates with multiple AI providers:
  - DeepSeek (deepseek-chat, deepseek-reasoner)
  - Alibaba Qwen (qwen-turbo, qwen-plus, qwen-max)
  - Tencent Hunyuan (hunyuan-lite, hunyuan-standard, hunyuan-pro)
  - Baidu Wenxin (ERNIE-4.0, ERNIE-3.5)
  - MiniMax (abab6.5s-chat, abab5.5-chat)
  - ByteDance Doubao (doubao-pro-4k, doubao-pro-32k)
  - OpenAI (gpt-4o, gpt-4o-mini)

- **Conversation Branching** - Fork conversations to explore alternatives
  - Create branches at any point
  - Switch between branches seamlessly
  - Track message counts per branch
  - Default "Main" branch for all conversations

- **Real-time Streaming** - SSE-based streaming responses
- **Context Sidebar** - View recent conversation context

### AI-AI Chat
Automated multi-agent conversations with customizable templates:
- Pre-defined templates (Tech Ethics Debate, World Hot News, I Ching Wisdom, etc.)
- Customizable agent personalities and roles
- Web search integration for real-time information
- Export conversation to Markdown

### MCP Integration
Extensible tool system via Model Context Protocol:
- **filesystem-local** - File operations
- **terminal** - Safe command execution
- **search** - Web search (Baidu, Serper)
- **memory** - Knowledge graph persistence
- **code-analysis** - Code review tools
- **playwright/puppeteer** - Browser automation

### RAG Knowledge Base
- PostgreSQL with pgvector for semantic search
- Document upload and processing
- Automatic context retrieval

## Tech Stack

| Component | Technology |
|-----------|------------|
| Backend | Go 1.23, Gin, GORM |
| Database | PostgreSQL 15 + pgvector |
| Cache | Redis |
| Frontend | Vue 3, TypeScript, Element Plus |
| Build | Vite, Docker |
| MCP | mark3labs/mcp-go |

## Quick Start

### Prerequisites
- Docker Desktop
- Git

### Production

```bash
git clone https://github.com/Davidhome139/deepinsight.git
cd deepinsight
docker-compose up --build

# Access: http://localhost (Frontend), http://localhost:8080 (API)
```

### Development

```bash
# Windows
scripts\dev.bat up

# Linux/Mac
./scripts/dev.sh up

# Access: http://localhost:5173 (Frontend), http://localhost:8080 (API)
```

## Project Structure

```
deepinsight/
├── backend/
│   ├── cmd/                    # Entry point
│   ├── internal/
│   │   ├── api/handlers/       # HTTP handlers
│   │   ├── models/             # Database models
│   │   ├── services/
│   │   │   ├── ai/             # AI provider adapters
│   │   │   ├── aichat/         # AI-AI chat service
│   │   │   ├── branch/         # Branch management
│   │   │   ├── chat/           # Chat with MCP
│   │   │   ├── mcp/            # MCP manager
│   │   │   ├── rag/            # RAG service
│   │   │   └── search/         # Search service
│   │   └── pkg/database/       # Database & migrations
│   └── config/
│       ├── config.yaml         # Main config
│       ├── models.yaml         # AI provider config
│       ├── mcpservers.yaml     # MCP server config
│       └── searchs.yaml        # Search provider config
├── frontend/
│   ├── public/config/          # Runtime configs (JSON)
│   └── src/
│       ├── components/         # Vue components
│       ├── views/              # Page views
│       ├── stores/             # Pinia stores
│       └── api/                # API client
├── docker-compose.yaml         # Production
└── docker-compose.dev.yaml     # Development
```

## Configuration

### AI Providers (`backend/config/models.yaml`)

```yaml
providers:
  deepseek:
    name: DeepSeek
    enabled: true
    apikey: your-api-key-here
    baseurl: https://api.deepseek.com
    models:
      - deepseek-chat
      - deepseek-reasoner
```

### MCP Servers (`backend/config/mcpservers.yaml`)

```yaml
servers:
  memory:
    name: Memory
    enabled: true
    command: npx
    args:
      - -y
      - '@modelcontextprotocol/server-memory'
```

### Search Providers (`backend/config/searchs.yaml`)

```yaml
providers:
  baidu:
    name: Baidu AI Search
    enabled: true
    apikey: your-api-key-here
settings:
  defaultprovider: baidu
```

## API Endpoints

### Chat
| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/chat/stream` | Stream chat messages |
| GET | `/api/v1/chat/conversations` | List conversations |
| GET | `/api/v1/chat/conversations/:id/branches` | Get branches |
| POST | `/api/v1/chat/conversations/:id/branches` | Create branch |
| GET | `/api/v1/chat/branches/:id/messages` | Get branch messages |

### AI-AI Chat
| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/aichat/sessions` | Create session |
| WS | `/api/v1/aichat/sessions/:id/ws` | WebSocket stream |
| GET | `/api/v1/aichat/templates` | List templates |
| GET | `/api/v1/aichat/sessions/:id/export` | Export to Markdown |

### Settings
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/settings/ai-providers` | AI provider settings |
| GET | `/api/v1/settings/mcpservers` | MCP server settings |

## Development Commands

| Command | Description |
|---------|-------------|
| `up` | Start development services |
| `down` | Stop all services |
| `logs [service]` | View logs |
| `build` | Rebuild and start |
| `clean` | Clean all data |
| `shell-backend` | Enter backend shell |
| `db` | Connect to PostgreSQL |

## Data Persistence

- **PostgreSQL Volume**: Chat messages, branches, sessions
- **MCP Memory**: `./backend/memory/memory.json`
- **Chat History**: `./backend/chat/`

## Security

- JWT-based authentication
- Whitelisted terminal commands only
- File system access restricted to configured paths
- API keys stored in backend config (not exposed to frontend)

## Troubleshooting

| Issue | Solution |
|-------|----------|
| MCP server not connected | Check Node.js availability in container |
| Database connection failed | Check `docker-compose logs db` |
| Search not working | Verify `defaultprovider` in searchs.yaml |

## License

MIT License

## Acknowledgments

- Built with Go, Vue 3, and Docker
- MCP integration powered by mark3labs/mcp-go
- Vector search powered by pgvector
