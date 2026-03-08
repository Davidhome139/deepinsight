# Tencent Yuanbao AI Assistant

A full-stack AI assistant application inspired by Tencent Yuanbao, featuring multi-model support, MCP (Model Context Protocol) integration, and a modern chat interface.

## Features

### Core Capabilities
- **Multi-Model Support** - Integrates with multiple AI providers (DeepSeek, Tencent Hunyuan, Qwen, MiniMax, etc.)
- **Real-time Chat** - WebSocket-based streaming responses with context persistence
- **AI-AI Chat** - Automated multi-agent conversations with customizable templates
  - Pre-defined conversation templates (Tech Ethics Debate, World Hot News, I Ching Wisdom, etc.)
  - Customizable agent personalities and roles
  - Proactive web search integration for real-time information
  - Session persistence and history
  - Real-time streaming output with turn-by-turn display
  - Export conversation history to Markdown
- **MCP Integration** - Extensible tool system via Model Context Protocol
- **Web Search** - Built-in search capabilities with multiple providers (Baidu, Serper, etc.)
- **Knowledge Graph Memory** - Persistent memory using MCP Memory server
- **Code Analysis** - Built-in code review and analysis tools
- **File System Access** - Read files and list directories
- **Terminal Execution** - Execute safe shell commands

### AI Programming Features
- **AI Programming Page** - Dedicated interface for code-related tasks
- **Multi-Agent System** - Agent-based task execution with planning
- **Skills System** - Pre-defined skills for common tasks (git, security, code review, etc.)
- **Vector Memory & RAG** - PostgreSQL with pgvector for semantic storage

### Supported MCP Servers
- **filesystem-local** - Local file system operations
- **terminal** - Safe command execution
- **search** - Web search integration
- **code-analysis** - Code analysis and suggestions
- **memory** - Knowledge graph-based persistent memory
- **fetch** - HTTP requests
- **playwright** - Browser automation
- **puppeteer** - Browser control
- **sqlite** - SQLite database operations
- **context7** - Context management

## Tech Stack

### Backend
- **Language**: Go 1.23
- **Framework**: Gin
- **ORM**: GORM
- **Database**: PostgreSQL 15 with pgvector extension
- **Cache**: Redis
- **MCP**: mark3labs/mcp-go
- **Authentication**: JWT

### Frontend
- **Framework**: Vue 3 + TypeScript
- **UI Library**: Element Plus + Naive UI
- **Build Tool**: Vite
- **State Management**: Pinia
- **Editor**: Monaco Editor
- **Terminal**: xterm.js

### Infrastructure
- **Containerization**: Docker + Docker Compose
- **Web Server**: Nginx (production)
- **Development**: Hot reload with Air (backend) and Vite HMR (frontend)

## Quick Start

### Prerequisites
- Docker Desktop
- Git

### Production Deployment

```bash
# Clone the repository
git clone <repository-url>
cd newDouBao

# Start all services
docker-compose up --build

# Access the application
# Frontend: http://localhost
# Backend API: http://localhost:8080
```

### Development Environment

```bash
# Windows
scripts\dev.bat up

# Linux/Mac
chmod +x scripts/dev.sh
./scripts/dev.sh up
```

Development services:
- Frontend: http://localhost:5173
- Backend API: http://localhost:8080
- Database: localhost:5432
- Redis: localhost:6379

## Project Structure

```
newDouBao/
├── backend/                 # Go backend application
│   ├── cmd/                # Application entry point
│   ├── internal/           # Internal packages
│   │   ├── api/           # HTTP handlers and routes
│   │   ├── config/        # Configuration management
│   │   ├── models/        # Database models
│   │   ├── services/      # Business logic
│   │   │   ├── agent/     # Agent system
│   │   │   ├── ai/        # AI provider integration
│   │   │   ├── aichat/    # AI-AI chat service
│   │   │   ├── chat/      # Chat service with MCP
│   │   │   ├── mcp/       # MCP manager
│   │   │   └── search/    # Search service
│   │   └── pkg/           # Shared packages
│   ├── config/            # Configuration files
│   │   ├── config.yaml    # Main configuration
│   │   ├── mcpservers.yaml # MCP server configurations
│   │   └── searchs.yaml   # Search provider configurations
│   ├── chat/              # Chat history storage
│   ├── memory/            # MCP Memory persistence
│   └── plans/             # Agent execution plans
├── frontend/               # Vue 3 frontend application
│   ├── public/
│   │   └── config/
│   │       └── ai-chat-templates.json  # AI-AI chat templates (non-dev editable)
│   ├── src/
│   │   ├── components/    # Vue components
│   │   ├── views/         # Page views
│   │   ├── stores/        # Pinia stores
│   │   ├── config/        # TypeScript configurations
│   │   └── api/           # API client
│   └── dist/              # Production build
├── scripts/               # Development scripts
│   ├── dev.bat           # Windows development script
│   └── dev.sh            # Linux/Mac development script
├── docker-compose.yaml    # Production compose file
├── docker-compose.dev.yaml # Development compose file
└── DEVELOPMENT.md         # Detailed development guide
```

## Configuration

### AI Providers

Configure AI providers in `backend/config/config.yaml`:

```yaml
ai:
  providers:
    deepseek:
      enabled: true
      apikey: "your-api-key"
      baseurl: "https://api.deepseek.com"
    tencent:
      enabled: true
      secretid: "your-secret-id"
      secretkey: "your-secret-key"
```

### MCP Servers

Configure MCP servers in `backend/config/mcpservers.yaml`:

```yaml
servers:
  memory:
    name: Memory (知识图谱)
    enabled: true
    type: mcpservers
    command: npx
    args:
      - -y
      - '@modelcontextprotocol/server-memory'
    env:
      MEMORY_FILE_PATH: /app/memory/memory.json
```

### Search Providers

Configure search providers in `backend/config/searchs.yaml`:

```yaml
providers:
  baidu:
    name: 百度AI搜索
    enabled: true
    apikey: "your-api-key"
    baseurl: "https://qianfan.baidubce.com/v2/ai_search/chat/completions"
settings:
  defaultprovider: baidu
```

### AI-AI Chat Templates

**For Non-Developers**: Edit templates in `frontend/public/config/ai-chat-templates.json`:

```json
[
  {
    "id": "my-custom-template",
    "name": "Custom Template Name",
    "description": "Template description",
    "icon": "🎯",
    "config": {
      "title": "Session Title",
      "topic": "Discussion topic",
      "max_rounds": 10,
      "global_constraint": "Optional constraints",
      "agent_a": {
        "name": "Agent A Name",
        "role": "Agent A role description",
        "style": {
          "language_style": "professional",
          "knowledge_level": "expert",
          "tone": "Objective"
        },
        "model": "deepseek-chat",
        "temperature": 0.7,
        "max_tokens": 300,
        "allowed_tools": ["search/web_search"]
      },
      "agent_b": {
        "name": "Agent B Name",
        "role": "Agent B role description",
        "style": {
          "language_style": "casual",
          "knowledge_level": "expert",
          "tone": "Insightful"
        },
        "model": "deepseek-chat",
        "temperature": 0.8,
        "max_tokens": 300,
        "allowed_tools": []
      }
    }
  }
]
```

**Template Fields**:
- `id` (required): Unique identifier
- `name` (required): Template display name
- `description` (required): Short description
- `icon` (required): Emoji icon
- `config.title` (required): Session title
- `config.topic` (required): Discussion topic
- `config.max_rounds` (required): Maximum conversation rounds
- `config.global_constraint` (optional): Global constraints for both agents
- `config.agent_a` (optional): Agent A configuration
- `config.agent_b` (optional): Agent B configuration

**How to Add/Modify Templates**:
1. Open `frontend/public/config/ai-chat-templates.json` with any text editor
2. Copy an existing template as a starting point
3. Modify the values to create your custom template
4. Save the file
5. Refresh the browser to see the changes (no rebuild needed!)

## Development Commands

### Windows (scripts\dev.bat)

| Command | Description |
|---------|-------------|
| `up` | Start all development services |
| `down` | Stop all services |
| `restart` | Restart all services |
| `logs [service]` | View logs (backend/frontend/db/redis) |
| `build` | Rebuild and start |
| `clean` | Clean all data and containers |
| `shell-backend` | Enter backend container shell |
| `shell-frontend` | Enter frontend container shell |
| `db` | Connect to PostgreSQL database |

### Linux/Mac (./scripts/dev.sh)

Same commands as Windows, plus:

| Command | Description |
|---------|-------------|
| `status` | View service status |

## API Documentation

API documentation is available via Swagger UI at `/swagger/index.html` when running in development mode.

### Key Endpoints

#### Chat Service
- `POST /api/v1/chat/stream` - Stream chat messages
- `GET /api/v1/chat/models` - List available models
- `GET /api/v1/chat/mcp-tools` - List available MCP tools
- `GET /api/v1/chat/history` - Get chat history
- `POST /api/v1/chat/title` - Generate chat title

#### AI-AI Chat Service
- `POST /api/v1/aichat/sessions` - Create new AI-AI chat session
- `GET /api/v1/aichat/sessions/:id` - Get session details
- `GET /api/v1/aichat/sessions` - List all sessions
- `DELETE /api/v1/aichat/sessions/:id` - Delete session
- `WS /api/v1/aichat/sessions/:id/ws` - WebSocket connection for real-time updates
- `POST /api/v1/aichat/sessions/:id/start` - Start conversation
- `POST /api/v1/aichat/sessions/:id/pause` - Pause conversation
- `POST /api/v1/aichat/sessions/:id/resume` - Resume conversation
- `POST /api/v1/aichat/sessions/:id/stop` - Stop conversation
- `GET /api/v1/aichat/sessions/:id/export` - Export conversation to Markdown
- `GET /api/v1/aichat/templates` - List available templates
- `GET /api/v1/aichat/templates/:id` - Get template details
- `GET /api/v1/aichat/models` - List available AI models
- `GET /api/v1/aichat/mcp-tools` - List available MCP tools

#### Settings
- `GET /api/v1/settings/ai-providers` - AI provider settings
- `GET /api/v1/settings/mcpservers` - MCP server settings

## Data Persistence

### Production
- **PostgreSQL**: `postgres_data` Docker volume
  - Chat messages and conversations
  - AI-AI chat sessions and messages
  - User data and authentication
- **MCP Memory**: `./backend/memory/memory.json`
- **Chat History**: `./backend/chat/`
- **Agent Plans**: `./backend/plans/`

### Development
- Additional volumes for hot reload and caching
- See `DEVELOPMENT.md` for details

## MCP Tool Usage

### Smart Intent Detection

The system automatically detects user intent and triggers appropriate MCP tools:

| Intent | Keywords | MCP Tool |
|--------|----------|----------|
| List Directory | "列出", "list", "show files" | filesystem-local/list_directory |
| Read File | "读取", "read", "open" | filesystem-local/read_file |
| Web Search | "搜索", "search", "查找" | search/web_search |
| Execute Command | "执行", "execute", "run" | terminal/execute_command |
| Code Analysis | "分析代码", "analyze code" | code-analysis/analyze_code |

### Manual Tool Selection

Users can also manually select MCP tools from the dropdown in the chat interface.

## AI-AI Chat Usage

### Getting Started

1. **Access AI-AI Chat**: Navigate to the AI-AI Chat page from the main menu
2. **Choose a Template**: Click "📋 选择模板" to select a pre-defined conversation template
3. **Configure Session** (optional): Click "⚙️ 配置" to customize:
   - Session title and topic
   - Maximum conversation rounds
   - Agent personalities and roles
   - AI models and parameters (temperature, max tokens)
   - Enable/disable MCP tools (e.g., web search)
4. **Start Conversation**: Click "▶️ 开始对话" to begin the AI-AI conversation
5. **Monitor Progress**: Watch real-time streaming output as agents converse
6. **Control Flow**: Use Pause/Resume/Stop buttons to control the conversation
7. **Export Results**: Click export to save the conversation as Markdown

### Available Templates

#### 🌍 World Hot News
- **Description**: Analyze current international hot news
- **Agents**: International Observer vs News Commentator
- **Features**: Automatic web search for latest news
- **Use Case**: Stay updated on global events with AI analysis

#### ☯️ I Ching Wisdom
- **Description**: Explore life wisdom based on I Ching theory
- **Agents**: I Ching Master vs Modern Philosopher
- **Use Case**: Philosophical discussions combining ancient wisdom with modern life

#### ⚖️ Tech Ethics Debate
- **Description**: Debate on technology ethics topics
- **Use Case**: Explore ethical implications of AI and technology

#### 👔 Technical Interview
- **Description**: Simulate technical interview scenarios
- **Use Case**: Practice interview questions and answers

#### 🧩 Collaborative Problem Solving
- **Description**: Two AIs collaborate to solve technical problems
- **Use Case**: System design discussions and architecture planning

#### 🏛️ Historical Figures Dialogue
- **Description**: Cross-temporal dialogue between historical figures
- **Use Case**: Imaginative conversations exploring ideas across time

### Session Management

- **Session History**: View all past AI-AI conversations in the left sidebar
- **Load Session**: Click on a history item to view previous conversations
- **Delete Session**: Remove unwanted sessions from history
- **Session Persistence**: All conversations are automatically saved

### MCP Tools in AI-AI Chat

AI-AI chat supports MCP tools just like regular chat:
- **Proactive Search**: Sessions with topics containing "latest", "news", "today" automatically trigger web search
- **Reactive Search**: Agents can request searches during conversation
- **Tool Results**: Search results are formatted and displayed in the conversation
- **Custom Styling**: Search results use FangSong font for easy identification

## Security Considerations

- **Terminal Commands**: Only whitelisted safe commands are allowed (ls, pwd, cat, etc.)
- **File System Access**: Restricted to allowed paths configured in MCP servers
- **API Keys**: Stored in configuration files, not exposed to frontend
- **Authentication**: JWT-based authentication for API access

## Troubleshooting

### Common Issues

1. **Port Conflicts**
   - Modify `docker-compose.yaml` to use different ports

2. **MCP Server Not Connected**
   - Check MCP server configuration
   - Verify Node.js is available in the container

3. **Database Connection Failed**
   - Check database health: `docker-compose logs db`
   - Verify connection string in config

4. **Search Not Working**
   - Verify `defaultprovider` is set in `searchs.yaml`
   - Check API keys for search providers

### Logs

```bash
# View all logs
docker-compose logs -f

# View specific service
docker-compose logs -f backend
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Commit your changes
4. Push to the branch
5. Create a Pull Request

## License

[Your License Here]

## Acknowledgments

- Inspired by Tencent Yuanbao
- Built with Go, Vue 3, and Docker
- MCP integration powered by mark3labs/mcp-go
