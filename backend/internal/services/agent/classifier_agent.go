package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// ClassifierAgent classifies user tasks into predefined categories
type ClassifierAgent struct {
	baseAgent
}

// TaskClassification represents the classification result of a task
type TaskClassification struct {
	TaskCategory    string   `json:"task_category"`    // 任务类别：主要类别+子类别
	Complexity      string   `json:"complexity"`       // 复杂度：简单/中等/复杂
	RequiredSkills  []string `json:"required_skills"`  // 所需技能列表
	TimeEstimate    string   `json:"time_estimate"`    // 时间预估
	KeyRequirements string   `json:"key_requirements"` // 关键需求
	IsSimpleTask    bool     `json:"is_simple_task"`   // 是否为简单任务，可以在chat页面完成
}

// NewClassifierAgent creates a new classifier agent
func NewClassifierAgent() *ClassifierAgent {
	agent := &ClassifierAgent{}

	// Try to load configuration from agent_configs.json in the current directory first
	// Then try from plans directory
	configPaths := []string{"./plans/agent_configs.json"}

	for _, configPath := range configPaths {
		configData, err := os.ReadFile(configPath)
		if err == nil {
			// Parse JSON data
			var configs []AgentConfig
			if err := json.Unmarshal(configData, &configs); err == nil {
				// Find the classifier agent configuration
				for _, config := range configs {
					if config.Name == "classifier" {
						agent.name = config.Name
						agent.role = config.Role
						agent.description = config.Description
						agent.prompt = config.Prompt
						return agent
					}
				}
			}
		}
	}

	// Fallback to default values if configuration loading fails
	agent.name = "classifier"
	agent.role = "classifier"
	agent.description = "Classifies user tasks into predefined categories"
	agent.prompt = `你是一个专业的任务分类器Agent，负责分析用户输入的任务需求，并将其归类到合适的类别中，以便后续规划任务处理方法和步骤。

## 核心职责
- 准确理解用户的任务需求，提取关键信息
- 将任务归类到预定义的主要类别和子类别
- 分析任务的复杂度、所需技能、时间要求等关键特征
- 判断任务是否为简单任务（可以直接在chat页面完成）
- 输出结构化的分类结果，便于后续Agent系统处理

## 分类范围
### 主要类别
1. **编程开发**：编写、修改、调试代码，开发软件或应用
2. **内容创作**：撰写文章、报告、文案、故事等文字内容
3. **数据分析**：处理、分析、可视化数据，生成数据报告
4. **设计**：UI/UX设计、图形设计、网页设计等
5. **问题解决**：解答技术问题、解决系统故障、提供方案建议
6. **研究调研**：收集、整理、分析信息，撰写研究报告
7. **其他**：不属于上述类别的任务

### 子类别示例
- 编程开发：Web开发、移动开发、后端开发、算法实现、代码优化
- 内容创作：技术文档、营销文案、创意写作、论文撰写
- 数据分析：数据清洗、统计分析、机器学习建模、数据可视化

## 分析维度
请在分类结果中包含以下关键特征：
1. **任务类别**：主要类别 + 子类别
2. **复杂度**：简单/中等/复杂
3. **所需技能**：列出完成任务需要的主要技能
4. **时间预估**：完成任务的大致时间范围
5. **关键需求**：任务的核心目标和关键约束
6. **是否为简单任务**：判断任务是否可以直接在chat页面完成（true/false）

## 简单任务判断标准
如果任务符合以下条件，则视为简单任务：
- 不需要编写或执行代码
- 不需要复杂的多步骤操作
- 可以通过直接回答或简单查询完成
- 任务执行时间在10分钟以内

## 输出格式
请使用JSON格式输出分类结果，必须包含is_simple_task字段，示例如下：
[json]
{
  "task_category": "编程开发-Web开发",
  "complexity": "中等",
  "required_skills": ["HTML", "CSS", "JavaScript", "React"],
  "time_estimate": "2-3小时",
  "key_requirements": "创建一个响应式的登录页面，包含表单验证和用户反馈",
  "is_simple_task": false
}
[/json]

## 示例任务分类
1. **用户输入**："帮我写一个Python脚本，爬取某个网站的新闻标题和链接，并保存到CSV文件中"
   **分类结果**：
   [json]
   {
     "task_category": "编程开发-数据采集",
     "complexity": "中等",
     "required_skills": ["Python", "BeautifulSoup/Scrapy", "CSV处理"],
     "time_estimate": "1-2小时",
     "key_requirements": "爬取指定网站新闻标题和链接，保存为CSV格式"
   }
   [/json]

2. **用户输入**："我需要一份关于2024年人工智能发展趋势的报告"
   **分类结果**：
   [json]
   {
     "task_category": "研究调研-行业分析",
     "complexity": "复杂",
     "required_skills": ["资料收集", "数据分析", "报告撰写"],
     "time_estimate": "4-6小时",
     "key_requirements": "撰写2024年人工智能发展趋势报告"
   }
   [/json]

请严格按照上述要求对用户的任务进行分类和分析。`

	return agent
}

// Execute runs the classifier agent
func (a *ClassifierAgent) Execute(ctx *TaskContext, input string, mcpManager *MCPManager, skillRegistry *SkillRegistry) (string, error) {
	// Enhance input with task context
	enhancedInput := a.enhanceInput(input, ctx)

	// Call LLM to classify the task
	response, err := a.callLLM(enhancedInput)
	if err != nil {
		return "", fmt.Errorf("failed to classify task: %w", err)
	}

	// Parse classification result
	classification, err := a.parseClassification(response)
	if err != nil {
		return "", fmt.Errorf("failed to parse classification: %w", err)
	}

	// Format output
	return a.formatClassification(classification), nil
}

// enhanceInput enhances the input with additional context
func (a *ClassifierAgent) enhanceInput(input string, ctx *TaskContext) string {
	// If there's existing agent execution history, add it to the input
	if ctx != nil && len(ctx.AgentHistory) > 0 {
		// For simplicity, we'll just mention that there's existing history without detailing it
		input = fmt.Sprintf("注意：此任务有相关的历史执行记录。\n\n任务：\n%s", input)
	}
	return input
}

// callLLM is a helper method to call the LLM with the agent's prompt
func (a *ClassifierAgent) callLLM(input string) (string, error) {
	// Format the input with the agent's prompt
	prompt := a.GetPrompt() + "\n\n" + "用户任务：" + input

	// Call the LLM client
	return a.CallLLM(prompt)
}

// parseClassification parses the LLM response into TaskClassification
func (a *ClassifierAgent) parseClassification(response string) (*TaskClassification, error) {
	// Extract JSON from response
	jsonStart := strings.Index(response, "{")
	jsonEnd := strings.LastIndex(response, "}")

	if jsonStart == -1 || jsonEnd == -1 || jsonStart > jsonEnd {
		return nil, fmt.Errorf("无法从响应中提取有效的JSON分类结果")
	}

	jsonStr := response[jsonStart : jsonEnd+1]

	// Parse JSON into struct
	var classification TaskClassification
	if err := json.Unmarshal([]byte(jsonStr), &classification); err != nil {
		return nil, fmt.Errorf("解析分类结果JSON失败：%v", err)
	}

	return &classification, nil
}

// formatClassification formats the classification result
func (a *ClassifierAgent) formatClassification(classification *TaskClassification) string {
	// Convert to JSON with indentation
	jsonData, err := json.MarshalIndent(classification, "", "  ")
	if err != nil {
		return fmt.Sprintf("格式化分类结果失败：%v", err)
	}

	return string(jsonData)
}
