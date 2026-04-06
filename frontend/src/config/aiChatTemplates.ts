/**
 * AI-AI Chat Templates Configuration
 * 
 * Templates are now loaded from /public/config/ai-chat-templates.json
 * Non-developers can easily modify the JSON file to add/edit templates
 */

export interface AgentStyle {
  language_style: 'professional' | 'casual' | 'formal'
  knowledge_level: 'expert' | 'intermediate' | 'beginner'
  tone: string
}

export interface AgentConfig {
  name: string
  role: string
  style: AgentStyle
  model: string
  temperature: number
  max_tokens: number
  allowed_tools: string[]
}

export interface TemplateConfig {
  title: string
  topic: string
  global_constraint?: string
  max_rounds: number
  agent_a?: AgentConfig
  agent_b?: AgentConfig
}

export interface ChatTemplate {
  id: string
  name: string
  description: string
  icon: string
  config: TemplateConfig
}

/**
 * Load templates — returns the hardcoded fallback.
 * Templates are served by the backend API (/api/v1/ai-chat/templates);
 * this function is only reached when the API is unavailable.
 */
export function loadTemplates(): ChatTemplate[] {
  return getDefaultTemplates()
}

/**
 * Default fallback templates (in case JSON file is not available)
 */
function getDefaultTemplates(): ChatTemplate[] {
  return [
    {
      id: 'debate-tech-ethics',
      name: 'Tech Ethics Debate',
      description: 'Two AIs debate on technology ethics topics',
      icon: '⚖️',
      config: {
        title: 'AI Ethics Debate',
        topic: 'Should AI development prioritize efficiency or safety?',
        global_constraint: 'Keep each response under 150 words',
        max_rounds: 8
      }
    }
  ]
}

/**
 * Predefined AI-AI Chat Templates (kept for backward compatibility)
 * @deprecated Use loadTemplates() instead
 */
export const aiChatTemplates: ChatTemplate[] = []

/**
 * Get template by ID
 */
export function getTemplateById(templates: ChatTemplate[], id: string): ChatTemplate | undefined {
  return templates.find(template => template.id === id)
}

/**
 * Get all template IDs
 */
export function getTemplateIds(templates: ChatTemplate[]): string[] {
  return templates.map(template => template.id)
}
