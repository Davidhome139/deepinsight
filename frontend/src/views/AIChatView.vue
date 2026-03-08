<template>
  <div class="ai-chat-view">
    <!-- 头部 -->
    <div class="header">
      <div class="header-left">
        <h2>🤖 AI-AI 自动聊天</h2>
        <div class="mode-switcher">
          <router-link to="/" class="mode-btn">AI-人类聊天</router-link>
          <span class="mode-btn active">AI-AI聊天</span>
        </div>
      </div>
      <div class="header-right">
        <button class="btn btn-secondary" @click="$router.push('/programming')">
          💻 AI Programming
        </button>
        <button class="btn btn-secondary" @click="$router.push('/video')">
          🎬 Video Generation
        </button>
        <button class="btn btn-secondary" @click="showTemplates = true">
          📋 选择模板
        </button>
        <button class="btn btn-primary" @click="showConfig = true" v-if="!isRunning">
          ⚙️ 配置
        </button>
      </div>
    </div>

    <!-- 主内容区 -->
    <div class="main-content">
      <!-- 会话历史侧边栏 -->
      <aside class="session-history" :class="{ collapsed: historyCollapsed }">
        <div class="history-header" v-show="!historyCollapsed">
          <h4>历史会话</h4>
          <button class="collapse-btn" @click="historyCollapsed = true">
            <el-icon><Fold /></el-icon>
          </button>
        </div>
        <div class="history-list" v-show="!historyCollapsed">
          <div 
            v-for="session in sessionHistory" 
            :key="session.id"
            class="history-item"
            :class="{ active: sessionId === session.id }"
            @click="loadSession(session.id)"
          >
            <div class="session-title">{{ session.title }}</div>
            <div class="session-meta">
              <span>{{ session.current_round }}/{{ session.max_rounds }} 轮</span>
              <span>{{ formatHistoryDate(session.created_at) }}</span>
            </div>
          </div>
          <div v-if="sessionHistory.length === 0" class="empty-history">
            暂无历史会话
          </div>
        </div>
        <div class="expand-btn" v-show="historyCollapsed" @click="historyCollapsed = false">
          <el-icon><Fold /></el-icon>
        </div>
      </aside>

      <!-- 配置面板 -->
      <div class="config-panel" v-if="showConfig && !isRunning">
        <div class="panel-header">
          <h3>会话配置</h3>
          <button class="close-btn" @click="showConfig = false">×</button>
        </div>
        
        <div class="panel-body">
          <!-- 基础配置 -->
          <div class="config-section">
            <h4>基础设置</h4>
            <div class="form-group">
              <label>会话标题</label>
              <input v-model="sessionConfig.title" placeholder="输入会话标题" />
            </div>
            <div class="form-group">
              <label>讨论主题</label>
              <textarea v-model="sessionConfig.topic" placeholder="输入讨论主题" rows="2" />
            </div>
            <div class="form-group">
              <label>全局限定</label>
              <textarea v-model="sessionConfig.global_constraint" placeholder="例如：每次发言不超过150字" rows="2" />
            </div>
            <div class="form-row">
              <div class="form-group">
                <label>最大轮数</label>
                <input type="number" v-model.number="sessionConfig.max_rounds" min="1" max="50" />
              </div>
              <div class="form-group">
                <label>回合延迟(秒)</label>
                <input type="number" v-model.number="delaySeconds" min="0" max="10" />
              </div>
            </div>
          </div>

          <!-- AI-A 配置 -->
          <div class="config-section">
            <h4>🅰️ AI-A 配置</h4>
            <div class="form-group">
              <label>名称</label>
              <input v-model="sessionConfig.agent_a.name" placeholder="AI名称" />
            </div>
            <div class="form-group">
              <label>角色设定</label>
              <textarea v-model="sessionConfig.agent_a.role" placeholder="描述AI的角色、背景、性格" rows="3" />
            </div>
            <div class="form-row">
              <div class="form-group">
                <label>语言风格</label>
                <select v-model="sessionConfig.agent_a.style.language_style">
                  <option value="professional">专业</option>
                  <option value="casual">口语化</option>
                  <option value="poetic">诗意</option>
                  <option value="academic">学术</option>
                  <option value="humorous">幽默</option>
                </select>
              </div>
              <div class="form-group">
                <label>知识水平</label>
                <select v-model="sessionConfig.agent_a.style.knowledge_level">
                  <option value="beginner">初学者</option>
                  <option value="intermediate">中级</option>
                  <option value="expert">专家</option>
                </select>
              </div>
            </div>
            <div class="form-group">
              <label>语气</label>
              <input v-model="sessionConfig.agent_a.style.tone" placeholder="例如：理性、热情" />
            </div>
            <div class="form-row">
              <div class="form-group">
                <label>AI模型</label>
                <select v-model="sessionConfig.agent_a.model">
                  <option v-for="m in availableModels" :key="m.id" :value="m.id">{{ m.provider }} / {{ m.name }}</option>
                </select>
              </div>
              <div class="form-group">
                <label>Temperature</label>
                <input type="range" v-model.number="sessionConfig.agent_a.temperature" min="0" max="2" step="0.1" />
                <span>{{ sessionConfig.agent_a.temperature }}</span>
              </div>
            </div>
          </div>

          <!-- AI-B 配置 -->
          <div class="config-section">
            <h4>🅱️ AI-B 配置</h4>
            <div class="form-group">
              <label>名称</label>
              <input v-model="sessionConfig.agent_b.name" placeholder="AI名称" />
            </div>
            <div class="form-group">
              <label>角色设定</label>
              <textarea v-model="sessionConfig.agent_b.role" placeholder="描述AI的角色、背景、性格" rows="3" />
            </div>
            <div class="form-row">
              <div class="form-group">
                <label>语言风格</label>
                <select v-model="sessionConfig.agent_b.style.language_style">
                  <option value="professional">专业</option>
                  <option value="casual">口语化</option>
                  <option value="poetic">诗意</option>
                  <option value="academic">学术</option>
                  <option value="humorous">幽默</option>
                </select>
              </div>
              <div class="form-group">
                <label>知识水平</label>
                <select v-model="sessionConfig.agent_b.style.knowledge_level">
                  <option value="beginner">初学者</option>
                  <option value="intermediate">中级</option>
                  <option value="expert">专家</option>
                </select>
              </div>
            </div>
            <div class="form-group">
              <label>语气</label>
              <input v-model="sessionConfig.agent_b.style.tone" placeholder="例如：理性、热情" />
            </div>
            <div class="form-row">
              <div class="form-group">
                <label>AI模型</label>
                <select v-model="sessionConfig.agent_b.model">
                  <option v-for="m in availableModels" :key="m.id" :value="m.id">{{ m.provider }} / {{ m.name }}</option>
                </select>
              </div>
              <div class="form-group">
                <label>Temperature</label>
                <input type="range" v-model.number="sessionConfig.agent_b.temperature" min="0" max="2" step="0.1" />
                <span>{{ sessionConfig.agent_b.temperature }}</span>
              </div>
            </div>
          </div>

          <!-- Additional Agents (Multi-agent mode) -->
          <div v-if="sessionConfig.agents.length > 0" class="config-section">
            <h4>🎭 Additional Agents</h4>
            <div v-for="(agent, index) in sessionConfig.agents" :key="index" class="agent-card">
              <div class="agent-card-header">
                <span :style="{ color: agentColors[index + 2] }">● Agent {{ String.fromCharCode(67 + index) }}</span>
                <button class="remove-agent-btn" @click="removeAgent(index)">✕</button>
              </div>
              <div class="form-group">
                <label>名称</label>
                <input v-model="agent.name" placeholder="AI名称" />
              </div>
              <div class="form-group">
                <label>角色设定</label>
                <textarea v-model="agent.role" placeholder="描述AI的角色、背景、性格" rows="2" />
              </div>
              <div class="form-row">
                <div class="form-group">
                  <label>AI模型</label>
                  <select v-model="agent.model">
                    <option v-for="m in availableModels" :key="m.id" :value="m.id">{{ m.provider }} / {{ m.name }}</option>
                  </select>
                </div>
                <div class="form-group">
                  <label>Temperature</label>
                  <input type="range" v-model.number="agent.temperature" min="0" max="2" step="0.1" />
                  <span>{{ agent.temperature }}</span>
                </div>
              </div>
            </div>
          </div>
          
          <div class="add-agent-section">
            <button class="add-agent-btn" @click="addAgent">
              ➕ Add Agent (Multi-Agent Mode)
            </button>
            <p class="add-agent-hint" v-if="sessionConfig.agents.length === 0">
              Enable multi-agent mode with 3+ AI participants
            </p>
          </div>

          <!-- 终止条件 -->
          <div class="config-section">
            <h4>终止条件</h4>
            <div class="form-group">
              <label>终止类型</label>
              <select v-model="sessionConfig.termination_config.type">
                <option value="fixed_rounds">固定轮数</option>
                <option value="open_ended">开放式（手动停止）</option>
                <option value="keyword">关键词触发</option>
              </select>
            </div>
            <div class="form-group" v-if="sessionConfig.termination_config.type === 'keyword'">
              <label>终止关键词（逗号分隔）</label>
              <input v-model="terminationKeywords" placeholder="结束,总结,到此为止" />
            </div>
            <div class="form-row">
              <div class="form-group">
                <label>相似度阈值</label>
                <input type="range" v-model.number="sessionConfig.termination_config.similarity_threshold" min="0" max="1" step="0.05" />
                <span>{{ sessionConfig.termination_config.similarity_threshold }}</span>
              </div>
              <div class="form-group">
                <label>连续相似轮数</label>
                <input type="number" v-model.number="sessionConfig.termination_config.consecutive_similar_rounds" min="2" max="5" />
              </div>
            </div>
          </div>
        </div>

        <div class="panel-footer">
          <button class="btn btn-secondary" @click="resetConfig">重置</button>
          <button class="btn btn-primary" @click="saveConfig">保存配置</button>
        </div>
      </div>

      <!-- 聊天区域 -->
      <div class="chat-area" :class="{ 'with-monitor': showMonitor }">
        <!-- 控制栏 -->
        <div class="control-bar">
          <div class="status-info">
            <span class="status-badge" :class="statusClass">{{ statusText }}</span>
            <span class="round-info">第 {{ currentRound }}/{{ maxRounds }} 轮</span>
            <span class="token-info" v-if="tokenUsage.total > 0">
              Tokens: {{ tokenUsage.total }}
            </span>
          </div>
          <div class="view-switcher">
            <el-button-group>
              <el-button :type="viewMode === 'list' ? 'primary' : 'default'" size="small" @click="viewMode = 'list'">
                <el-icon><List /></el-icon> List
              </el-button>
              <el-button :type="viewMode === 'timeline' ? 'primary' : 'default'" size="small" @click="viewMode = 'timeline'">
                <el-icon><DataLine /></el-icon> Timeline
              </el-button>
              <el-button :type="viewMode === 'split' ? 'primary' : 'default'" size="small" @click="viewMode = 'split'">
                <el-icon><Grid /></el-icon> Split
              </el-button>
            </el-button-group>
          </div>
          <div class="control-buttons">
            <button 
              class="btn btn-success" 
              @click="startSession" 
              v-if="!isRunning && !isPaused"
              :disabled="!canStart"
            >
              ▶️ 开始
            </button>
            <button 
              class="btn btn-warning" 
              @click="pauseSession" 
              v-if="isRunning"
            >
              ⏸️ 暂停
            </button>
            <button 
              class="btn btn-success" 
              @click="resumeSession" 
              v-if="isPaused"
            >
              ▶️ 继续
            </button>
            <button 
              class="btn btn-danger" 
              @click="stopSession" 
              v-if="isRunning || isPaused"
            >
              ⏹️ 停止
            </button>
            <button class="btn btn-secondary" @click="showMonitor = !showMonitor">
              📊 {{ showMonitor ? '隐藏' : '显示' }}监控
            </button>
            <el-dropdown @command="exportSession" trigger="click" :disabled="!sessionId">
              <button class="btn btn-secondary" :disabled="!sessionId">
                📥 导出 <el-icon style="vertical-align: middle;"><ArrowDown /></el-icon>
              </button>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item command="markdown">Markdown (.md)</el-dropdown-item>
                  <el-dropdown-item command="json">JSON (.json)</el-dropdown-item>
                  <el-dropdown-item command="text">Plain Text (.txt)</el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </div>
        </div>

        <!-- 导演指令输入 -->
        <div class="director-panel" v-if="isRunning || isPaused">
          <div class="director-input">
            <select v-model="directorTarget">
              <option value="agent_a">对 AI-A</option>
              <option value="agent_b">对 AI-B</option>
              <option value="both">对双方</option>
            </select>
            <input 
              v-model="directorCommand" 
              placeholder="输入导演指令，例如：请从现在开始反驳对方的观点" 
              @keyup.enter="sendDirectorCommand"
            />
            <button class="btn btn-primary" @click="sendDirectorCommand">发送指令</button>
          </div>
        </div>

        <!-- 会话结束提示横幅 -->
        <div v-if="sessionEndMessage" class="session-end-banner">
          <span>✅ {{ sessionEndMessage }}</span>
          <button class="btn-dismiss" @click="sessionEndMessage = ''">×</button>
        </div>

        <!-- 消息列表 - List View -->
        <div class="message-list" ref="messageList" v-show="viewMode === 'list'">
          <div 
            v-for="(message, index) in messages" 
            :key="index"
            class="message"
            :class="{ 
              'agent-a': message.agent_id === 'agent_a', 
              'agent-b': message.agent_id === 'agent_b',
              'tool-call': message.message_type === 'tool_call',
              'system-message': message.agent_id === 'system'
            }"
          >
            <div class="message-header">
              <span class="agent-icon">{{ message.agent_id === 'agent_a' ? '🅰️' : message.agent_id === 'agent_b' ? '🅱️' : '⚙️' }}</span>
              <span class="agent-name">{{ message.agent_name }}</span>
              <span class="message-round">第{{ message.round }}轮</span>
              <span class="message-time">{{ formatTime(message.timestamp) }}</span>
            </div>
            <div class="message-content" v-html="formatMessageContent(message.content)"></div>
            <div class="message-meta" v-if="message.tokens > 0">
              <span>Tokens: {{ message.tokens }}</span>
              <span v-if="message.latency_ms > 0">响应: {{ message.latency_ms }}ms</span>
            </div>
            
            <!-- 工具调用显示 -->
            <div class="tool-calls" v-if="message.tool_calls && Object.keys(message.tool_calls).length > 0">
              <div class="tool-call-item" v-for="(args, name) in message.tool_calls" :key="name">
                <span class="tool-name">🔧 {{ name }}</span>
                <pre class="tool-args">{{ JSON.stringify(args, null, 2) }}</pre>
              </div>
            </div>
          </div>
          
          <!-- 输入中提示 -->
          <div class="typing-indicator" v-if="isRunning && messages.length > 0">
            <span class="dot"></span>
            <span class="dot"></span>
            <span class="dot"></span>
          </div>
        </div>

        <!-- Timeline View -->
        <div class="timeline-view" v-show="viewMode === 'timeline'">
          <div class="timeline-container">
            <div class="timeline-line"></div>
            <div 
              v-for="(message, index) in messages" 
              :key="index"
              class="timeline-item"
              :class="message.agent_id"
            >
              <div class="timeline-marker" :class="message.agent_id"></div>
              <div class="timeline-card">
                <div class="timeline-header">
                  <span class="timeline-agent">
                    {{ message.agent_id === 'agent_a' ? '🅰️' : message.agent_id === 'agent_b' ? '🅱️' : '⚙️' }}
                    {{ message.agent_name }}
                  </span>
                  <span class="timeline-round">R{{ message.round }}</span>
                </div>
                <div class="timeline-content" v-html="formatMessageContent(message.content)"></div>
                <div class="timeline-footer">
                  <span v-if="message.tokens > 0">{{ message.tokens }} tokens</span>
                  <span v-if="message.latency_ms > 0">{{ message.latency_ms }}ms</span>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- Split View -->
        <div class="split-view" v-show="viewMode === 'split'">
          <div class="split-column agent-a-column">
            <div class="column-header">🅰️ {{ sessionConfig.agent_a?.name || 'AI-A' }}</div>
            <div class="column-messages">
              <div 
                v-for="(message, index) in agentAMessages" 
                :key="index"
                class="split-message"
              >
                <div class="split-round">R{{ message.round }}</div>
                <div class="split-content" v-html="formatMessageContent(message.content)"></div>
                <div class="split-meta" v-if="message.tokens > 0">
                  {{ message.tokens }} tokens
                </div>
              </div>
            </div>
          </div>
          <div class="split-divider">
            <div class="round-markers">
              <div v-for="n in maxRounds" :key="n" class="round-marker" :class="{ active: n <= currentRound }">
                {{ n }}
              </div>
            </div>
          </div>
          <div class="split-column agent-b-column">
            <div class="column-header">🅱️ {{ sessionConfig.agent_b?.name || 'AI-B' }}</div>
            <div class="column-messages">
              <div 
                v-for="(message, index) in agentBMessages" 
                :key="index"
                class="split-message"
              >
                <div class="split-round">R{{ message.round }}</div>
                <div class="split-content" v-html="formatMessageContent(message.content)"></div>
                <div class="split-meta" v-if="message.tokens > 0">
                  {{ message.tokens }} tokens
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- 空状态 -->
        <div class="empty-state" v-if="messages.length === 0">
          <div class="empty-icon">🤖💬🤖</div>
          <p>配置好参数后点击"开始"启动 AI-AI 对话</p>
          <button class="btn btn-primary" @click="showConfig = true">配置会话</button>
        </div>
      </div>

      <!-- 监控面板 -->
      <div class="monitor-panel" v-if="showMonitor">
        <div class="panel-header">
          <h4>实时监控</h4>
        </div>
        <div class="panel-body">
          <!-- Token 使用 -->
          <div class="monitor-section">
            <h5>Token 使用</h5>
            <div class="token-bar">
              <div class="token-segment agent-a" :style="{ width: tokenPercentA + '%' }">
                A: {{ tokenUsage.agent_a }}
              </div>
              <div class="token-segment agent-b" :style="{ width: tokenPercentB + '%' }">
                B: {{ tokenUsage.agent_b }}
              </div>
            </div>
            <div class="token-detail">
              <div>AI-A: {{ tokenUsage.agent_a_input }} in / {{ tokenUsage.agent_a_output }} out</div>
              <div>AI-B: {{ tokenUsage.agent_b_input }} in / {{ tokenUsage.agent_b_output }} out</div>
            </div>
          </div>
      
          <!-- 对话流程 -->
          <div class="monitor-section">
            <h5>对话流程</h5>
            <div class="flow-graph">
              <div 
                v-for="n in maxRounds" 
                :key="n"
                class="round-node"
                :class="{ 
                  'completed': n < currentRound, 
                  'current': n === currentRound,
                  'pending': n > currentRound 
                }"
              >
                {{ n }}
              </div>
            </div>
          </div>
      
          <!-- 统计信息 -->
          <div class="monitor-section">
            <h5>统计信息</h5>
            <div class="stats-grid">
              <div class="stat-item">
                <span class="stat-label">总消息</span>
                <span class="stat-value">{{ messages.length }}</span>
              </div>
              <div class="stat-item">
                <span class="stat-label">运行时间</span>
                <span class="stat-value">{{ runningTime }}</span>
              </div>
              <div class="stat-item">
                <span class="stat-label">平均延迟</span>
                <span class="stat-value">{{ avgLatency }}ms</span>
              </div>
            </div>
          </div>
      
          <!-- 对话评估 -->
          <div class="monitor-section" v-if="evaluation || evaluationLoading">
            <h5>
              对话评估
              <button
                v-if="!evaluationLoading && lastTerminatedSessionId"
                class="btn-refresh-eval"
                @click="refreshEvaluation"
                title="重新加载评估"
              >🔄</button>
            </h5>
            <div v-if="evaluationLoading" class="evaluation-loading">
              <span>⏳</span> 正在生成评估报告... （第 {{ evaluationAttempt }}/25 次检测）
            </div>
            <template v-if="evaluation">
            <div class="stats-grid">
              <div class="stat-item">
                <span class="stat-label">总体评分</span>
                <span class="stat-value">{{ evaluation.overall_score?.toFixed(1) }}</span>
              </div>
              <div class="stat-item">
                <span class="stat-label">主题紧扣度</span>
                <span class="stat-value">{{ evaluation.topic_adherence?.score?.toFixed(1) }}</span>
              </div>
              <div class="stat-item">
                <span class="stat-label">角色一致性</span>
                <span class="stat-value">{{ evaluation.role_consistency?.overall?.toFixed(1) }}</span>
              </div>
              <div class="stat-item">
                <span class="stat-label">逻辑连贯性</span>
                <span class="stat-value">{{ evaluation.logical_coherence?.score?.toFixed(1) }}</span>
              </div>
              <div class="stat-item">
                <span class="stat-label">精彩程度</span>
                <span class="stat-value">{{ evaluation.engagement?.score?.toFixed(1) }}</span>
              </div>
            </div>
            <div class="evaluation-summary" v-if="evaluation.summary">
              <div class="stat-label">总结</div>
              <div class="stat-value">{{ evaluation.summary }}</div>
            </div>
            <div class="evaluation-highlights" v-if="evaluation.highlights && evaluation.highlights.length">
              <div class="stat-label">亮点片段</div>
              <ul>
                <li v-for="(h, idx) in evaluation.highlights" :key="idx">
                  <strong>第{{ h.round }}轮 - {{ h.agent_name }}：</strong>{{ h.content }}
                </li>
              </ul>
            </div>
            </template>
          </div>
        </div>
      </div>
    </div>

    <!-- 模板选择弹窗 -->
    <div class="modal" v-if="showTemplates" @click.self="showTemplates = false">
      <div class="modal-content modal-content-wide">
        <div class="modal-header">
          <h3>📋 模板库</h3>
          <div class="modal-header-actions">
            <button class="btn btn-primary btn-sm" @click="openCreateTemplate">➕ 新建模板</button>
            <button class="close-btn" @click="showTemplates = false">×</button>
          </div>
        </div>
        <div class="modal-body">
          <div class="template-grid">
            <div 
              v-for="tmpl in templates" 
              :key="tmpl.id"
              class="template-card"
            >
              <div class="template-card-body" @click="applyTemplate(tmpl)">
                <div class="template-icon">{{ tmpl.icon || '🤖' }}</div>
                <div class="template-info">
                  <h4>{{ tmpl.name }}</h4>
                  <p>{{ tmpl.description }}</p>
                  <span class="template-badge builtin" v-if="tmpl.is_builtin">内置</span>
                  <span class="template-badge custom" v-else>自定义</span>
                </div>
              </div>
              <div class="template-card-actions">
                <button class="tpl-btn" @click.stop="cloneTemplateAction(tmpl.id)" title="克隆此模板">📋</button>
                <button class="tpl-btn" @click.stop="openEditTemplate(tmpl)" v-if="!tmpl.is_builtin" title="编辑">✏️</button>
                <button class="tpl-btn tpl-btn-danger" @click.stop="deleteTemplateConfirm(tmpl.id)" v-if="!tmpl.is_builtin" title="删除">🗑️</button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 模板编辑器弹窗 -->
    <div class="modal" v-if="showTemplateEditor" @click.self="showTemplateEditor = false">
      <div class="modal-content modal-content-editor">
        <div class="modal-header">
          <h3>{{ templateEditorMode === 'create' ? '➕ 新建模板' : '✏️ 编辑模板' }}</h3>
          <button class="close-btn" @click="showTemplateEditor = false">×</button>
        </div>
        <div class="modal-body editor-body">
          <!-- 基本信息 -->
          <div class="editor-section">
            <h4 class="editor-section-title">基本信息</h4>
            <div class="editor-row">
              <div class="editor-field">
                <label>模板名称 *</label>
                <input v-model="templateForm.name" placeholder="例如：科技伦理辩论" />
              </div>
              <div class="editor-field editor-field-sm">
                <label>图标 (emoji)</label>
                <input v-model="templateForm.icon" placeholder="🤖" maxlength="4" />
              </div>
              <div class="editor-field editor-field-sm">
                <label>分类</label>
                <select v-model="templateForm.category">
                  <option value="debate">辩论</option>
                  <option value="interview">面试</option>
                  <option value="collaboration">协作</option>
                  <option value="roleplay">角色扮演</option>
                  <option value="creative">创意</option>
                  <option value="education">教育</option>
                  <option value="custom">自定义</option>
                </select>
              </div>
            </div>
            <div class="editor-field">
              <label>描述</label>
              <textarea v-model="templateForm.description" rows="2" placeholder="简单描述这个模板的用途" />
            </div>
          </div>

          <!-- 对话设置 -->
          <div class="editor-section">
            <h4 class="editor-section-title">对话设置</h4>
            <div class="editor-field">
              <label>讨论主题 *</label>
              <input v-model="templateForm.config.topic" placeholder="例如：人工智能发展应该优先考虑效率还是安全？" />
            </div>
            <div class="editor-field">
              <label>全局约束</label>
              <textarea v-model="templateForm.config.global_constraint" rows="2" placeholder="例如：每次发言不超过150字，禁止使用英文术语" />
            </div>
            <div class="editor-row">
              <div class="editor-field editor-field-sm">
                <label>最大轮数</label>
                <input type="number" v-model.number="templateForm.config.max_rounds" min="1" max="50" />
              </div>
              <div class="editor-field editor-field-sm">
                <label>终止方式</label>
                <select v-model="templateForm.config.termination_config.type">
                  <option value="fixed_rounds">固定轮数</option>
                  <option value="keyword">关键词检测</option>
                  <option value="open_ended">开放式</option>
                </select>
              </div>
              <div class="editor-field editor-field-sm" v-if="templateForm.config.termination_config.type === 'keyword'">
                <label>终止关键词</label>
                <input v-model="templateForm.config.termination_config.keywords_str" placeholder="结束,总结,到此为止" />
              </div>
            </div>
          </div>

          <!-- AI-A 配置 -->
          <div class="editor-section">
            <h4 class="editor-section-title">🅰️ AI-A 配置</h4>
            <div class="editor-row">
              <div class="editor-field">
                <label>名称</label>
                <input v-model="templateForm.config.agent_a.name" placeholder="例如：李博士" />
              </div>
              <div class="editor-field editor-field-sm">
                <label>模型</label>
                <select v-model="templateForm.config.agent_a.model">
                  <option value="">默认</option>
                  <option v-for="m in availableModels" :key="m.id" :value="m.id">{{ m.name }}</option>
                </select>
              </div>
              <div class="editor-field editor-field-sm">
                <label>Temperature</label>
                <input type="number" v-model.number="templateForm.config.agent_a.temperature" min="0" max="2" step="0.1" />
              </div>
              <div class="editor-field editor-field-sm">
                <label>Max Tokens</label>
                <input type="number" v-model.number="templateForm.config.agent_a.max_tokens" min="50" max="4000" step="50" />
              </div>
            </div>
            <div class="editor-field">
              <label>角色设定</label>
              <textarea v-model="templateForm.config.agent_a.role" rows="3" placeholder="描述AI-A的角色、背景、性格..." />
            </div>
            <div class="editor-row">
              <div class="editor-field editor-field-sm">
                <label>语言风格</label>
                <select v-model="templateForm.config.agent_a.style.language_style">
                  <option value="professional">专业</option>
                  <option value="casual">口语化</option>
                  <option value="poetic">诗意</option>
                  <option value="academic">学术</option>
                  <option value="humorous">幽默</option>
                </select>
              </div>
              <div class="editor-field editor-field-sm">
                <label>知识水平</label>
                <select v-model="templateForm.config.agent_a.style.knowledge_level">
                  <option value="beginner">初学者</option>
                  <option value="intermediate">中级</option>
                  <option value="expert">专家</option>
                </select>
              </div>
              <div class="editor-field">
                <label>语气</label>
                <input v-model="templateForm.config.agent_a.style.tone" placeholder="例如：理性、审慎" />
              </div>
            </div>
          </div>

          <!-- AI-B 配置 -->
          <div class="editor-section">
            <h4 class="editor-section-title">🅱️ AI-B 配置</h4>
            <div class="editor-row">
              <div class="editor-field">
                <label>名称</label>
                <input v-model="templateForm.config.agent_b.name" placeholder="例如：张总" />
              </div>
              <div class="editor-field editor-field-sm">
                <label>模型</label>
                <select v-model="templateForm.config.agent_b.model">
                  <option value="">默认</option>
                  <option v-for="m in availableModels" :key="m.id" :value="m.id">{{ m.name }}</option>
                </select>
              </div>
              <div class="editor-field editor-field-sm">
                <label>Temperature</label>
                <input type="number" v-model.number="templateForm.config.agent_b.temperature" min="0" max="2" step="0.1" />
              </div>
              <div class="editor-field editor-field-sm">
                <label>Max Tokens</label>
                <input type="number" v-model.number="templateForm.config.agent_b.max_tokens" min="50" max="4000" step="50" />
              </div>
            </div>
            <div class="editor-field">
              <label>角色设定</label>
              <textarea v-model="templateForm.config.agent_b.role" rows="3" placeholder="描述AI-B的角色、背景、性格..." />
            </div>
            <div class="editor-row">
              <div class="editor-field editor-field-sm">
                <label>语言风格</label>
                <select v-model="templateForm.config.agent_b.style.language_style">
                  <option value="professional">专业</option>
                  <option value="casual">口语化</option>
                  <option value="poetic">诗意</option>
                  <option value="academic">学术</option>
                  <option value="humorous">幽默</option>
                </select>
              </div>
              <div class="editor-field editor-field-sm">
                <label>知识水平</label>
                <select v-model="templateForm.config.agent_b.style.knowledge_level">
                  <option value="beginner">初学者</option>
                  <option value="intermediate">中级</option>
                  <option value="expert">专家</option>
                </select>
              </div>
              <div class="editor-field">
                <label>语气</label>
                <input v-model="templateForm.config.agent_b.style.tone" placeholder="例如：热情、直接" />
              </div>
            </div>
          </div>

          <!-- 保存按钮 -->
          <div class="editor-footer">
            <button class="btn btn-secondary" @click="showTemplateEditor = false">取消</button>
            <button class="btn btn-primary" @click="saveTemplateEditor" :disabled="!templateForm.name || !templateForm.config.topic">
              {{ templateEditorMode === 'create' ? '创建模板' : '保存修改' }}
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, nextTick, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import { Fold, ArrowDown, List, DataLine, Grid } from '@element-plus/icons-vue'
import { loadTemplates } from '@/config/aiChatTemplates'

const route = useRoute()
const authStore = useAuthStore()

// 状态
const showConfig = ref(true)
const showMonitor = ref(false)
const showTemplates = ref(false)
const historyCollapsed = ref(false)
const sessionHistory = ref([])
const isRunning = ref(false)
const isPaused = ref(false)
const currentRound = ref(0)
const maxRounds = ref(10)
const messages = ref([])
const sessionId = ref(null)
const ws = ref(null)
const messageList = ref(null)
const directorCommand = ref('')
const directorTarget = ref('agent_a')
const delaySeconds = ref(1)
const startTime = ref(null)
const runningTime = ref('00:00')
const timer = ref(null)
const viewMode = ref('list') // list, timeline, split

// Computed properties for split view
const agentAMessages = computed(() => messages.value.filter(m => m.agent_id === 'agent_a'))
const agentBMessages = computed(() => messages.value.filter(m => m.agent_id === 'agent_b'))

// Token 使用
const tokenUsage = ref({
  agent_a: 0,
  agent_b: 0,
  agent_a_input: 0,
  agent_a_output: 0,
  agent_b_input: 0,
  agent_b_output: 0,
  total: 0
})

// 可用模型
const availableModels = ref([])

// 可用MCP工具
const availableMCPTools = ref([])

// 对话评估结果
const evaluation = ref(null)
const evaluationLoading = ref(false)
const evaluationAttempt = ref(0)
const EVAL_SESSION_KEY = 'ai_chat_pending_eval_session'
const lastTerminatedSessionId = ref(localStorage.getItem(EVAL_SESSION_KEY) || null)
const sessionEndMessage = ref('')  // 替代 alert() 的非阻塞提示

// 模板编辑器状态
const showTemplateEditor = ref(false)
const templateEditorMode = ref('create')  // 'create' | 'edit'
const editingTemplateId = ref(null)

const defaultTemplateForm = () => ({
  name: '',
  description: '',
  category: 'custom',
  icon: '🤖',
  config: {
    topic: '',
    global_constraint: '',
    max_rounds: 10,
    termination_config: {
      type: 'fixed_rounds',
      max_rounds: 10,
      keywords_str: '',
      similarity_threshold: 0.85,
      consecutive_similar_rounds: 3
    },
    agent_a: {
      name: '',
      role: '',
      model: '',
      temperature: 0.7,
      max_tokens: 300,
      style: { language_style: 'professional', knowledge_level: 'expert', tone: '' }
    },
    agent_b: {
      name: '',
      role: '',
      model: '',
      temperature: 0.7,
      max_tokens: 300,
      style: { language_style: 'professional', knowledge_level: 'expert', tone: '' }
    }
  }
})
const templateForm = ref(defaultTemplateForm())

// 终止关键词
const terminationKeywords = ref('结束,总结,到此为止')

// 会话配置
const sessionConfig = ref({
  title: 'AI-AI 对话',
  topic: '讨论人工智能的伦理问题',
  global_constraint: '每次发言不超过150字',
  max_rounds: 10,
  termination_config: {
    type: 'fixed_rounds',
    max_rounds: 10,
    keywords: [],
    similarity_threshold: 0.85,
    consecutive_similar_rounds: 3
  },
  agent_a: {
    name: '李博士',
    role: '你是一位严谨的AI安全研究员，专注于人工智能的伦理和安全问题。',
    style: {
      language_style: 'professional',
      knowledge_level: 'expert',
      tone: '理性、审慎'
    },
    model: 'deepseek-chat',
    temperature: 0.7,
    max_tokens: 300,
    allowed_tools: ['search/web_search']  // 启用搜索工具
  },
  agent_b: {
    name: '张总',
    role: '你是一位科技创业公司的CEO，热衷于技术创新和商业化。',
    style: {
      language_style: 'casual',
      knowledge_level: 'expert',
      tone: '热情、直接'
    },
    model: 'deepseek-chat',
    temperature: 0.8,
    max_tokens: 300,
    allowed_tools: ['search/web_search']  // 启用搜索工具
  },
  // Multi-agent support (3+ agents)
  agents: []
})

// Multi-agent helpers
const isMultiAgentMode = computed(() => sessionConfig.value.agents.length > 0)

const addAgent = () => {
  const agentIndex = sessionConfig.value.agents.length + 3 // Start from agent_c
  sessionConfig.value.agents.push({
    name: `Agent ${String.fromCharCode(65 + agentIndex - 1)}`,
    role: '请描述这个AI的角色和背景',
    style: {
      language_style: 'professional',
      knowledge_level: 'intermediate',
      tone: ''
    },
    model: 'deepseek-chat',
    temperature: 0.7,
    max_tokens: 300,
    allowed_tools: []
  })
}

const removeAgent = (index) => {
  sessionConfig.value.agents.splice(index, 1)
}

// Agent colors for multi-agent mode
const agentColors = ['#6366f1', '#ec4899', '#14b8a6', '#f59e0b', '#8b5cf6', '#ef4444', '#10b981', '#f97316']

const getAgentColor = (agentId) => {
  if (agentId === 'agent_a') return agentColors[0]
  if (agentId === 'agent_b') return agentColors[1]
  const index = parseInt(agentId.replace('agent_', ''), 10)
  return agentColors[index % agentColors.length]
}

// 模板列表
// Load templates from configuration
const templates = ref([])

// 计算属性
const statusText = computed(() => {
  if (isRunning.value) return '运行中'
  if (isPaused.value) return '已暂停'
  return '未开始'
})

const statusClass = computed(() => {
  if (isRunning.value) return 'running'
  if (isPaused.value) return 'paused'
  return 'pending'
})

const canStart = computed(() => {
  return sessionConfig.value.title && 
         sessionConfig.value.topic && 
         sessionConfig.value.agent_a.name && 
         sessionConfig.value.agent_b.name
})

const tokenPercentA = computed(() => {
  if (tokenUsage.value.total === 0) return 50
  return (tokenUsage.value.agent_a / tokenUsage.value.total) * 100
})

const tokenPercentB = computed(() => {
  if (tokenUsage.value.total === 0) return 50
  return (tokenUsage.value.agent_b / tokenUsage.value.total) * 100
})

const avgLatency = computed(() => {
  if (messages.value.length === 0) return 0
  const total = messages.value.reduce((sum, m) => sum + (m.latency_ms || 0), 0)
  return Math.round(total / messages.value.length)
})

// 方法
const formatTime = (timestamp) => {
  if (!timestamp) return ''
  const date = new Date(timestamp)
  return date.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit', second: '2-digit' })
}

// 格式化消息内容，为搜索结果添加特殊样式
const formatMessageContent = (content) => {
  if (!content) return ''
  
  // 先检测并标记搜索结果部分（在转义之前）
  const searchPatterns = [
    { pattern: /=== 最新搜索结果[\s\S]*?=== 搜索结果结束 ===/g, marker: '###SEARCH_START###' },
    { pattern: /=== WEB SEARCH RESULTS ===[\s\S]*?=== END OF SEARCH RESULTS ===/g, marker: '###SEARCH_START###' }
  ]
  
  let processed = content
  const searchBlocks = []
  
  searchPatterns.forEach(({ pattern, marker }) => {
    processed = processed.replace(pattern, (match) => {
      searchBlocks.push(match)
      return `${marker}${searchBlocks.length - 1}###SEARCH_END###`
    })
  })
  
  // 转义 HTML 特殊字符
  let escaped = processed
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#039;')
  
  // 保留换行
  escaped = escaped.replace(/\n/g, '<br>')
  
  // 恢复搜索结果块并包装样式
  searchBlocks.forEach((block, index) => {
    const escapedBlock = block
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;')
      .replace(/'/g, '&#039;')
      .replace(/\n/g, '<br>')
    
    escaped = escaped.replace(
      `###SEARCH_START###${index}###SEARCH_END###`,
      `<div class="search-results-block">${escapedBlock}</div>`
    )
  })
  
  return escaped
}

const saveConfig = () => {
  // 解析终止关键词
  if (terminationKeywords.value) {
    sessionConfig.value.termination_config.keywords = terminationKeywords.value.split(',').map(k => k.trim())
  }
  showConfig.value = false
}

const resetConfig = () => {
  sessionConfig.value = {
    title: 'AI-AI 对话',
    topic: '讨论人工智能的伦理问题',
    global_constraint: '',
    max_rounds: 10,
    termination_config: {
      type: 'fixed_rounds',
      max_rounds: 10,
      keywords: [],
      similarity_threshold: 0.85,
      consecutive_similar_rounds: 3
    },
    agent_a: {
      name: 'AI-A',
      role: '',
      style: { language_style: 'professional', knowledge_level: 'expert', tone: '' },
      model: 'deepseek-chat',
      temperature: 0.7,
      max_tokens: 300,
      allowed_tools: ['search/web_search']
    },
    agent_b: {
      name: 'AI-B',
      role: '',
      style: { language_style: 'casual', knowledge_level: 'expert', tone: '' },
      model: 'deepseek-chat',
      temperature: 0.8,
      max_tokens: 300,
      allowed_tools: ['search/web_search']
    }
  }
}

const applyTemplate = (tmpl) => {
  Object.assign(sessionConfig.value, tmpl.config)
  showTemplates.value = false
}

// ── 模板编辑器函数 ──────────────────────────────────────────────────────

const fetchTemplatesFromAPI = async () => {
  try {
    const response = await fetch('/api/v1/ai-chat/templates', {
      headers: { 'Authorization': `Bearer ${authStore.token}` }
    })
    if (response.ok) {
      templates.value = await response.json()
    } else {
      templates.value = loadTemplates()
    }
  } catch {
    templates.value = loadTemplates()
  }
}

const openCreateTemplate = () => {
  templateEditorMode.value = 'create'
  editingTemplateId.value = null
  templateForm.value = defaultTemplateForm()
  showTemplateEditor.value = true
}

const openEditTemplate = (tmpl) => {
  templateEditorMode.value = 'edit'
  editingTemplateId.value = tmpl.id
  const cfg = tmpl.config || {}
  const agentA = cfg.agent_a || {}
  const agentB = cfg.agent_b || {}
  const termCfg = cfg.termination_config || {}
  templateForm.value = {
    name: tmpl.name || '',
    description: tmpl.description || '',
    category: tmpl.category || 'custom',
    icon: tmpl.icon || '🤖',
    config: {
      topic: cfg.topic || '',
      global_constraint: cfg.global_constraint || '',
      max_rounds: cfg.max_rounds || 10,
      termination_config: {
        type: termCfg.type || 'fixed_rounds',
        max_rounds: termCfg.max_rounds || 10,
        keywords_str: (termCfg.keywords || []).join(','),
        similarity_threshold: termCfg.similarity_threshold || 0.85,
        consecutive_similar_rounds: termCfg.consecutive_similar_rounds || 3
      },
      agent_a: {
        name: agentA.name || '',
        role: agentA.role || '',
        model: agentA.model || '',
        temperature: agentA.temperature ?? 0.7,
        max_tokens: agentA.max_tokens || 300,
        style: {
          language_style: agentA.style?.language_style || 'professional',
          knowledge_level: agentA.style?.knowledge_level || 'expert',
          tone: agentA.style?.tone || ''
        }
      },
      agent_b: {
        name: agentB.name || '',
        role: agentB.role || '',
        model: agentB.model || '',
        temperature: agentB.temperature ?? 0.7,
        max_tokens: agentB.max_tokens || 300,
        style: {
          language_style: agentB.style?.language_style || 'professional',
          knowledge_level: agentB.style?.knowledge_level || 'expert',
          tone: agentB.style?.tone || ''
        }
      }
    }
  }
  showTemplateEditor.value = true
}

const saveTemplateEditor = async () => {
  const form = templateForm.value
  const termCfg = { ...form.config.termination_config }
  // Convert keywords_str to array, remove UI-only field
  const keywords = (termCfg.keywords_str || '').split(',').map(k => k.trim()).filter(k => k)
  delete termCfg.keywords_str
  termCfg.keywords = keywords

  const payload = {
    name: form.name,
    description: form.description,
    category: form.category,
    icon: form.icon,
    config: {
      topic: form.config.topic,
      global_constraint: form.config.global_constraint,
      max_rounds: form.config.max_rounds,
      termination_config: termCfg,
      agent_a: form.config.agent_a,
      agent_b: form.config.agent_b
    }
  }

  const isEdit = templateEditorMode.value === 'edit'
  const url = isEdit
    ? `/api/v1/ai-chat/templates/${editingTemplateId.value}`
    : '/api/v1/ai-chat/templates'
  const method = isEdit ? 'PUT' : 'POST'

  try {
    const response = await fetch(url, {
      method,
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${authStore.token}`
      },
      body: JSON.stringify(payload)
    })
    if (response.ok) {
      await fetchTemplatesFromAPI()
      showTemplateEditor.value = false
    } else {
      const err = await response.json()
      alert(err.error || '保存失败')
    }
  } catch (error) {
    alert('网络错误，请重试')
  }
}

const cloneTemplateAction = async (id) => {
  try {
    const response = await fetch(`/api/v1/ai-chat/templates/${id}/clone`, {
      method: 'POST',
      headers: { 'Authorization': `Bearer ${authStore.token}` }
    })
    if (response.ok) {
      await fetchTemplatesFromAPI()
    } else {
      const err = await response.json()
      alert(err.error || '克隆失败')
    }
  } catch {
    alert('网络错误，请重试')
  }
}

const deleteTemplateConfirm = async (id) => {
  if (!confirm('确认删除此模板？此操作不可撤销。')) return
  try {
    const response = await fetch(`/api/v1/ai-chat/templates/${id}`, {
      method: 'DELETE',
      headers: { 'Authorization': `Bearer ${authStore.token}` }
    })
    if (response.ok) {
      await fetchTemplatesFromAPI()
    } else {
      const err = await response.json()
      alert(err.error || '删除失败')
    }
  } catch {
    alert('网络错误，请重试')
  }
}

const createSession = async () => {
  try {
    const response = await fetch('/api/v1/ai-chat/sessions', {
      method: 'POST',
      headers: { 
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${authStore.token}`
      },
      body: JSON.stringify(sessionConfig.value)
    })
    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`)
    }
    const data = await response.json()
    sessionId.value = data.id
    maxRounds.value = data.max_rounds
    // Refresh history list
    fetchSessionHistory()
    return data
  } catch (error) {
    console.error('Failed to create session:', error)
    alert('创建会话失败: ' + error.message)
    throw error
  }
}

const startSession = async () => {
  try {
    if (!sessionId.value) {
      await createSession()
    }
    
    if (!sessionId.value) {
      throw new Error('Session ID is not available')
    }
    
    // Connect WebSocket first, then start session to avoid missing early messages
    await connectWebSocket()
    // Small delay to ensure subscription is registered before session starts
    await new Promise(resolve => setTimeout(resolve, 200))
    
    const response = await fetch(`/api/v1/ai-chat/sessions/${sessionId.value}/start`, { 
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${authStore.token}`
      }
    })
    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`)
    }
    isRunning.value = true
    isPaused.value = false
    startTime.value = Date.now()
    startTimer()
  } catch (error) {
    console.error('Failed to start session:', error)
    alert('启动会话失败: ' + error.message)
  }
}

const pauseSession = async () => {
  try {
    await fetch(`/api/v1/ai-chat/sessions/${sessionId.value}/pause`, { 
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${authStore.token}`
      }
    })
    isRunning.value = false
    isPaused.value = true
    stopTimer()
    ws.value?.close()
  } catch (error) {
    console.error('Failed to pause session:', error)
  }
}

const resumeSession = async () => {
  try {
    await connectWebSocket()
    await fetch(`/api/v1/ai-chat/sessions/${sessionId.value}/start`, { 
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${authStore.token}`
      }
    })
    isRunning.value = true
    isPaused.value = false
    startTimer()
  } catch (error) {
    console.error('Failed to resume session:', error)
  }
}

const stopSession = async () => {
  try {
    await fetch(`/api/v1/ai-chat/sessions/${sessionId.value}/stop`, { 
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${authStore.token}`
      }
    })
    isRunning.value = false
    isPaused.value = false
    stopTimer()
    ws.value?.close()
    // Reset session so next Start creates a new one
    sessionId.value = null
    messages.value = []
    currentRound.value = 0
    // Refresh history list
    fetchSessionHistory()
  } catch (error) {
    console.error('Failed to stop session:', error)
  }
}

const sendDirectorCommand = async () => {
  if (!directorCommand.value.trim()) return
  
  try {
    await fetch(`/api/v1/ai-chat/sessions/${sessionId.value}/director-command`, {
      method: 'POST',
      headers: { 
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${authStore.token}`
      },
      body: JSON.stringify({
        target_agent: directorTarget.value,
        command: directorCommand.value
      })
    })
    directorCommand.value = ''
  } catch (error) {
    console.error('Failed to send director command:', error)
  }
}

const exportSession = async (format) => {
  if (!sessionId.value) return
  const ext = format === 'json' ? 'json' : format === 'text' ? 'txt' : 'md'
  try {
    const res = await fetch(
      `/api/v1/ai-chat/sessions/${sessionId.value}/export?format=${format}`,
      { headers: { 'Authorization': `Bearer ${authStore.token}` } }
    )
    if (!res.ok) throw new Error(`Export failed: ${res.status}`)
    const blob = await res.blob()
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `ai-chat-${sessionId.value.slice(0, 8)}.${ext}`
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(url)
  } catch (error) {
    console.error('Export failed:', error)
  }
}

const connectWebSocket = () => {
  return new Promise((resolve, reject) => {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const wsUrl = `${protocol}//${window.location.host}/api/v1/ws/ai-chat/${sessionId.value}?token=${authStore.token}`
    ws.value = new WebSocket(wsUrl)
    
    ws.value.onopen = () => {
      console.log('WebSocket connected')
      resolve()
    }

    ws.value.onmessage = (event) => {
      const data = JSON.parse(event.data)
      handleWebSocketMessage(data)
    }
    
    ws.value.onclose = () => {
      console.log('WebSocket closed')
    }
    
    ws.value.onerror = (error) => {
      console.error('WebSocket error:', error)
      reject(error)
    }
  })
}

// 用于流式消息的临时存储
const streamingMessages = ref(new Map())

const handleWebSocketMessage = (event) => {
  console.log('[WS] Received event:', event.type, event.data)
  switch (event.type) {
    case 'message':
      messages.value.push(event.data)
      currentRound.value = event.data.round
      // Update token usage
      if (event.data.agent_id === 'agent_a') {
        tokenUsage.value.agent_a += event.data.tokens || 0
      } else {
        tokenUsage.value.agent_b += event.data.tokens || 0
      }
      tokenUsage.value.total = tokenUsage.value.agent_a + tokenUsage.value.agent_b
      nextTick(() => {
        messageList.value?.scrollTo({ top: messageList.value.scrollHeight, behavior: 'smooth' })
      })
      break
    case 'message_start':
      // 开始新的流式消息
      streamingMessages.value.set(event.data.agentId, {
        agentId: event.data.agentId,
        agentName: event.data.agentName,
        round: event.data.round,
        content: '',
        isStreaming: true
      })
      break
    case 'message_chunk':
      // 追加流式消息 chunk
      const streamingMsg = streamingMessages.value.get(event.data.agentId)
      if (streamingMsg) {
        streamingMsg.content += event.data.chunk
        // 更新或添加临时消息到列表
        const existingIndex = messages.value.findIndex(m => 
          m.agent_id === event.data.agentId && m.isStreaming
        )
        if (existingIndex >= 0) {
          messages.value[existingIndex].content = streamingMsg.content
        } else {
          messages.value.push({
            agent_id: event.data.agentId,
            agent_name: event.data.agentName,
            round: streamingMsg.round,
            content: streamingMsg.content,
            isStreaming: true,
            timestamp: new Date()
          })
        }
        nextTick(() => {
          messageList.value?.scrollTo({ top: messageList.value.scrollHeight, behavior: 'smooth' })
        })
      }
      break
    case 'message_complete':
      // 流式消息完成，替换为完整消息
      streamingMessages.value.delete(event.data.agentId)
      const completeIndex = messages.value.findIndex(m => 
        m.agent_id === event.data.agentId && m.isStreaming
      )
      if (completeIndex >= 0) {
        messages.value[completeIndex] = {
          agent_id: event.data.agentId,
          agent_name: event.data.agentName,
          round: event.data.round,
          content: event.data.content,
          tokens: event.data.tokens,
          isStreaming: false,
          timestamp: new Date()
        }
      }
      // Update token usage
      if (event.data.agentId === 'agent_a') {
        tokenUsage.value.agent_a += event.data.tokens || 0
      } else {
        tokenUsage.value.agent_b += event.data.tokens || 0
      }
      tokenUsage.value.total = tokenUsage.value.agent_a + tokenUsage.value.agent_b
      currentRound.value = event.data.round
      break
    case 'status':
      if (event.data.status === 'paused') {
        isRunning.value = false
        isPaused.value = true
      }
      break
    case 'termination': {
      isRunning.value = false
      isPaused.value = false
      stopTimer()
      ws.value?.close()
      // 立即开始轮询评估（必须在任何阻塞操作之前启动）
      const terminatedSessionId = event.data.sessionId || event.data.session_id || event.data.id
      lastTerminatedSessionId.value = terminatedSessionId
      localStorage.setItem(EVAL_SESSION_KEY, terminatedSessionId)
      fetchEvaluation(terminatedSessionId)
      // Reset session so next Start creates a new one
      sessionId.value = null
      // 用非阻塞的 in-page 提示替代 alert()
      sessionEndMessage.value = `对话已结束：${event.data.message}`
      break
    }
    case 'error':
      alert(`错误：${event.data.message}`)
      break
    case 'director_command_applied':
      // 导演指令已应用，显示系统提示
      messages.value.push({
        agent_id: 'system',
        agent_name: '系统',
        round: currentRound.value,
        content: `🎬 导演指令已发送至 ${event.data.targetAgent === 'agent_a' ? 'AI-A' : event.data.targetAgent === 'agent_b' ? 'AI-B' : '双方'}`,
        message_type: 'system',
        timestamp: new Date()
      })
      nextTick(() => {
        messageList.value?.scrollTo({ top: messageList.value.scrollHeight, behavior: 'smooth' })
      })
      break
  }
}

const startTimer = () => {
  timer.value = setInterval(() => {
    const elapsed = Math.floor((Date.now() - startTime.value) / 1000)
    const minutes = Math.floor(elapsed / 60).toString().padStart(2, '0')
    const seconds = (elapsed % 60).toString().padStart(2, '0')
    runningTime.value = `${minutes}:${seconds}`
  }, 1000)
}

const stopTimer = () => {
  clearInterval(timer.value)
}

// 加载可用模型
const fetchModels = async () => {
  try {
    const response = await fetch('/api/v1/ai-chat/models', {
      headers: {
        'Authorization': `Bearer ${authStore.token}`
      }
    })
    if (response.ok) {
      const models = await response.json()
      availableModels.value = models  // store full objects: { id, name, provider }
      // Set default model if available
      if (models.length > 0 && !sessionConfig.value.agent_a.model) {
        sessionConfig.value.agent_a.model = models[0].id
      }
      if (models.length > 0 && !sessionConfig.value.agent_b.model) {
        sessionConfig.value.agent_b.model = models[0].id
      }
    }
  } catch (error) {
    console.error('Failed to fetch models:', error)
  }
}

// 加载可用MCP工具
const fetchMCPTools = async () => {
  try {
    const response = await fetch('/api/v1/ai-chat/mcp-tools', {
      headers: {
        'Authorization': `Bearer ${authStore.token}`
      }
    })
    if (response.ok) {
      const tools = await response.json()
      availableMCPTools.value = tools
    }
  } catch (error) {
    console.error('Failed to fetch MCP tools:', error)
  }
}

// 轮询评估结果（评估需要 LLM 调用，可能需要数秒到数十秒）
const fetchEvaluation = async (id) => {
  if (!id) return
  const maxRetries = 25      // 最多重试 25 次
  const retryDelayMs = 3000  // 每次间隔 3 秒（最多等 75 秒）

  evaluationLoading.value = true
  evaluation.value = null
  evaluationAttempt.value = 0
  // 自动打开监控面板，让用户看到加载中的提示
  showMonitor.value = true

  for (let attempt = 1; attempt <= maxRetries; attempt++) {
    evaluationAttempt.value = attempt
    try {
      const response = await fetch(`/api/v1/ai-chat/sessions/${id}/evaluation`, {
        headers: {
          'Authorization': `Bearer ${authStore.token}`
        }
      })
      if (response.ok) {
        evaluation.value = await response.json()
        evaluationLoading.value = false
        // 评估成功加载，清除 localStorage 中的待处理记录
        localStorage.removeItem(EVAL_SESSION_KEY)
        console.log('[AI-Chat] Evaluation loaded on attempt', attempt)
        // 滚动到评估区域
        nextTick(() => {
          const el = document.querySelector('.monitor-section:last-child')
          if (el) el.scrollIntoView({ behavior: 'smooth', block: 'start' })
        })
        return
      }
    } catch (error) {
      console.error('Failed to fetch evaluation:', error)
    }
    if (attempt < maxRetries) {
      console.log(`[AI-Chat] Evaluation not ready yet, retry ${attempt}/${maxRetries} in ${retryDelayMs / 1000}s...`)
      await new Promise(resolve => setTimeout(resolve, retryDelayMs))
    }
  }
  evaluationLoading.value = false
  console.warn('[AI-Chat] Evaluation not available after', maxRetries, 'attempts')
}

// 手动刷新评估结果
const refreshEvaluation = () => {
  const id = lastTerminatedSessionId.value
  if (!id) return
  fetchEvaluation(id)
}
// 加载历史会话列表
const fetchSessionHistory = async () => {
  try {
    const response = await fetch('/api/v1/ai-chat/sessions', {
      headers: {
        'Authorization': `Bearer ${authStore.token}`
      }
    })
    if (response.ok) {
      const sessions = await response.json()
      sessionHistory.value = sessions.sort((a, b) => 
        new Date(b.created_at) - new Date(a.created_at)
      )
    }
  } catch (error) {
    console.error('Failed to fetch session history:', error)
  }
}

// 加载会话（用作上下文）
const loadSession = async (id) => {
  try {
    const response = await fetch(`/api/v1/ai-chat/sessions/${id}`, {
      headers: {
        'Authorization': `Bearer ${authStore.token}`
      }
    })
    if (response.ok) {
      const session = await response.json()
      
      // 加载会话配置
      sessionConfig.value.title = session.title
      sessionConfig.value.topic = session.topic
      sessionConfig.value.global_constraint = session.global_constraint
      sessionConfig.value.max_rounds = session.max_rounds
      sessionConfig.value.termination_config = session.termination_config
      
      // 加载Agent A配置（使用实际的后端字段名）
      sessionConfig.value.agent_a = {
        name: session.agent_a_name,
        role: session.agent_a_role,
        style: session.agent_a_style,
        model: session.agent_a_model,
        temperature: session.agent_a_temperature,
        max_tokens: session.agent_a_max_tokens
      }
      
      // 加载Agent B配置（使用实际的后端字段名）
      sessionConfig.value.agent_b = {
        name: session.agent_b_name,
        role: session.agent_b_role,
        style: session.agent_b_style,
        model: session.agent_b_model,
        temperature: session.agent_b_temperature,
        max_tokens: session.agent_b_max_tokens
      }
      
      // 加载消息历史
      messages.value = session.messages || []
      currentRound.value = session.current_round
      maxRounds.value = session.max_rounds
      
      // 不要设置sessionId - 让Start按钮创建新会话，将历史作为上下文
      sessionId.value = null
      
      // 更新token统计
      tokenUsage.value.agent_a = (session.token_usage?.agent_a_input || 0) + (session.token_usage?.agent_a_output || 0)
      tokenUsage.value.agent_b = (session.token_usage?.agent_b_input || 0) + (session.token_usage?.agent_b_output || 0)
      tokenUsage.value.total = session.token_usage?.total || 0
      
      // 关闭配置面板，确保聊天区域可见
      showConfig.value = false
      
      // 滚动到底部
      nextTick(() => {
        messageList.value?.scrollTo({ top: messageList.value.scrollHeight, behavior: 'smooth' })
      })
      
      alert(`已加载会话: ${session.title}`)
    }
  } catch (error) {
    console.error('Failed to load session:', error)
    alert('加载会话失败: ' + error.message)
  }
}

// 格式化历史日期
const formatHistoryDate = (timestamp) => {
  if (!timestamp) return ''
  const date = new Date(timestamp)
  const now = new Date()
  const diff = now - date
  const days = Math.floor(diff / (1000 * 60 * 60 * 24))
  
  if (days === 0) return '今天'
  if (days === 1) return '昨天'
  if (days < 7) return `${days}天前`
  return date.toLocaleDateString('zh-CN', { month: 'short', day: 'numeric' })
}

// 生命周期
onMounted(async () => {
  // 从 API 加载模板（优先使用数据库，失败时降级到 JSON 配置）
  await fetchTemplatesFromAPI()

  fetchModels()
  fetchMCPTools()
  fetchSessionHistory()

  // 如果有未完成的评估（页面刷新/导航前中断），自动恢复加载
  const pendingEvalId = localStorage.getItem(EVAL_SESSION_KEY)
  if (pendingEvalId) {
    console.log('[AI-Chat] Resuming evaluation fetch for session:', pendingEvalId)
    lastTerminatedSessionId.value = pendingEvalId
    // 先尝试直接获取（可能已经完成），失败再轮询
    fetchEvaluation(pendingEvalId)
  }
})

// Keep termination_config.max_rounds in sync with top-level max_rounds
watch(() => sessionConfig.value.max_rounds, (val) => {
  sessionConfig.value.termination_config.max_rounds = val
})

onUnmounted(() => {
  stopTimer()
  ws.value?.close()
})
</script>

<style scoped>
.ai-chat-view {
  display: flex;
  flex-direction: column;
  height: 100vh;
  background: var(--bg-secondary);
}

.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 20px;
  background: var(--bg-primary);
  border-bottom: 1px solid var(--border-primary);
}

.header-left {
  display: flex;
  align-items: center;
  gap: 20px;
}

.header-left h2 {
  margin: 0;
  font-size: 20px;
  color: var(--text-primary);
}

.mode-switcher {
  display: flex;
  gap: 8px;
}

.mode-btn {
  padding: 6px 12px;
  border-radius: 4px;
  text-decoration: none;
  color: var(--text-secondary);
  background: var(--bg-tertiary);
  font-size: 14px;
}

.mode-btn.active {
  background: var(--accent-primary);
  color: white;
}

.header-right {
  display: flex;
  gap: 10px;
}

.main-content {
  display: flex;
  flex: 1;
  overflow: hidden;
}

.session-history {
  width: 260px;
  background: var(--bg-secondary);
  border-right: 1px solid var(--border-primary);
  display: flex;
  flex-direction: column;
  overflow: hidden;
  transition: width 0.3s;
}

.session-history.collapsed {
  width: 0;
  min-width: 0;
  border: none;
}

.history-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 15px;
  border-bottom: 1px solid var(--border-primary);
  background: var(--bg-primary);
}

.history-header h4 {
  margin: 0;
  font-size: 14px;
  color: var(--text-primary);
}

.collapse-btn {
  background: none;
  border: none;
  cursor: pointer;
  padding: 4px;
  color: var(--text-secondary);
}

.collapse-btn:hover {
  color: var(--text-primary);
}

.history-list {
  flex: 1;
  overflow-y: auto;
  padding: 8px;
}

.history-item {
  padding: 12px;
  margin-bottom: 8px;
  background: var(--card-bg);
  border: 1px solid var(--border-primary);
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.2s;
}

.history-item:hover {
  border-color: var(--accent-primary);
  box-shadow: var(--shadow-sm);
}

.history-item.active {
  background: var(--bg-tertiary);
  border-color: var(--accent-primary);
}

.session-title {
  font-size: 13px;
  font-weight: 500;
  color: var(--text-primary);
  margin-bottom: 4px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.session-meta {
  display: flex;
  justify-content: space-between;
  font-size: 11px;
  color: var(--text-tertiary);
}

.empty-history {
  text-align: center;
  padding: 40px 20px;
  color: var(--text-tertiary);
  font-size: 13px;
}

.expand-btn {
  position: absolute;
  left: 0;
  top: 50%;
  transform: translateY(-50%);
  width: 24px;
  height: 60px;
  background: var(--bg-secondary);
  border: 1px solid var(--border-primary);
  border-left: none;
  border-radius: 0 4px 4px 0;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: all 0.2s;
}

.expand-btn:hover {
  background: var(--bg-primary);
  width: 28px;
}

.config-panel {
  width: 400px;
  background: var(--bg-primary);
  border-right: 1px solid var(--border-primary);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.panel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 15px 20px;
  border-bottom: 1px solid var(--border-primary);
}

.panel-header h3 {
  margin: 0;
  color: var(--text-primary);
}

.close-btn {
  background: none;
  border: none;
  font-size: 24px;
  cursor: pointer;
  color: var(--text-tertiary);
}

.panel-body {
  flex: 1;
  overflow-y: auto;
  padding: 20px;
}

.config-section {
  margin-bottom: 24px;
  padding-bottom: 20px;
  border-bottom: 1px solid var(--border-primary);
}

.config-section h4 {
  margin: 0 0 15px 0;
  color: var(--text-primary);
  font-size: 16px;
}

/* Multi-agent styles */
.agent-card {
  background: var(--bg-tertiary);
  border-radius: 8px;
  padding: 16px;
  margin-bottom: 12px;
}

.agent-card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
  font-weight: 500;
}

.remove-agent-btn {
  background: transparent;
  border: none;
  font-size: 16px;
  cursor: pointer;
  color: var(--text-tertiary);
  padding: 4px 8px;
}

.remove-agent-btn:hover {
  color: var(--accent-danger);
}

.add-agent-section {
  text-align: center;
  padding: 16px;
  margin-bottom: 24px;
}

.add-agent-btn {
  padding: 12px 24px;
  background: var(--bg-tertiary);
  border: 2px dashed var(--accent-primary);
  border-radius: 8px;
  color: var(--accent-primary);
  font-size: 14px;
  cursor: pointer;
  transition: all 0.2s;
}

.add-agent-btn:hover {
  background: var(--bg-hover);
}

.add-agent-hint {
  margin: 8px 0 0 0;
  font-size: 12px;
  color: var(--text-tertiary);
}

.form-group {
  margin-bottom: 15px;
}

.form-group label {
  display: block;
  margin-bottom: 5px;
  font-size: 14px;
  color: var(--text-secondary);
}

.form-group input,
.form-group textarea,
.form-group select {
  width: 100%;
  padding: 8px 12px;
  border: 1px solid var(--border-secondary);
  border-radius: 4px;
  font-size: 14px;
  background: var(--input-bg);
  color: var(--text-primary);
}

.form-group input[type="range"] {
  width: calc(100% - 50px);
}

.form-row {
  display: flex;
  gap: 15px;
}

.form-row .form-group {
  flex: 1;
}

.panel-footer {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  padding: 15px 20px;
  border-top: 1px solid var(--border-primary);
}

.chat-area {
  flex: 1;
  display: flex;
  flex-direction: column;
  background: var(--bg-primary);
}

.chat-area.with-monitor {
  min-width: 0;  /* 允许 flex 收缩，让监控面板正常显示在右侧 */
}

.control-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 20px;
  background: var(--bg-secondary);
  border-bottom: 1px solid var(--border-primary);
}

.status-info {
  display: flex;
  align-items: center;
  gap: 15px;
}

.status-badge {
  padding: 4px 12px;
  border-radius: 12px;
  font-size: 12px;
  font-weight: 500;
}

.status-badge.pending {
  background: var(--bg-tertiary);
  color: var(--text-secondary);
}

.status-badge.running {
  background: var(--accent-success);
  color: white;
}

.status-badge.paused {
  background: var(--accent-warning);
  color: white;
}

.round-info,
.token-info {
  font-size: 14px;
  color: var(--text-secondary);
}

.control-buttons {
  display: flex;
  gap: 10px;
}

.director-panel {
  padding: 12px 20px;
  background: #f6ffed;
  border-bottom: 1px solid #b7eb8f;
}

.session-end-banner {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 20px;
  background: #f6ffed;
  border-bottom: 2px solid #52c41a;
  color: #389e0d;
  font-size: 14px;
  font-weight: 500;
}

.btn-dismiss {
  background: none;
  border: none;
  color: #666;
  font-size: 18px;
  cursor: pointer;
  padding: 0 4px;
  line-height: 1;
}

.btn-dismiss:hover {
  color: #333;
}

.director-input {
  display: flex;
  gap: 10px;
}

.director-input select {
  width: 120px;
  padding: 8px;
  border: 1px solid #d9d9d9;
  border-radius: 4px;
}

.director-input input {
  flex: 1;
  padding: 8px 12px;
  border: 1px solid #d9d9d9;
  border-radius: 4px;
}

.message-list {
  flex: 1;
  overflow-y: auto;
  padding: 20px;
}

.message {
  margin-bottom: 20px;
  padding: 15px;
  border-radius: 8px;
  max-width: 80%;
}

.message.agent-a {
  background: #e6f7ff;
  margin-right: auto;
}

.message.agent-b {
  background: #f6ffed;
  margin-left: auto;
}

.message.system-message {
  background: #fffbe6;
  border: 1px dashed #ffd666;
  margin: 8px auto;
  max-width: 60%;
  font-size: 13px;
  text-align: center;
}

.message-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
  font-size: 13px;
}

.agent-icon {
  font-size: 16px;
}

.agent-name {
  font-weight: 600;
  color: #333;
}

.message-round {
  color: #1890ff;
  background: rgba(24, 144, 255, 0.1);
  padding: 2px 8px;
  border-radius: 10px;
}

.message-time {
  color: #999;
  margin-left: auto;
}

.message-content {
  line-height: 1.6;
  color: #333;
  white-space: pre-wrap;
}

/* 搜索结果样式 - 使用仿宋体和较小字号 */
/* 使用 :deep() 以便在 v-html 渲染的内容中生效 */
.message-content :deep(.search-results-block) {
  font-family: 'FangSong', 'STFangsong', '仿宋', 'KaiTi', serif !important;
  font-size: 0.85em !important; /* 比正常文本小2号 */
  color: #555 !important;
  background: rgba(33, 150, 243, 0.04) !important;
  padding: 4px !important;
  border-radius: 4px !important;
  margin: 4px 0 !important;
  border-left: 3px solid #2196f3 !important;
  line-height: 1.2 !important;
  display: block !important;
}

/* 搜索结果标题加粗 */
.message-content :deep(.search-results-block strong) {
  color: #1976d2;
}

/* 搜索结果链接样式 */
.message-content :deep(.search-results-block a) {
  color: #1976d2;
  text-decoration: none;
}

.message-content :deep(.search-results-block a:hover) {
  text-decoration: underline;
}

.message-meta {
  display: flex;
  gap: 15px;
  margin-top: 8px;
  font-size: 12px;
  color: #999;
}

.tool-calls {
  margin-top: 10px;
  padding: 10px;
  background: rgba(0, 0, 0, 0.03);
  border-radius: 4px;
}

.tool-call-item {
  margin-bottom: 8px;
}

.tool-name {
  font-weight: 500;
  color: #1890ff;
}

.tool-args {
  margin: 5px 0 0 0;
  padding: 8px;
  background: white;
  border-radius: 4px;
  font-size: 12px;
  overflow-x: auto;
}

.typing-indicator {
  display: flex;
  gap: 6px;
  padding: 15px;
  justify-content: center;
}

.dot {
  width: 8px;
  height: 8px;
  background: #ccc;
  border-radius: 50%;
  animation: bounce 1.4s infinite ease-in-out;
}

.dot:nth-child(1) { animation-delay: -0.32s; }
.dot:nth-child(2) { animation-delay: -0.16s; }

@keyframes bounce {
  0%, 80%, 100% { transform: scale(0); }
  40% { transform: scale(1); }
}

.empty-state {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  color: #999;
}

.empty-icon {
  font-size: 48px;
  margin-bottom: 20px;
}

.monitor-panel {
  width: 300px;
  flex-shrink: 0;
  background: white;
  border-left: 1px solid #e0e0e0;
  display: flex;
  flex-direction: column;
}

.monitor-panel .panel-header {
  padding: 15px;
}

.monitor-panel .panel-header h4 {
  margin: 0;
}

.monitor-panel .panel-body {
  flex: 1;
  overflow-y: auto;
  padding: 15px;
}

.monitor-section {
  margin-bottom: 24px;
}

.evaluation-loading {
  color: #888;
  font-size: 13px;
  padding: 12px 0;
  display: flex;
  align-items: center;
  gap: 6px;
}

.btn-refresh-eval {
  margin-left: 8px;
  background: none;
  border: none;
  cursor: pointer;
  font-size: 13px;
  padding: 2px 4px;
  border-radius: 4px;
  vertical-align: middle;
  opacity: 0.7;
}

.btn-refresh-eval:hover {
  opacity: 1;
  background: #f0f0f0;
}

.evaluation-summary {
  margin-top: 12px;
  padding: 10px;
  background: #f8f9fa;
  border-radius: 6px;
  font-size: 13px;
  line-height: 1.6;
}

.evaluation-highlights {
  margin-top: 10px;
  font-size: 12px;
}

.evaluation-highlights ul {
  margin: 6px 0 0 0;
  padding-left: 16px;
}

.evaluation-highlights li {
  margin-bottom: 4px;
  line-height: 1.5;
}

.monitor-section h5 {
  margin: 0 0 12px 0;
  font-size: 14px;
  color: #666;
}

.token-bar {
  display: flex;
  height: 24px;
  border-radius: 12px;
  overflow: hidden;
  background: #f0f0f0;
}

.token-segment {
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 11px;
  color: white;
  transition: width 0.3s;
}

.token-segment.agent-a {
  background: #1890ff;
}

.token-segment.agent-b {
  background: #52c41a;
}

.token-detail {
  margin-top: 8px;
  font-size: 12px;
  color: #666;
}

.flow-graph {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.round-node {
  width: 28px;
  height: 28px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
  font-size: 12px;
  font-weight: 500;
}

.round-node.completed {
  background: #52c41a;
  color: white;
}

.round-node.current {
  background: #1890ff;
  color: white;
  animation: pulse 2s infinite;
}

.round-node.pending {
  background: #f0f0f0;
  color: #999;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.6; }
}

.stats-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 12px;
}

.stat-item {
  display: flex;
  flex-direction: column;
  padding: 10px;
  background: #f5f5f5;
  border-radius: 4px;
}

.stat-label {
  font-size: 12px;
  color: #999;
}

.stat-value {
  font-size: 18px;
  font-weight: 600;
  color: #333;
}

.btn {
  padding: 8px 16px;
  border: none;
  border-radius: 4px;
  font-size: 14px;
  cursor: pointer;
  transition: all 0.2s;
}

.btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-primary {
  background: #1890ff;
  color: white;
}

.btn-primary:hover:not(:disabled) {
  background: #40a9ff;
}

.btn-secondary {
  background: #f0f0f0;
  color: #666;
}

.btn-secondary:hover {
  background: #e0e0e0;
}

.btn-success {
  background: #52c41a;
  color: white;
}

.btn-success:hover:not(:disabled) {
  background: #73d13d;
}

.btn-warning {
  background: #faad14;
  color: white;
}

.btn-warning:hover {
  background: #ffc53d;
}

.btn-danger {
  background: #ff4d4f;
  color: white;
}

.btn-danger:hover {
  background: #ff7875;
}

/* 模板弹窗 */
.modal {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.modal-content {
  background: white;
  border-radius: 8px;
  width: 600px;
  max-height: 80vh;
  display: flex;
  flex-direction: column;
}

.modal-content-wide {
  width: 760px;
}

.modal-content-editor {
  width: 820px;
  max-height: 92vh;
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 20px;
  border-bottom: 1px solid #e0e0e0;
  flex-shrink: 0;
}

.modal-header h3 {
  margin: 0;
}

.modal-header-actions {
  display: flex;
  gap: 10px;
  align-items: center;
}

.btn-sm {
  padding: 5px 12px;
  font-size: 13px;
}

.modal-body {
  flex: 1;
  overflow-y: auto;
  padding: 20px;
}

/* ── 模板卡片重构 ── */
.template-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 12px;
}

.template-card {
  border: 1px solid #e0e0e0;
  border-radius: 8px;
  overflow: hidden;
  transition: all 0.2s;
  display: flex;
  flex-direction: column;
}

.template-card:hover {
  border-color: #1890ff;
  box-shadow: 0 2px 8px rgba(24, 144, 255, 0.15);
}

.template-card-body {
  display: flex;
  gap: 12px;
  padding: 14px;
  cursor: pointer;
  flex: 1;
  align-items: flex-start;
}

.template-card-body:hover {
  background: #f9fbff;
}

.template-icon {
  font-size: 28px;
  flex-shrink: 0;
  width: 36px;
  text-align: center;
}

.template-info h4 {
  margin: 0 0 4px 0;
  font-size: 14px;
  font-weight: 600;
}

.template-info p {
  margin: 0 0 6px 0;
  font-size: 12px;
  color: #666;
  line-height: 1.4;
}

.template-badge {
  display: inline-block;
  padding: 1px 7px;
  border-radius: 10px;
  font-size: 11px;
}

.template-badge.builtin {
  background: #e6f7ff;
  color: #1890ff;
}

.template-badge.custom {
  background: #f6ffed;
  color: #52c41a;
}

.template-card-actions {
  display: flex;
  border-top: 1px solid #f0f0f0;
  padding: 6px 10px;
  gap: 4px;
  justify-content: flex-end;
  background: #fafafa;
}

.tpl-btn {
  background: none;
  border: 1px solid #e0e0e0;
  border-radius: 4px;
  padding: 3px 8px;
  cursor: pointer;
  font-size: 13px;
  transition: all 0.15s;
}

.tpl-btn:hover {
  background: #f0f0f0;
  border-color: #bbb;
}

.tpl-btn-danger:hover {
  background: #fff1f0;
  border-color: #ffccc7;
  color: #f5222d;
}

/* ── 模板编辑器 ── */
.editor-body {
  overflow-y: auto;
  padding: 20px;
}

.editor-section {
  margin-bottom: 20px;
  padding-bottom: 16px;
  border-bottom: 1px solid #f0f0f0;
}

.editor-section:last-of-type {
  border-bottom: none;
}

.editor-section-title {
  margin: 0 0 14px 0;
  font-size: 14px;
  font-weight: 600;
  color: #333;
  padding-left: 8px;
  border-left: 3px solid #1890ff;
}

.editor-row {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
}

.editor-field {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-width: 150px;
  margin-bottom: 10px;
}

.editor-field-sm {
  flex: 0 0 130px;
  min-width: 110px;
}

.editor-field label {
  font-size: 12px;
  color: #666;
  margin-bottom: 4px;
  font-weight: 500;
}

.editor-field input,
.editor-field select,
.editor-field textarea {
  padding: 7px 10px;
  border: 1px solid #d9d9d9;
  border-radius: 4px;
  font-size: 13px;
  transition: border-color 0.2s;
  font-family: inherit;
}

.editor-field input:focus,
.editor-field select:focus,
.editor-field textarea:focus {
  outline: none;
  border-color: #1890ff;
  box-shadow: 0 0 0 2px rgba(24, 144, 255, 0.1);
}

.editor-footer {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  padding-top: 14px;
  border-top: 1px solid #f0f0f0;
}

/* View Switcher */
.view-switcher {
  display: flex;
  gap: 8px;
}

/* Timeline View */
.timeline-view {
  flex: 1;
  overflow-y: auto;
  padding: 20px;
}

.timeline-container {
  position: relative;
  padding-left: 100px;
}

.timeline-line {
  position: absolute;
  left: 50px;
  top: 0;
  bottom: 0;
  width: 2px;
  background: linear-gradient(to bottom, #409eff, #67c23a);
}

.timeline-item {
  position: relative;
  margin-bottom: 20px;
}

.timeline-item.agent_a .timeline-card {
  margin-left: 0;
  margin-right: 50%;
}

.timeline-item.agent_b .timeline-card {
  margin-left: 50%;
  margin-right: 0;
}

.timeline-marker {
  position: absolute;
  left: -58px;
  top: 20px;
  width: 16px;
  height: 16px;
  border-radius: 50%;
  border: 3px solid #fff;
  box-shadow: 0 2px 4px rgba(0,0,0,0.1);
}

.timeline-marker.agent_a {
  background: #409eff;
}

.timeline-marker.agent_b {
  background: #67c23a;
}

.timeline-card {
  background: #fff;
  border-radius: 8px;
  padding: 12px 16px;
  box-shadow: 0 2px 8px rgba(0,0,0,0.1);
}

.timeline-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.timeline-agent {
  font-weight: 600;
  font-size: 14px;
}

.timeline-round {
  font-size: 12px;
  color: #909399;
  background: #f5f7fa;
  padding: 2px 8px;
  border-radius: 4px;
}

.timeline-content {
  font-size: 14px;
  line-height: 1.6;
  color: #333;
}

.timeline-footer {
  display: flex;
  gap: 12px;
  margin-top: 8px;
  font-size: 12px;
  color: #909399;
}

/* Split View */
.split-view {
  flex: 1;
  display: flex;
  overflow: hidden;
}

.split-column {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.split-column.agent-a-column {
  background: linear-gradient(to bottom, rgba(64, 158, 255, 0.05), rgba(64, 158, 255, 0.02));
}

.split-column.agent-b-column {
  background: linear-gradient(to bottom, rgba(103, 194, 58, 0.05), rgba(103, 194, 58, 0.02));
}

.column-header {
  padding: 12px 16px;
  font-weight: 600;
  font-size: 16px;
  border-bottom: 1px solid #ebeef5;
  background: rgba(255,255,255,0.8);
}

.column-messages {
  flex: 1;
  overflow-y: auto;
  padding: 12px;
}

.split-message {
  background: #fff;
  border-radius: 8px;
  padding: 12px;
  margin-bottom: 12px;
  box-shadow: 0 1px 4px rgba(0,0,0,0.05);
}

.split-round {
  font-size: 11px;
  color: #909399;
  margin-bottom: 4px;
}

.split-content {
  font-size: 14px;
  line-height: 1.6;
}

.split-meta {
  font-size: 11px;
  color: #909399;
  margin-top: 8px;
}

.split-divider {
  width: 60px;
  background: #f5f7fa;
  border-left: 1px solid #ebeef5;
  border-right: 1px solid #ebeef5;
  display: flex;
  flex-direction: column;
  align-items: center;
  padding-top: 20px;
}

.round-markers {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.round-marker {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  font-weight: 500;
  background: #e4e7ed;
  color: #909399;
}

.round-marker.active {
  background: #409eff;
  color: #fff;
}
</style>
