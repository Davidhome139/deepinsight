# Git Pre-Commit 敏感信息检测

这是一个用于在 Git 提交前自动检测敏感信息的工具集。它可以帮助防止 API 密钥、密码、令牌等敏感信息意外提交到代码仓库。

## 功能特性

- 检测多种敏感信息模式：
  - API 密钥和 Secret Key
  - 密码
  - 认证令牌（JWT、GitHub Token 等）
  - 数据库连接 URL
  - JWT 密钥
  - AWS 凭证
  - AI 服务提供商凭证（阿里云、DeepSeek、MiniMax、腾讯等）
  - 私钥文件
- 自动在 Git 提交前运行检查
- 可手动运行检测
- 支持排除特定文件和目录

## 安装

### 前置要求

- Python 3.6 或更高版本
- Git

### 安装 Git Hooks

#### Windows 用户

双击运行：
```
scripts\pre-commit\install-hooks.bat
```

或者在 PowerShell 中运行：
```powershell
.\scripts\pre-commit\install-hooks.bat
```

#### Linux/Mac 用户

```bash
chmod +x scripts/pre-commit/install-hooks.sh
./scripts/pre-commit/install-hooks.sh
```

## 使用方法

### 自动检测（推荐）

安装 hooks 后，每次执行 `git commit` 时，检测会自动运行。如果检测到敏感信息，提交将被阻止。

### 手动检测

#### 检测暂存区的文件（模拟 pre-commit）

```bash
python scripts/pre-commit/detect-secrets.py --staged
```

#### 检测指定文件

```bash
python scripts/pre-commit/detect-secrets.py path/to/file1.py path/to/file2.yaml
```

#### 检测整个仓库

```bash
python scripts/pre-commit/detect-secrets.py --all
```

### 运行测试

```bash
python scripts/pre-commit/test-detector.py
```

## 配置

### 排除文件

编辑 `scripts/pre-commit/detect-secrets.py` 中的 `EXCLUDED_FILES` 和 `EXCLUDED_EXTENSIONS` 列表来添加或修改排除规则。

### 添加新的检测模式

在 `SENSITIVE_PATTERNS` 列表中添加新的正则表达式模式。

## 卸载

删除 `.git/hooks/pre-commit` 文件即可卸载钩子：

```bash
rm .git/hooks/pre-commit
```

或者在 Windows 中：
```cmd
del .git\hooks\pre-commit
```

## 文件说明

- `detect-secrets.py` - 主要的检测脚本
- `pre-commit` - Git pre-commit 钩子
- `install-hooks.sh` - Linux/Mac 安装脚本
- `install-hooks.bat` - Windows 安装脚本
- `test-samples.txt` - 测试样本文件
- `test-detector.py` - 测试脚本
- `README.md` - 本文档

## 注意事项

- 此工具不能保证 100% 检测到所有敏感信息，请保持良好的安全习惯
- 始终使用环境变量或安全的密钥管理系统来存储敏感信息
- 定期审查代码仓库以确保没有敏感信息泄露
- 即使使用了此工具，也不要在代码中硬编码真实的密钥

## 故障排除

### Python 未找到

确保 Python 3 已安装并添加到系统 PATH 中。

### 钩子没有运行

确保钩子文件有执行权限（Linux/Mac）：
```bash
chmod +x .git/hooks/pre-commit
```

## 许可证

本项目遵循主项目的许可证。
