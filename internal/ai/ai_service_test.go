package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/EasyPeek/EasyPeek-backend/internal/config"
	"github.com/EasyPeek/EasyPeek-backend/internal/models"
	"github.com/sashabaranov/go-openai"
	"github.com/stretchr/testify/assert"
)

func TestSummarizeNews(t *testing.T) {
	// 1. 创建一个模拟服务器
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 验证请求是否正确 (可选)
		assert.Equal(t, "/chat/completions", r.URL.Path)

		// 2. 准备一个预定义的成功响应
		resp := openai.ChatCompletionResponse{
			Choices: []openai.ChatCompletionChoice{
				{
					Message: openai.ChatCompletionMessage{
						Content: "这是一个模拟的新闻摘要。",
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer mockServer.Close()

	// 3. 配置 AIService 使用模拟服务器的地址
	cfg := &config.Config{
		OpenRouter: config.OpenRouterConfig{
			APIKey:  "dummy-key",
			APIHost: mockServer.URL, // 关键点：指向模拟服务器
			Model:   "test-model",
		},
	}
	aiService, err := NewAIService(cfg)
	assert.NoError(t, err)

	// 4. 执行需要测试的函数
	news := &models.News{Title: "测试标题", Content: "测试内容"}
	summary, err := aiService.SummarizeNews(context.Background(), news)

	// 5. 断言结果是否符合预期
	assert.NoError(t, err)
	assert.Equal(t, "这是一个模拟的新闻摘要。", summary)
}

// 你还可以添加测试 API 返回错误的情况
func TestSummarizeNews_APIError(t *testing.T) {
    // ... 设置一个返回 500 错误的模拟服务器 ...
    // ... 断言 aiService.SummarizeNews 返回一个错误 ...
}