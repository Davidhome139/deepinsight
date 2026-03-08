# DeepInsight - AI Assistant Platform

An intelligent AI assistant platform that helps you have conversations with multiple AI models, manage knowledge, and automate tasks.

## Features

### 💬 Smart Chat
Chat with AI using your preferred model. The system supports real-time streaming responses and remembers your conversation context.

**Supported AI Models:**
- DeepSeek (deepseek-chat, deepseek-reasoner)
- Alibaba Qwen (qwen-turbo, qwen-plus, qwen-max)
- Tencent Hunyuan (hunyuan-lite, hunyuan-standard, hunyuan-pro)
- Baidu Wenxin (ERNIE-4.0, ERNIE-3.5)
- MiniMax, ByteDance Doubao, OpenAI GPT-4o

### 🌿 Conversation Branching
Explore different conversation paths without losing your original thread:
- Create branches at any message to try alternative approaches
- Switch between branches instantly
- Each branch maintains its own message history
- Perfect for comparing different AI responses

### 🤖 AI-AI Chat
Watch two AI agents have a conversation on any topic:
- **World Hot News**: AI analyzes current international events
- **Tech Ethics Debate**: Explore technology ethics from different perspectives
- **I Ching Wisdom**: Ancient philosophy meets modern thinking
- **Technical Interview**: Practice interview scenarios
- **Problem Solving**: Two AIs collaborate to solve problems

### 🔧 MCP Tools
Extend AI capabilities with built-in tools:
- **Web Search**: Search the internet for real-time information
- **File Operations**: Read and browse local files
- **Terminal**: Execute safe shell commands
- **Memory**: Persistent knowledge storage
- **Code Analysis**: Review and analyze code

### 📚 Knowledge Base (RAG)
Upload documents and let AI answer questions based on your content:
- Automatic document processing
- Semantic search for relevant context
- Vector-based retrieval

## Installation

### Requirements
- Docker Desktop (Windows/Mac/Linux)
- Git

### Quick Start (Recommended)

```bash
# 1. Clone the repository
git clone https://github.com/Davidhome139/deepinsight.git
cd deepinsight

# 2. Start all services
docker-compose up --build

# 3. Open in browser
# http://localhost
```

That's it! The application will be running at `http://localhost`.

### Development Setup

For development with hot-reload:

```bash
# Windows
scripts\dev.bat up

# Linux/Mac
chmod +x scripts/dev.sh
./scripts/dev.sh up
```

Development URLs:
- Frontend: http://localhost:5173
- Backend API: http://localhost:8080

## Usage Guide

### Starting a Chat

1. Open the application in your browser
2. Select an AI model from the dropdown (default: deepseek-chat)
3. Type your message and press Enter or click Send
4. Watch the AI response stream in real-time

### Creating a Conversation Branch

1. Hover over any message in the conversation
2. Click the "Branch" icon
3. A new branch is created from that point
4. Switch between branches using the branch panel on the right

### Using AI-AI Chat

1. Navigate to "AI-AI Chat" from the menu
2. Click "Select Template" to choose a conversation topic
3. (Optional) Click "Configure" to customize:
   - Agent names and personalities
   - AI models for each agent
   - Maximum conversation rounds
   - Enable/disable web search
4. Click "Start" to begin the conversation
5. Use Pause/Resume/Stop to control the flow
6. Export the conversation as Markdown when done

### Using MCP Tools

The AI automatically detects when to use tools. You can also:
1. Select a specific tool from the MCP dropdown
2. Type your request (e.g., "search for latest AI news")
3. The AI will use the appropriate tool and include results in the response

### Managing Knowledge Base

1. Navigate to "Knowledge Base" from the menu
2. Click "Upload" to add documents (PDF, TXT, MD)
3. Documents are processed and indexed automatically
4. In chat, the AI will reference relevant documents when answering

## Configuration

### Adding Your API Keys

Edit `backend/config/models.yaml`:

```yaml
providers:
  deepseek:
    enabled: true
    apikey: your-deepseek-api-key
  
  aliyun:
    enabled: true
    apikey: your-aliyun-api-key
```

### Configuring Search

Edit `backend/config/searchs.yaml`:

```yaml
providers:
  baidu:
    enabled: true
    apikey: your-baidu-api-key
    
settings:
  defaultprovider: baidu
```

### Adding MCP Servers

Edit `backend/config/mcpservers.yaml`:

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

## Common Commands

| Command | Description |
|---------|-------------|
| `docker-compose up` | Start all services |
| `docker-compose down` | Stop all services |
| `docker-compose logs -f` | View logs |
| `docker-compose logs backend` | View backend logs |
| `scripts/dev.bat up` | Start development mode (Windows) |
| `./scripts/dev.sh up` | Start development mode (Linux/Mac) |

## Troubleshooting

### Application won't start
```bash
# Check if ports are in use
docker-compose down
docker-compose up --build
```

### AI not responding
- Check your API key in `backend/config/models.yaml`
- Verify the provider is enabled (`enabled: true`)
- Check backend logs: `docker-compose logs backend`

### MCP tools not working
- Ensure Node.js is available in the container
- Check MCP configuration in `backend/config/mcpservers.yaml`
- Restart services: `docker-compose restart`

### Database connection error
```bash
# Reset database
docker-compose down -v
docker-compose up --build
```

## System Requirements

- **Memory**: 4GB RAM minimum, 8GB recommended
- **Disk**: 2GB free space
- **OS**: Windows 10+, macOS 10.15+, or Linux

## Support

- **Issues**: https://github.com/Davidhome139/deepinsight/issues
- **Documentation**: See `docs/` folder for detailed guides

## License

MIT License
