package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

// AIAnalysisType AI分析类型
type AIAnalysisType string

const (
	AIAnalysisTypeNews  AIAnalysisType = "news"  // 新闻分析
	AIAnalysisTypeEvent AIAnalysisType = "event" // 事件分析
)

// AnalysisStep 分析步骤（用于展示推理过程）
type AnalysisStep struct {
	Step        int     `json:"step"`        // 步骤序号
	Title       string  `json:"title"`       // 步骤标题
	Description string  `json:"description"` // 步骤描述
	Result      string  `json:"result"`      // 步骤结果
	Confidence  float64 `json:"confidence"`  // 置信度
}

// AnalysisSteps 分析步骤集合
type AnalysisSteps []AnalysisStep

// Value 实现 driver.Valuer 接口
func (a AnalysisSteps) Value() (driver.Value, error) {
	return json.Marshal(a)
}

// Scan 实现 sql.Scanner 接口
func (a *AnalysisSteps) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, a)
	case string:
		return json.Unmarshal([]byte(v), a)
	default:
		return nil
	}
}

// TrendPrediction 趋势预测
type TrendPrediction struct {
	Timeframe   string   `json:"timeframe"`   // 时间范围（短期/中期/长期）
	Trend       string   `json:"trend"`       // 趋势描述
	Probability float64  `json:"probability"` // 概率
	Factors     []string `json:"factors"`     // 影响因素
}

// TrendPredictions 趋势预测集合
type TrendPredictions []TrendPrediction

// Value 实现 driver.Valuer 接口
func (t TrendPredictions) Value() (driver.Value, error) {
	return json.Marshal(t)
}

// Scan 实现 sql.Scanner 接口
func (t *TrendPredictions) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, t)
	case string:
		return json.Unmarshal([]byte(v), t)
	default:
		return nil
	}
}

// AIAnalysis AI分析结果模型
type AIAnalysis struct {
	ID       uint           `json:"id" gorm:"primaryKey"`
	Type     AIAnalysisType `json:"type" gorm:"type:varchar(20);not null;index"` // 分析类型
	TargetID uint           `json:"target_id" gorm:"not null;index"`             // 目标ID（新闻ID或事件ID）

	// 基础分析结果
	Summary        string  `json:"summary" gorm:"type:text"`          // 摘要
	Keywords       string  `json:"keywords" gorm:"type:text"`         // 关键词（JSON数组）
	Sentiment      string  `json:"sentiment" gorm:"type:varchar(20)"` // 情感分析（positive/negative/neutral）
	SentimentScore float64 `json:"sentiment_score" gorm:"default:0"`  // 情感分数

	// 进阶分析结果
	EventAnalysis    string           `json:"event_analysis" gorm:"type:text"`    // 事件分析
	TrendPredictions TrendPredictions `json:"trend_predictions" gorm:"type:json"` // 趋势预测
	AnalysisSteps    AnalysisSteps    `json:"analysis_steps" gorm:"type:json"`    // 分析步骤（推理过程）

	// 影响力评估
	ImpactLevel string  `json:"impact_level" gorm:"type:varchar(20)"` // 影响级别（high/medium/low）
	ImpactScore float64 `json:"impact_score" gorm:"default:0"`        // 影响力分数
	ImpactScope string  `json:"impact_scope" gorm:"type:text"`        // 影响范围描述

	// 相关性分析
	RelatedTopics string `json:"related_topics" gorm:"type:text"` // 相关话题（JSON数组）
	RelatedEvents string `json:"related_events" gorm:"type:text"` // 相关事件ID（JSON数组）

	// AI模型信息
	ModelName    string  `json:"model_name" gorm:"type:varchar(50)"`    // 使用的AI模型名称
	ModelVersion string  `json:"model_version" gorm:"type:varchar(20)"` // 模型版本
	Confidence   float64 `json:"confidence" gorm:"default:0"`           // 整体置信度

	// 状态和时间
	Status         string `json:"status" gorm:"type:varchar(20);default:'completed'"` // 分析状态
	ProcessingTime int    `json:"processing_time" gorm:"default:0"`                   // 处理时间（秒）

	// GORM 时间戳
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

// TableName 指定表名
func (AIAnalysis) TableName() string {
	return "ai_analyses"
}

// AIAnalysisResponse API响应结构
type AIAnalysisResponse struct {
	ID               uint              `json:"id"`
	Type             AIAnalysisType    `json:"type"`
	TargetID         uint              `json:"target_id"`
	Summary          string            `json:"summary"`
	Keywords         []string          `json:"keywords"`
	Sentiment        string            `json:"sentiment"`
	SentimentScore   float64           `json:"sentiment_score"`
	EventAnalysis    string            `json:"event_analysis,omitempty"`
	TrendPredictions []TrendPrediction `json:"trend_predictions,omitempty"`
	AnalysisSteps    []AnalysisStep    `json:"analysis_steps,omitempty"`
	ImpactLevel      string            `json:"impact_level"`
	ImpactScore      float64           `json:"impact_score"`
	ImpactScope      string            `json:"impact_scope"`
	RelatedTopics    []string          `json:"related_topics,omitempty"`
	RelatedEvents    []uint            `json:"related_events,omitempty"`
	ModelName        string            `json:"model_name"`
	ModelVersion     string            `json:"model_version"`
	Confidence       float64           `json:"confidence"`
	Status           string            `json:"status"`
	ProcessingTime   int               `json:"processing_time"`
	CreatedAt        time.Time         `json:"created_at"`
	UpdatedAt        time.Time         `json:"updated_at"`
}

// ToResponse 转换为响应结构
func (a *AIAnalysis) ToResponse() AIAnalysisResponse {
	var keywords []string
	if a.Keywords != "" {
		json.Unmarshal([]byte(a.Keywords), &keywords)
	}

	var relatedTopics []string
	if a.RelatedTopics != "" {
		json.Unmarshal([]byte(a.RelatedTopics), &relatedTopics)
	}

	var relatedEvents []uint
	if a.RelatedEvents != "" {
		json.Unmarshal([]byte(a.RelatedEvents), &relatedEvents)
	}

	return AIAnalysisResponse{
		ID:               a.ID,
		Type:             a.Type,
		TargetID:         a.TargetID,
		Summary:          a.Summary,
		Keywords:         keywords,
		Sentiment:        a.Sentiment,
		SentimentScore:   a.SentimentScore,
		EventAnalysis:    a.EventAnalysis,
		TrendPredictions: a.TrendPredictions,
		AnalysisSteps:    a.AnalysisSteps,
		ImpactLevel:      a.ImpactLevel,
		ImpactScore:      a.ImpactScore,
		ImpactScope:      a.ImpactScope,
		RelatedTopics:    relatedTopics,
		RelatedEvents:    relatedEvents,
		ModelName:        a.ModelName,
		ModelVersion:     a.ModelVersion,
		Confidence:       a.Confidence,
		Status:           a.Status,
		ProcessingTime:   a.ProcessingTime,
		CreatedAt:        a.CreatedAt,
		UpdatedAt:        a.UpdatedAt,
	}
}

// AIAnalysisRequest AI分析请求
type AIAnalysisRequest struct {
	Type     AIAnalysisType `json:"type" binding:"required,oneof=news event"`
	TargetID uint           `json:"target_id" binding:"required"`
	Options  struct {
		EnableSummary     bool `json:"enable_summary"`      // 启用摘要
		EnableKeywords    bool `json:"enable_keywords"`     // 启用关键词提取
		EnableSentiment   bool `json:"enable_sentiment"`    // 启用情感分析
		EnableTrends      bool `json:"enable_trends"`       // 启用趋势预测
		EnableImpact      bool `json:"enable_impact"`       // 启用影响力评估
		ShowAnalysisSteps bool `json:"show_analysis_steps"` // 显示分析步骤
	} `json:"options"`
}
