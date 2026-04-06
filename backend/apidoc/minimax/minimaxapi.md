# MiniMax API 文档

## 概述

MiniMax 是一家通用人工智能公司，提供多模态大模型能力，包括文本生成、语音合成、图像生成等服务。本文档主要介绍 MiniMax 的文本对话 API。

## API 基础信息

- **官方网站**: https://www.minimaxi.com/
- **开放平台**: https://platform.minimaxi.com/
- **Base URL**: `https://api.minimax.chat/v1`
- **认证方式**: Bearer Token（通过 Authorization Header 传递）
- **请求格式**: JSON
- **响应格式**: JSON（支持流式和非流式响应）

## 认证

### 获取 API Key

1. 访问 [MiniMax 开放平台](https://platform.minimaxi.com/)
2. 注册并登录账号
3. 在控制台「API管理」中创建 API Key
4. 保存 API Key（仅显示一次）

### 认证方式

在请求头中添加 Authorization：

```http
POST https://api.minimax.chat/v1/text/chatcompletion_pro?GroupId={group_id}
Authorization: Bearer {api_key}
Content-Type: application/json
```

**注意事项**：
- API Key 必须通过 `Authorization: Bearer {api_key}` 方式传递
- Group ID 作为 URL 查询参数传递
- API Key 和 Group ID 均可在开放平台控制台获取

## 支持的模型

MiniMax 提供多个系列的大语言模型：

### minimax 系列（通用对话模型）

| 模型名称 | 模型ID | 上下文长度 | 特点 | 推荐场景 |
|---------|--------|-----------|------|---------|
| abab6.5-chat | `abab6.5-chat` | 245K tokens | 最新旗舰版本，性能最强 | 复杂任务、长文本理解、代码生成 |
| abab6.5s-chat | `abab6.5s-chat` | 245K tokens | 速度优化版本 | 需要快速响应的场景 |
| abab6.5t-chat | `abab6.5t-chat` | 8K tokens | 轻量级版本 | 简单对话、成本敏感场景 |
| abab6.5g-chat | `abab6.5g-chat` | 8K tokens | 通用版本 | 日常对话 |
| abab5.5-chat | `abab5.5-chat` | 16K tokens | 标准版本 | 通用对话 |
| abab5.5s-chat | `abab5.5s-chat` | 8K tokens | 快速版本 | 快速对话 |

### 价格说明

不同模型的定价不同，具体请参考 [MiniMax 定价页面](https://platform.minimaxi.com/docs/pricing/overview)。

- 按 token 计费，输入和输出 token 分开计算
- abab6.5 系列价格较高但能力更强
- abab5.5 系列性价比较高

## API 端点

### 1. ChatCompletion Pro（推荐）

这是 MiniMax 最新的对话补全接口，支持流式和非流式输出。

**端点**: `/text/chatcompletion_pro`

**方法**: POST

**完整 URL**: 
```
https://api.minimax.chat/v1/text/chatcompletion_pro?GroupId={group_id}
```

#### 请求参数

```json
{
  "model": "abab6.5-chat",
  "messages": [
    {
      "role": "system",
      "content": "你是一个AI助手"
    },
    {
      "role": "user", 
      "content": "你好"
    }
  ],
  "stream": true,
  "temperature": 0.7,
  "top_p": 0.95,
  "max_tokens": 2048,
  "tools": [],
  "tool_choice": "none"
}
```

#### 请求头

```http
Content-Type: application/json
Authorization: Bearer {api_key}
```

#### 参数详细说明

| 参数 | 类型 | 必填 | 默认值 | 说明 |
|-----|------|-----|-------|------|
| model | string | 是 | - | 模型名称，如 `abab6.5-chat` |
| messages | array | 是 | - | 对话消息列表 |
| messages[].role | string | 是 | - | 消息角色：`system`/`user`/`assistant`/`tool` |
| messages[].content | string | 是 | - | 消息内容 |
| stream | boolean | 否 | false | 是否启用流式输出 |
| temperature | float | 否 | 0.7 | 温度参数，范围 0.0-1.0，值越大输出越随机 |
| top_p | float | 否 | 0.95 | 核采样参数，范围 0.0-1.0 |
| max_tokens | integer | 否 | 模型最大值 | 最大生成 token 数 |
| tools | array | 否 | [] | 函数调用工具列表 |
| tool_choice | string | 否 | "none" | 工具选择策略：`none`/`auto`/`{name}` |
| tokens_to_generate | integer | 否 | 512 | 建议生成的 token 数量 |
| mask_sensitive_info | boolean | 否 | false | 是否脱敏输出中的敏感信息 |

#### 响应示例（非流式）

**成功响应** (HTTP 200)：

```json
{
  "id": "chat-20240101123456",
  "object": "chat.completion",
  "created": 1704096896,
  "model": "abab6.5-chat",
  "choices": [
    {
      "index": 0,
      "message": {
        "role": "assistant",
        "content": "你好！我是 MiniMax AI 助手，很高兴为你服务。有什么我可以帮助你的吗？"
      },
      "finish_reason": "stop"
    }
  ],
  "usage": {
    "prompt_tokens": 15,
    "completion_tokens": 25,
    "total_tokens": 40
  },
  "input_sensitive": false,
  "output_sensitive": false
}
```

**finish_reason 说明**：
- `stop`: 正常结束
- `length`: 达到最大长度限制
- `tool_calls`: 需要调用工具
- `content_filter`: 内容被过滤

#### 响应示例（流式）

流式响应采用 Server-Sent Events (SSE) 格式，每条消息以 `data:` 开头：

```
data: {"id":"chat-20240101123456","object":"chat.completion.chunk","created":1704096896,"model":"abab6.5-chat","choices":[{"index":0,"delta":{"role":"assistant","content":"你好"},"finish_reason":null}]}

data: {"id":"chat-20240101123456","object":"chat.completion.chunk","created":1704096896,"model":"abab6.5-chat","choices":[{"index":0,"delta":{"content":"！"},"finish_reason":null}]}

data: {"id":"chat-20240101123456","object":"chat.completion.chunk","created":1704096896,"model":"abab6.5-chat","choices":[{"index":0,"delta":{"content":"我是"},"finish_reason":null}]}

data: {"id":"chat-20240101123456","object":"chat.completion.chunk","created":1704096896,"model":"abab6.5-chat","choices":[{"index":0,"delta":{},"finish_reason":"stop"}],"usage":{"prompt_tokens":15,"completion_tokens":25,"total_tokens":40}}

data: [DONE]
```

**流式响应字段说明**：
- 每个 chunk 包含 `delta` 字段，表示本次新增的内容
- 最后一个 chunk 的 `finish_reason` 非空，表示生成结束
- 最后会发送 `[DONE]` 标记流结束

### 2. ChatCompletion V2（兼容接口）

这是较早的接口版本，功能类似但推荐使用 Pro 版本。

**端点**: `/text/chatcompletion_v2`

**参数和响应格式与 Pro 版本类似**

## 函数调用（Function Calling / Tools）

MiniMax 支持函数调用功能，允许模型调用外部工具。

### 工具定义示例

```json
{
  "model": "abab6.5-chat",
  "messages": [
    {
      "role": "user",
      "content": "北京今天天气怎么样？"
    }
  ],
  "tools": [
    {
      "type": "function",
      "function": {
        "name": "get_weather",
        "description": "获取指定城市的天气信息",
        "parameters": {
          "type": "object",
          "properties": {
            "city": {
              "type": "string",
              "description": "城市名称"
            },
            "unit": {
              "type": "string",
              "enum": ["celsius", "fahrenheit"],
              "description": "温度单位"
            }
          },
          "required": ["city"]
        }
      }
    }
  ],
  "tool_choice": "auto"
}
```

### 工具调用响应

```json
{
  "id": "chat-20240101123456",
  "choices": [
    {
      "message": {
        "role": "assistant",
        "content": null,
        "tool_calls": [
          {
            "id": "call_abc123",
            "type": "function",
            "function": {
              "name": "get_weather",
              "arguments": "{\"city\":\"北京\",\"unit\":\"celsius\"}"
            }
          }
        ]
      },
      "finish_reason": "tool_calls"
    }
  ]
}
```

### 工具响应提交

将工具执行结果作为消息提交：

```json
{
  "model": "abab6.5-chat",
  "messages": [
    {
      "role": "user",
      "content": "北京今天天气怎么样？"
    },
    {
      "role": "assistant",
      "tool_calls": [...]
    },
    {
      "role": "tool",
      "tool_call_id": "call_abc123",
      "content": "{\"temperature\":15,\"condition\":\"晴\"}"
    }
  ]
}
```

## 错误处理

### 错误响应格式

```json
{
  "error": {
    "message": "Invalid API key provided",
    "type": "invalid_request_error",
    "code": "invalid_api_key"
  }
}
```

### 常见错误码

| HTTP 状态码 | 错误码 | 说明 | 解决方案 |
|-----------|-------|------|---------|
| 400 | invalid_request_error | 请求参数错误 | 检查请求参数格式 |
| 401 | invalid_api_key | API Key 无效或未提供 | 检查 Authorization Header |
| 403 | insufficient_quota | 配额不足 | 充值或升级套餐 |
| 429 | rate_limit_exceeded | 请求频率超限 | 降低请求频率或升级套餐 |
| 500 | server_error | 服务器内部错误 | 重试或联系技术支持 |
| 503 | service_unavailable | 服务暂时不可用 | 稍后重试 |

### 错误处理最佳实践

1. **实现重试机制**：对于 5xx 错误和 429 错误，使用指数退避重试
2. **日志记录**：记录所有错误响应以便排查问题
3. **降级策略**：在服务不可用时提供备选方案
4. **配额监控**：监控 API 使用量，避免超出限制

## 速率限制

### 限制说明

MiniMax API 有以下限制：

- **RPM (Requests Per Minute)**: 每分钟请求数
- **TPM (Tokens Per Minute)**: 每分钟 token 数
- **并发数**: 同时进行的请求数

具体限制根据您的套餐而定，超出限制会返回 429 错误。

### 响应头

速率限制信息会在响应头中返回：

```http
X-RateLimit-Limit-Requests: 100
X-RateLimit-Remaining-Requests: 95
X-RateLimit-Reset-Requests: 2024-01-01T00:01:00Z
```

## 代码示例

### Python 示例

```python
import requests
import json

def chat_with_minimax(api_key, group_id, message):
    url = f"https://api.minimax.chat/v1/text/chatcompletion_pro?GroupId={group_id}"
    
    headers = {
        "Authorization": f"Bearer {api_key}",
        "Content-Type": "application/json"
    }
    
    payload = {
        "model": "abab6.5-chat",
        "messages": [
            {"role": "user", "content": message}
        ],
        "stream": False
    }
    
    response = requests.post(url, headers=headers, json=payload)
    
    if response.status_code == 200:
        data = response.json()
        return data['choices'][0]['message']['content']
    else:
        raise Exception(f"API Error: {response.status_code} - {response.text}")

# 使用示例
api_key = "YOUR_API_KEY"
group_id = "YOUR_GROUP_ID"
result = chat_with_minimax(api_key, group_id, "你好")
print(result)
```

### 流式输出示例

```python
import requests
import json

def chat_stream(api_key, group_id, message):
    url = f"https://api.minimax.chat/v1/text/chatcompletion_pro?GroupId={group_id}"
    
    headers = {
        "Authorization": f"Bearer {api_key}",
        "Content-Type": "application/json"
    }
    
    payload = {
        "model": "abab6.5-chat",
        "messages": [
            {"role": "user", "content": message}
        ],
        "stream": True
    }
    
    response = requests.post(url, headers=headers, json=payload, stream=True)
    
    for line in response.iter_lines():
        if line:
            line = line.decode('utf-8')
            if line.startswith('data: '):
                data = line[6:]  # 去掉 'data: ' 前缀
                if data == '[DONE]':
                    break
                try:
                    chunk = json.loads(data)
                    content = chunk['choices'][0]['delta'].get('content', '')
                    if content:
                        print(content, end='', flush=True)
                except json.JSONDecodeError:
                    continue

# 使用示例
chat_stream("YOUR_API_KEY", "YOUR_GROUP_ID", "介绍一下人工智能")
```

### JavaScript/Node.js 示例

```javascript
const fetch = require('node-fetch');

async function chatWithMinimax(apiKey, groupId, message) {
  const url = `https://api.minimax.chat/v1/text/chatcompletion_pro?GroupId=${groupId}`;
  
  const response = await fetch(url, {
    method: 'POST',
    headers: {
      'Authorization': `Bearer ${apiKey}`,
      'Content-Type': 'application/json'
    },
    body: JSON.stringify({
      model: 'abab6.5-chat',
      messages: [
        { role: 'user', content: message }
      ],
      stream: false
    })
  });
  
  if (!response.ok) {
    throw new Error(`API Error: ${response.status}`);
  }
  
  const data = await response.json();
  return data.choices[0].message.content;
}

// 使用示例
chatWithMinimax('YOUR_API_KEY', 'YOUR_GROUP_ID', '你好')
  .then(result => console.log(result))
  .catch(error => console.error(error));
```

## 最佳实践

### 1. 流式输出

对于需要实时反馈的场景，强烈建议使用流式输出：

**优点**：
- 降低首字延迟，用户体验更好
- 可以提前展示部分结果
- 适合聊天机器人等交互场景

**实现要点**：
```python
# 设置 stream=True
payload = {
    "model": "abab6.5-chat",
    "messages": messages,
    "stream": True  # 启用流式输出
}

# 使用 stream=True 接收响应
response = requests.post(url, headers=headers, json=payload, stream=True)
```

### 2. 上下文管理

对于多轮对话，需要合理管理上下文：

```python
class ChatContext:
    def __init__(self, max_tokens=4000):
        self.messages = []
        self.max_tokens = max_tokens
    
    def add_message(self, role, content):
        self.messages.append({"role": role, "content": content})
        # 超出限制时移除旧消息（保留 system 消息）
        self._trim_messages()
    
    def _trim_messages(self):
        # 简单策略：保留系统消息和最近10轮对话
        system_msgs = [m for m in self.messages if m['role'] == 'system']
        other_msgs = [m for m in self.messages if m['role'] != 'system']
        if len(other_msgs) > 20:  # 10轮对话 = 20条消息
            other_msgs = other_msgs[-20:]
        self.messages = system_msgs + other_msgs
```

### 3. 错误重试策略

实现指数退避重试：

```python
import time
from typing import Callable

def retry_with_backoff(func: Callable, max_retries=3, initial_delay=1):
    """使用指数退避的重试装饰器"""
    for attempt in range(max_retries):
        try:
            return func()
        except Exception as e:
            if attempt == max_retries - 1:
                raise e
            delay = initial_delay * (2 ** attempt)
            print(f"请求失败，{delay}秒后重试...")
            time.sleep(delay)
```

### 4. 提示词优化

**清晰的指令**：
```python
# ❌ 不够清晰
"写个故事"

# ✅ 清晰明确
"请写一个关于人工智能的科幻短篇故事，要求：1) 800字左右 2) 情节紧凑 3) 结局出人意料"
```

**使用 system 消息**：
```python
messages = [
    {
        "role": "system",
        "content": "你是一个专业的技术文档撰写助手，擅长用清晰简洁的语言解释复杂的技术概念。"
    },
    {
        "role": "user",
        "content": "解释什么是 Transformer"
    }
]
```

### 5. Token 使用优化

**监控 token 使用**：
```python
response = requests.post(url, headers=headers, json=payload)
data = response.json()

# 获取 token 使用情况
usage = data.get('usage', {})
print(f"输入 tokens: {usage['prompt_tokens']}")
print(f"输出 tokens: {usage['completion_tokens']}")
print(f"总计 tokens: {usage['total_tokens']}")
```

**控制输出长度**：
```python
payload = {
    "model": "abab6.5-chat",
    "messages": messages,
    "max_tokens": 500,  # 限制输出长度
    "tokens_to_generate": 300  # 建议生成长度
}
```

### 6. 安全性建议

**保护 API Key**：
- 不要在代码中硬编码 API Key
- 使用环境变量或配置文件存储
- 定期轮换 API Key

```python
import os

api_key = os.environ.get('MINIMAX_API_KEY')
group_id = os.environ.get('MINIMAX_GROUP_ID')

if not api_key or not group_id:
    raise ValueError("请设置 MINIMAX_API_KEY 和 MINIMAX_GROUP_ID 环境变量")
```

**内容过滤**：
- 启用敏感信息脱敏：`mask_sensitive_info: true`
- 检查响应中的 `input_sensitive` 和 `output_sensitive` 字段

## 多模态能力

MiniMax 还提供其他多模态能力（需单独查看相应文档）：

- **语音合成 (TTS)**: 文本转语音
- **语音识别 (ASR)**: 语音转文本
- **图像生成**: 文生图能力
- **视频生成**: T2V 文生视频

## 计费说明

### 计费方式

- 按照实际使用的 token 数量计费
- 输入 token（prompt_tokens）和输出 token（completion_tokens）**分开计费**
- 不同模型定价不同

### 示例

假设 abab6.5-chat 的价格为：
- 输入：¥0.03 / 1K tokens
- 输出：¥0.06 / 1K tokens

一次对话使用：
- 输入 1000 tokens：¥0.03
- 输出 500 tokens：¥0.03
- **总计**：¥0.06

### 成本优化建议

1. **选择合适的模型**：简单任务使用 abab5.5 系列
2. **控制输出长度**：设置合理的 `max_tokens`
3. **优化提示词**：减少不必要的输入 token
4. **复用上下文**：避免重复发送相同内容

## 参考资源

- **官方文档**: https://platform.minimaxi.com/docs
- **API 参考**: https://platform.minimaxi.com/docs/api-reference/
- **定价信息**: https://platform.minimaxi.com/docs/pricing/overview
- **控制台**: https://platform.minimaxi.com/user-center/basic-information
- **技术支持**: support@minimaxi.com

## 常见问题 (FAQ)

### Q: 如何获取 Group ID？
A: 登录开放平台控制台，在「基本信息」页面可以查看 Group ID。

### Q: API Key 可以重置吗？
A: 可以。在控制台的「API管理」中删除旧的 Key 并创建新的。

### Q: 支持哪些编程语言？
A: MiniMax API 是标准的 HTTP RESTful API，支持所有能发送 HTTP 请求的编程语言。

### Q: 流式输出如何处理错误？
A: 流式输出中如果发生错误，会在 SSE 流中返回错误消息，需要解析每行数据。

### Q: 如何提高响应速度？
A: 1) 使用带 `s` 后缀的速度优化模型 2) 减少输入 token 数量 3) 启用流式输出

## 更新日志

- **2024-01**: 发布 abab6.5 系列模型，支持 245K 超长上下文
- **2024-01**: 推出 ChatCompletion Pro 接口
- **2023-12**: 增加函数调用 (Function Calling) 支持
- **2023-09**: API v2 版本上线
- **2023-06**: abab5.5 系列模型发布
