# AI-AI 自动聊天功能设计文档

## 1. 系统架构概览

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              AI-AI Chat System                               │
├─────────────────────────────────────────────────────────────────────────────┤
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐    │
│  │   Frontend   │  │   Backend    │  │  AI Service  │  │   MCP Tools  │    │
│  │              │  │              │  │              │  │              │    │
│  │ • Config UI  │  │ • AIChatMgr  │  │ • DeepSeek   │  │ • Filesystem │    │
│  │ • Monitor    │  │ • Session    │  │ • Hunyuan    │  │ • Search     │    │
│  │ • Display    │  │ • Controller │  │ • Qwen       │  │ • Terminal   │    │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘    │
│         │                 │                 │                 │             │
│         └─────────────────┴─────────────────┴─────────────────┘             │
│                              WebSocket / HTTP                               │
└─────────────────────────────────────────────────────────────────────────────┘
```

## 2. 核心模块设计

### 2.1 数据模型

```go
// AIChatSession AI-AI 聊天会话
type AIChatSession struct {
    ID              string                    `json:"id" gorm:"primaryKey"`
    Title           string                    `json:"title"`
    Status          SessionStatus             `json:"status"` // pending, running, paused, completed, error
    CurrentRound    int                       `json:"current_round"`
    MaxRounds       int                       `json:"max_rounds"` // 0 = unlimited
    
    // 配置
    Topic           string                    `json:"topic"`
    GlobalConstraint string                   `json:"global_constraint"`
    TerminationConfig TerminationConfig      `json:"termination_config" gorm:"embedded"`
    
    // AI 配置
    AgentA          AgentConfig               `json:"agent_a" gorm:"embedded;embeddedPrefix:agent_a_"`
    AgentB          AgentConfig               `json:"agent_b" gorm:"embedded;embeddedPrefix:agent_b_"`
    
    // 对话历史
    Messages        []AIChatMessage           `json:"messages" gorm:"foreignKey:SessionID"`
    
    // 分支管理
    ParentID        *string                   `json:"parent_id,omitempty"`
    BranchPoint     *int                      `json:"branch_point,omitempty"` // 从第几轮分支
    
    // 元数据
    TokenUsage      TokenUsage                `json:"token_usage" gorm:"embedded"`
    StartedAt       *time.Time                `json:"started_at"`
    CompletedAt     *time.Time                `json:"completed_at"`
    CreatedAt       time.Time                 `json:"created_at"`
    UpdatedAt       time.Time                 `json:"updated_at"`
}

// AgentConfig AI 代理配置
type AgentConfig struct {
    Name            string                    `json:"name"`
    Role            string                    `json:"role"`           // 人设/系统提示词
    Style           StyleConfig               `json:"style" gorm:"embedded"`
    Model           string                    `json:"model"`          // 使用的AI模型
    Temperature     float64                   `json:"temperature"`
    MaxTokens       int                       `json:"max_tokens"`
    
    // MCP 工具权限
    AllowedTools    []string                  `json:"allowed_tools" gorm:"serializer:json"` // 允许的工具列表，空=全部
    BlockedTools    []string                  `json:"blocked_tools" gorm:"serializer:json"` // 禁止的工具列表
    
    // 运行时状态
    SystemPrompt    string                    `json:"system_prompt" gorm:"-"` // 动态生成的系统提示
}

// StyleConfig 风格配置
type StyleConfig struct {
    LanguageStyle   LanguageStyle             `json:"language_style"` // professional, casual, poetic
    KnowledgeLevel  KnowledgeLevel            `json:"knowledge_level"` // beginner, intermediate, expert
    Tone            string                    `json:"tone"`           // friendly, formal, humorous, etc.
}

// AIChatMessage AI-AI 对话消息
type AIChatMessage struct {
    ID              uint                      `json:"id" gorm:"primaryKey"`
    SessionID       string                    `json:"session_id"`
    Round           int                       `json:"round"`          // 第几轮
    AgentID         string                    `json:"agent_id"`       // agent_a 或 agent_b
    AgentName       string                    `json:"agent_name"`
    
    // 内容
    Content         string                    `json:"content"`
    MessageType     MessageType               `json:"message_type"`   // text, tool_call, tool_result, director_cmd
    
    // 工具调用
    ToolCalls       []ToolCall                `json:"tool_calls" gorm:"serializer:json"`
    ToolResults     []ToolResult              `json:"tool_results" gorm:"serializer:json"`
    
    // 元数据
    Tokens          int                       `json:"tokens"`
    Latency         int64                     `json:"latency_ms"`     // 响应延迟
    Timestamp       time.Time                 `json:"timestamp"`
}

// TerminationConfig 终止条件配置
type TerminationConfig struct {
    Type            TerminationType           `json:"type"` // fixed_rounds, open_ended, keyword
    MaxRounds       int                       `json:"max_rounds"`
    Keywords        []string                  `json:"keywords" gorm:"serializer:json"`
    SimilarityThreshold float64               `json:"similarity_threshold"` // 相似度阈值
    ConsecutiveSimilarRounds int              `json:"consecutive_similar_rounds"`
}

// DirectorCommand 导演指令
type DirectorCommand struct {
    ID              string                    `json:"id"`
    SessionID       string                    `json:"session_id"`
    TargetAgent     string                    `json:"target_agent"` // agent_a, agent_b, both
    Command         string                    `json:"command"`
    InsertAfterRound *int                     `json:"insert_after_round,omitempty"`
    Executed        bool                      `json:"executed"`
    Timestamp       time.Time                 `json:"timestamp"`
}

// TokenUsage Token 使用统计
type TokenUsage struct {
    AgentAInput     int                       `json:"agent_a_input"`
    AgentAOutput    int                       `json:"agent_a_output"`
    AgentBInput     int                       `json:"agent_b_input"`
    AgentBOutput    int                       `json:"agent_b_output"`
    Total           int                       `json:"total"`
}
```

### 2.2 核心服务接口

```go
// AIChatManager 会话管理器
type AIChatManager interface {
    // 会话管理
    CreateSession(config SessionConfig) (*AIChatSession, error)
    GetSession(id string) (*AIChatSession, error)
    ListSessions(filter SessionFilter) ([]*AIChatSession, error)
    DeleteSession(id string) error
    
    // 分支管理
    CreateBranch(sessionID string, round int, title string) (*AIChatSession, error)
    GetBranches(parentID string) ([]*AIChatSession, error)
    
    // 快照
    CreateSnapshot(sessionID string, title string) (*SessionSnapshot, error)
    RestoreSnapshot(snapshotID string) (*AIChatSession, error)
}

// AIChatEngine 对话引擎
type AIChatEngine interface {
    // 对话控制
    StartSession(sessionID string) error
    PauseSession(sessionID string) error
    ResumeSession(sessionID string) error
    StopSession(sessionID string) error
    
    // 执行对话轮次
    ExecuteRound(sessionID string) error
    ExecuteNextTurn(sessionID string) (*AIChatMessage, error)
    
    // 导演指令
    InjectDirectorCommand(sessionID string, cmd DirectorCommand) error
    
    // WebSocket 流
    SubscribeToSession(sessionID string, client chan<- StreamEvent)
    UnsubscribeFromSession(sessionID string, client chan<- StreamEvent)
}

// TerminationDetector 终止检测器
type TerminationDetector interface {
    ShouldTerminate(session *AIChatSession) (bool, TerminationReason)
    CalculateSimilarity(msg1, msg2 string) float64
}

// DialogEvaluator 对话评估器
type DialogEvaluator interface {
    Evaluate(session *AIChatSession) (*EvaluationReport, error)
}
```

### 2.3 核心流程伪代码

```
// AI-AI 对话主流程
function StartAIChatSession(sessionID):
    session = loadSession(sessionID)
    session.status = "running"
    session.startedAt = now()
    
    // 初始化系统提示词
    session.agentA.systemPrompt = buildSystemPrompt(session.agentA, session.topic, session.globalConstraint)
    session.agentB.systemPrompt = buildSystemPrompt(session.agentB, session.topic, session.globalConstraint)
    
    // 开始对话循环
    while session.status == "running":
        // 检查终止条件
        if shouldTerminate(session):
            break
        
        // 确定当前发言方
        currentAgent = getCurrentAgent(session)
        
        // 构建上下文
        context = buildContext(session, currentAgent)
        
        // 检查是否有待执行的导演指令
        directorCmd = getPendingDirectorCommand(session, currentAgent)
        if directorCmd:
            context = injectDirectorCommand(context, directorCmd)
            markCommandExecuted(directorCmd)
        
        // 调用 AI 生成回复
        response = callAI(currentAgent, context)
        
        // 处理工具调用
        if response.hasToolCalls():
            toolResults = executeToolCalls(response.toolCalls, currentAgent.allowedTools)
            response = callAIWithToolResults(currentAgent, context, toolResults)
        
        // 保存消息
        message = saveMessage(session, currentAgent, response)
        
        // 广播给客户端
        broadcastToClients(sessionID, {
            type: "message",
            data: message
        })
        
        // 更新状态
        session.currentRound = calculateRound(session)
        updateTokenUsage(session, response.tokens)
        
        // 应用延迟（如果配置了）
        if session.config.delayBetweenTurns > 0:
            sleep(session.config.delayBetweenTurns)
    
    session.status = "completed"
    session.completedAt = now()
    
    // 可选：生成评估报告
    if session.config.enableEvaluation:
        report = generateEvaluationReport(session)
        saveEvaluationReport(sessionID, report)

// 构建系统提示词
function buildSystemPrompt(agent, topic, constraint):
    prompt = ""
    
    // 角色设定
    prompt += f"你是{agent.name}。{agent.role}\n"
    
    // 风格设定
    prompt += f"你的语言风格应该是{agent.style.languageStyle}，"
    prompt += f"知识水平为{agent.style.knowledgeLevel}，"
    prompt += f"语气为{agent.style.tone}。\n"
    
    // 主题
    prompt += f"当前讨论主题：{topic}\n"
    
    // 全局限定
    if constraint:
        prompt += f"重要限定：{constraint}\n"
    
    // MCP 工具说明
    prompt += "你可以使用以下工具：\n"
    for tool in agent.allowedTools:
        prompt += f"- {tool}\n"
    
    return prompt

// 终止条件检测
function shouldTerminate(session):
    config = session.terminationConfig
    
    // 1. 固定轮数
    if config.type == "fixed_rounds" and session.currentRound >= config.maxRounds:
        return true, "max_rounds_reached"
    
    // 2. 关键词检测
    if config.type == "keyword":
        lastMessage = getLastMessage(session)
        for keyword in config.keywords:
            if contains(lastMessage.content, keyword):
                return true, "keyword_triggered"
    
    // 3. 相似度检测（循环检测）
    if config.similarityThreshold > 0:
        recentMessages = getLastNMessages(session, config.consecutiveSimilarRounds)
        if allSimilar(recentMessages, config.similarityThreshold):
            return true, "similarity_threshold"
    
    // 4. 主题偏离检测
    if isOffTopic(session, session.topic):
        return true, "off_topic"
    
    return false, ""
```

## 3. API 接口定义

### 3.1 REST API

```yaml
# 会话管理
POST   /api/v1/ai-chat/sessions              # 创建会话
GET    /api/v1/ai-chat/sessions              # 列表查询
GET    /api/v1/ai-chat/sessions/:id          # 获取详情
DELETE /api/v1/ai-chat/sessions/:id          # 删除会话
POST   /api/v1/ai-chat/sessions/:id/branch   # 创建分支

# 对话控制
POST   /api/v1/ai-chat/sessions/:id/start    # 开始对话
POST   /api/v1/ai-chat/sessions/:id/pause    # 暂停对话
POST   /api/v1/ai-chat/sessions/:id/resume   # 恢复对话
POST   /api/v1/ai-chat/sessions/:id/stop     # 停止对话

# 导演指令
POST   /api/v1/ai-chat/sessions/:id/director-command  # 插入导演指令

# 快照管理
POST   /api/v1/ai-chat/sessions/:id/snapshots         # 创建快照
GET    /api/v1/ai-chat/snapshots/:id                  # 获取快照
POST   /api/v1/ai-chat/snapshots/:id/restore          # 恢复快照

# 评估报告
GET    /api/v1/ai-chat/sessions/:id/evaluation        # 获取评估报告

# 模板
GET    /api/v1/ai-chat/templates             # 获取预制模板列表
GET    /api/v1/ai-chat/templates/:id         # 获取模板详情

# 导出
GET    /api/v1/ai-chat/sessions/:id/export   # 导出会话 (format: markdown, json, txt)
```

### 3.2 WebSocket API

```javascript
// 连接
ws://localhost:8080/ws/ai-chat/:sessionId

// 客户端 -> 服务端
{
  "type": "subscribe",
  "sessionId": "xxx"
}

{
  "type": "director_command",
  "targetAgent": "agent_a", // agent_a, agent_b, both
  "command": "请从现在开始反驳对方的观点"
}

{
  "type": "control",
  "action": "pause" // pause, resume, stop
}

// 服务端 -> 客户端
{
  "type": "message",
  "data": {
    "round": 5,
    "agentId": "agent_a",
    "agentName": "科学家",
    "content": "...",
    "messageType": "text",
    "timestamp": "..."
  }
}

{
  "type": "tool_call",
  "data": {
    "agentId": "agent_a",
    "toolName": "web_search",
    "arguments": {...},
    "result": "..."
  }
}

{
  "type": "status",
  "data": {
    "status": "running",
    "currentRound": 5,
    "totalRounds": 10,
    "tokenUsage": {...}
  }
}

{
  "type": "termination",
  "data": {
    "reason": "max_rounds_reached",
    "summary": "对话已完成10轮"
  }
}

{
  "type": "director_command_applied",
  "data": {
    "commandId": "xxx",
    "targetAgent": "agent_a",
    "affectedRounds": [6, 7, 8]
  }
}
```

## 4. 配置示例

### 4.1 完整配置 JSON

```json
{
  "title": "AI伦理辩论：科学家 vs 企业家",
  "topic": "人工智能发展应该优先考虑效率还是安全？",
  "globalConstraint": "每次发言不超过150字，禁止使用英文术语",
  "maxRounds": 8,
  "terminationConfig": {
    "type": "fixed_rounds",
    "maxRounds": 8,
    "keywords": ["结束", "总结", "到此为止"],
    "similarityThreshold": 0.85,
    "consecutiveSimilarRounds": 3
  },
  "delayConfig": {
    "minDelay": 1000,
    "maxDelay": 3000,
    "enableTypingEffect": true
  },
  "agentA": {
    "name": "李博士",
    "role": "你是一位严谨的AI安全研究员，专注于人工智能的伦理和安全问题。你倾向于谨慎行事，认为技术发展必须以安全为前提。",
    "style": {
      "languageStyle": "professional",
      "knowledgeLevel": "expert",
      "tone": "理性、审慎"
    },
    "model": "deepseek-chat",
    "temperature": 0.7,
    "maxTokens": 300,
    "allowedTools": ["search/web_search", "filesystem-local/read_file"],
    "blockedTools": ["terminal/execute_command"]
  },
  "agentB": {
    "name": "张总",
    "role": "你是一位科技创业公司的CEO，热衷于技术创新和商业化。你倾向于快速迭代，认为过度监管会阻碍进步。",
    "style": {
      "languageStyle": "casual",
      "knowledgeLevel": "expert",
      "tone": "热情、直接"
    },
    "model": "deepseek-chat",
    "temperature": 0.8,
    "maxTokens": 300,
    "allowedTools": ["search/web_search"],
    "blockedTools": []
  },
  "enableEvaluation": true,
  "evaluationConfig": {
    "evaluatorModel": "deepseek-chat",
    "criteria": ["topic_adherence", "role_consistency", "logical_coherence", "engagement"]
  },
  "enableAuditLog": true
}
```

### 4.2 预制模板示例

```json
{
  "templates": [
    {
      "id": "debate-tech-ethics",
      "name": "科技伦理辩论",
      "description": "两个AI就科技伦理话题进行辩论",
      "icon": "scale-balanced",
      "config": {
        "topic": "请选择一个伦理话题",
        "agentA": {
          "name": "保守派",
          "role": "你代表谨慎、安全优先的立场",
          "style": { "languageStyle": "professional", "tone": "理性" }
        },
        "agentB": {
          "name": "激进派", 
          "role": "你代表创新、效率优先的立场",
          "style": { "languageStyle": "casual", "tone": "热情" }
        },
        "maxRounds": 6
      }
    },
    {
      "id": "interview-simulation",
      "name": "面试模拟",
      "description": "模拟技术面试场景",
      "icon": "user-tie",
      "config": {
        "topic": "Go语言后端开发职位面试",
        "agentA": {
          "name": "面试官",
          "role": "你是一位资深技术面试官",
          "style": { "languageStyle": "professional", "tone": "严肃" }
        },
        "agentB": {
          "name": "候选人",
          "role": "你是一位应聘的开发者",
          "style": { "languageStyle": "casual", "tone": "自信" }
        },
        "maxRounds": 10
      }
    },
    {
      "id": "collaborative-problem-solving",
      "name": "协作解题",
      "description": "两个AI协作解决一个复杂问题",
      "icon": "puzzle-piece",
      "config": {
        "topic": "设计一个高并发的分布式系统",
        "globalConstraint": "双方需要互相补充，不能重复对方的观点",
        "agentA": {
          "name": "架构师",
          "role": "你专注于系统架构设计",
          "allowedTools": ["search/web_search", "filesystem-local/read_file"]
        },
        "agentB": {
          "name": "工程师",
          "role": "你专注于实现细节",
          "allowedTools": ["terminal/execute_command"]
        }
      }
    },
    {
      "id": "roleplay-historical",
      "name": "历史人物对话",
      "description": "历史人物之间的跨时空对话",
      "icon": "landmark",
      "config": {
        "topic": "讨论人工智能的未来",
        "agentA": {
          "name": "图灵",
          "role": "你是阿兰·图灵，计算机科学之父",
          "style": { "languageStyle": "professional", "tone": "深思熟虑" }
        },
        "agentB": {
          "name": "冯·诺依曼",
          "role": "你是约翰·冯·诺依曼，现代计算机架构之父",
          "style": { "languageStyle": "professional", "tone": "睿智" }
        }
      }
    }
  ]
}
```

## 5. 前端界面设计

### 5.1 页面结构

```
AI-AI Chat Page
├── Header
│   ├── Mode Switcher (AI-Human / AI-AI)
│   └── Session Title
├── Configuration Panel (可折叠)
│   ├── Basic Settings
│   │   ├── Topic Input
│   │   ├── Global Constraint
│   │   └── Max Rounds
│   ├── Agent A Config
│   │   ├── Name, Role, Style
│   │   ├── Model Selection
│   │   └── Tool Permissions
│   ├── Agent B Config
│   │   └── (同上)
│   ├── Termination Conditions
│   └── Template Selector
├── Control Bar
│   ├── Start / Pause / Stop Buttons
│   ├── Current Round Display (5/10)
│   ├── Token Usage Display
│   └── Director Command Input
├── Main Chat Area
│   ├── Message List (交替显示)
│   │   ├── Agent A Message
│   │   │   ├── Avatar, Name
│   │   │   ├── Content
│   │   │   └── Tool Calls (if any)
│   │   └── Agent B Message
│   │       └── (同上)
│   └── Typing Indicator
├── Real-time Monitor Panel (侧边栏/可折叠)
│   ├── Flow Graph (可视化对话流程)
│   ├── Token Usage Chart
│   ├── Round Timeline
│   └── Similarity Alert
└── Footer
    ├── Export Buttons
    ├── Create Branch Button
    └── Evaluation Report Link
```

### 5.2 关键组件

```vue
<!-- AIChatSession.vue - 主会话组件 -->
<template>
  <div class="ai-chat-session">
    <ConfigPanel 
      v-model:config="sessionConfig"
      :templates="templates"
      @apply-template="applyTemplate"
    />
    
    <ControlBar
      :status="sessionStatus"
      :current-round="currentRound"
      :total-rounds="maxRounds"
      @start="startSession"
      @pause="pauseSession"
      @stop="stopSession"
      @director-command="sendDirectorCommand"
    />
    
    <div class="chat-container">
      <MessageList
        :messages="messages"
        :agents="[agentA, agentB]"
      />
      
      <MonitorPanel
        :session-stats="sessionStats"
        :flow-graph="flowGraphData"
        :similarity-alerts="similarityAlerts"
      />
    </div>
  </div>
</template>
```

## 6. 数据库表结构

```sql
-- AI-AI 会话表
CREATE TABLE ai_chat_sessions (
    id VARCHAR(255) PRIMARY KEY,
    title VARCHAR(500),
    status VARCHAR(50) DEFAULT 'pending',
    current_round INT DEFAULT 0,
    max_rounds INT DEFAULT 10,
    topic TEXT,
    global_constraint TEXT,
    
    -- Agent A 配置
    agent_a_name VARCHAR(255),
    agent_a_role TEXT,
    agent_a_style_language VARCHAR(50),
    agent_a_style_knowledge VARCHAR(50),
    agent_a_style_tone VARCHAR(100),
    agent_a_model VARCHAR(100),
    agent_a_temperature FLOAT,
    agent_a_max_tokens INT,
    agent_a_allowed_tools JSON,
    agent_a_blocked_tools JSON,
    
    -- Agent B 配置
    agent_b_name VARCHAR(255),
    agent_b_role TEXT,
    agent_b_style_language VARCHAR(50),
    agent_b_style_knowledge VARCHAR(50),
    agent_b_style_tone VARCHAR(100),
    agent_b_model VARCHAR(100),
    agent_b_temperature FLOAT,
    agent_b_max_tokens INT,
    agent_b_allowed_tools JSON,
    agent_b_blocked_tools JSON,
    
    -- 终止配置
    termination_type VARCHAR(50),
    termination_max_rounds INT,
    termination_keywords JSON,
    termination_similarity_threshold FLOAT,
    termination_consecutive_similar_rounds INT,
    
    -- 分支管理
    parent_id VARCHAR(255),
    branch_point INT,
    
    -- Token 统计
    token_agent_a_input INT DEFAULT 0,
    token_agent_a_output INT DEFAULT 0,
    token_agent_b_input INT DEFAULT 0,
    token_agent_b_output INT DEFAULT 0,
    token_total INT DEFAULT 0,
    
    -- 时间戳
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (parent_id) REFERENCES ai_chat_sessions(id)
);

-- AI-AI 消息表
CREATE TABLE ai_chat_messages (
    id SERIAL PRIMARY KEY,
    session_id VARCHAR(255) NOT NULL,
    round INT NOT NULL,
    agent_id VARCHAR(50) NOT NULL,
    agent_name VARCHAR(255),
    content TEXT,
    message_type VARCHAR(50) DEFAULT 'text',
    tool_calls JSON,
    tool_results JSON,
    tokens INT DEFAULT 0,
    latency_ms BIGINT,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (session_id) REFERENCES ai_chat_sessions(id) ON DELETE CASCADE
);

-- 导演指令表
CREATE TABLE ai_chat_director_commands (
    id VARCHAR(255) PRIMARY KEY,
    session_id VARCHAR(255) NOT NULL,
    target_agent VARCHAR(50) NOT NULL,
    command TEXT NOT NULL,
    insert_after_round INT,
    executed BOOLEAN DEFAULT FALSE,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (session_id) REFERENCES ai_chat_sessions(id) ON DELETE CASCADE
);

-- 会话快照表
CREATE TABLE ai_chat_snapshots (
    id VARCHAR(255) PRIMARY KEY,
    session_id VARCHAR(255) NOT NULL,
    title VARCHAR(500),
    round INT NOT NULL,
    snapshot_data JSON NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (session_id) REFERENCES ai_chat_sessions(id) ON DELETE CASCADE
);

-- 评估报告表
CREATE TABLE ai_chat_evaluations (
    id VARCHAR(255) PRIMARY KEY,
    session_id VARCHAR(255) NOT NULL,
    report JSON NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (session_id) REFERENCES ai_chat_sessions(id) ON DELETE CASCADE
);

-- 审计日志表
CREATE TABLE ai_chat_audit_logs (
    id SERIAL PRIMARY KEY,
    session_id VARCHAR(255) NOT NULL,
    event_type VARCHAR(100) NOT NULL,
    event_data JSON,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (session_id) REFERENCES ai_chat_sessions(id) ON DELETE CASCADE
);

-- 索引
CREATE INDEX idx_ai_chat_messages_session ON ai_chat_messages(session_id);
CREATE INDEX idx_ai_chat_messages_round ON ai_chat_messages(session_id, round);
CREATE INDEX idx_ai_chat_sessions_status ON ai_chat_sessions(status);
CREATE INDEX idx_ai_chat_sessions_parent ON ai_chat_sessions(parent_id);
```

## 7. 实现优先级

### Phase 1: 核心功能 (MVP)
1. 基础会话管理 (创建、启动、停止)
2. 双AI交替对话
3. 基础配置 (角色、主题、轮数)
4. WebSocket 实时推送
5. 基础前端界面

### Phase 2: 增强功能
1. 导演指令
2. 分支管理
3. 快照功能
4. 工具调用权限控制
5. Token 监控

### Phase 3: 高级功能
1. 智能终止检测
2. 对话评估器
3. 预制模板库
4. 导出功能
5. 审计日志

### Phase 4: 扩展功能
1. 多AI群聊 (3+)
2. 对话流图谱可视化
3. 高级分析面板
