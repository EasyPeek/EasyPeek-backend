package ai

import (
	"context"
	"fmt"

	"github.com/EasyPeek/EasyPeek-backend/internal/config"
	"github.com/EasyPeek/EasyPeek-backend/internal/models"
	"github.com/sashabaranov/go-openai"
)

// AIService provides methods for interacting with AI models.
type AIService struct {
	client *openai.Client
	model  string
}

// NewAIService creates a new AIService.
func NewAIService(cfg *config.Config) (*AIService, error) {
	openaiConfig := openai.DefaultConfig(cfg.OpenRouter.APIKey)
	openaiConfig.BaseURL = cfg.OpenRouter.APIHost

	client := openai.NewClientWithConfig(openaiConfig)

	return &AIService{
		client: client,
		model:  cfg.OpenRouter.Model,
	}, nil
}

// SummarizeNews generates a summary for the given news content.
func (s *AIService) SummarizeNews(ctx context.Context, news *models.News) (string, error) {
	prompt := fmt.Sprintf("请为以下新闻生成一个简洁的摘要，不超过200字：\n\n标题：%s\n内容：%s", news.Title, news.Content)

	resp, err := s.client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model: s.model,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: prompt,
				},
			},
		},
	)

	if err != nil {
		return "", fmt.Errorf("failed to call OpenRouter API: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no summary generated from OpenRouter")
	}

	return resp.Choices[0].Message.Content, nil
}