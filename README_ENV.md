# 环境变量配置说明

## AI 服务相关环境变量

在项目根目录创建 `.env` 文件，添加以下环境变量：

### 必需配置

```bash
# OpenAI API 密钥 (必需)
OPENAI_API_KEY=你的-openai-api-密钥

# 管理员账户配置
ADMIN_EMAIL=admin@easypeek.com
ADMIN_PASSWORD=admin123456
ADMIN_USERNAME=admin
```

### 可选配置

```bash
# AI 服务提供商 (默认: openai)
AI_PROVIDER=openai

# OpenAI API 端点 (默认: https://api.openai.com/v1/chat/completions)
OPENAI_API_ENDPOINT=https://api.openai.com/v1/chat/completions

# OpenAI 模型 (默认: gpt-3.5-turbo)
OPENAI_MODEL=gpt-3.5-turbo
```

## 使用方法

1. 在 `EasyPeek-backend/` 目录下创建 `.env` 文件
2. 将以上配置复制到 `.env` 文件中
3. 将 `你的-openai-api-密钥` 替换为实际的 OpenAI API 密钥
4. 根据需要修改其他配置项

## 注意事项

- `.env` 文件已被加入 `.gitignore`，不会被提交到 git 仓库
- 如果未设置 `OPENAI_API_KEY`，AI 功能将使用模拟模式
- 所有配置都有默认值，只有 `OPENAI_API_KEY` 需要必须设置才能使用真实的 AI 功能