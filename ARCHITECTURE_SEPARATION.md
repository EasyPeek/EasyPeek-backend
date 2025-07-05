# RSS与AI功能分离架构

## 分离概述

RSS服务和AI分析功能已完全解耦，各自专注于自己的核心职责：

### RSS服务职责
- **数据获取**: 从RSS源抓取新闻内容
- **数据存储**: 将新闻保存到数据库
- **数据管理**: RSS源的CRUD操作
- **统计信息**: RSS相关的统计数据

### AI服务职责
- **内容分析**: 新闻摘要、关键词提取、情感分析
- **事件分析**: 深度事件分析和影响评估
- **趋势预测**: 基于历史数据的趋势预测
- **批量处理**: 批量分析未处理的新闻

## 架构组件

### 1. RSS服务 (`rss_service.go`)
```go
type RSSService struct {
    db     *gorm.DB
    parser *gofeed.Parser
}
```
- ✅ 移除了 `aiService` 字段
- ✅ 移除了所有AI分析相关方法
- ✅ 专注于RSS数据获取和存储

### 2. AI服务 (`ai_service.go`)
```go
type AIService struct {
    db       *gorm.DB
    provider AIProvider
}
```
- ✅ 包含 `BatchAnalyzeUnprocessedNews()` 方法
- ✅ 包含 `AnalyzeNewsWithRetry()` 方法
- ✅ 专注于AI分析功能

### 3. 新闻AI分析调度器 (`news_analysis_scheduler.go`)
```go
type NewsAnalysisScheduler struct {
    aiService *services.AIService
    ticker    *time.Ticker
    done      chan bool
    running   bool
}
```
- ✅ 独立的调度器负责定时分析
- ✅ 15分钟执行一次批量分析
- ✅ 提供异步分析接口

### 4. API路由更新
- ✅ RSS Handler 移除了 `BatchAnalyzeNews` 方法
- ✅ AI Handler 添加了 `BatchAnalyzeUnprocessedNews` 方法
- ✅ 路由配置更新：
  - `/admin/rss-sources/batch-analyze` → AI Handler
  - `/ai/batch-analyze-unprocessed` → AI Handler

## 数据流程

### RSS数据流程
```
RSS源 → RSS Parser → 数据清理 → 数据库存储 → 完成
```

### AI分析流程
```
定时器触发 → 查询未分析新闻 → AI分析 → 保存分析结果 → 完成
```

### 完整流程
```
1. RSS Scheduler → RSS Service → 保存新闻到数据库
2. News Analysis Scheduler → AI Service → 分析新闻 → 保存分析结果
```

## 优势

1. **职责分离**: 每个服务专注于自己的核心功能
2. **独立扩展**: RSS和AI功能可独立优化和扩展
3. **故障隔离**: 一个服务的故障不会影响另一个服务
4. **测试友好**: 可以独立测试RSS和AI功能
5. **配置灵活**: 可以独立配置RSS抓取频率和AI分析频率

## 配置

### RSS配置
- RSS抓取频率: 由RSS Scheduler控制
- RSS源管理: 通过Admin API

### AI分析配置
- 分析频率: 15分钟 (News Analysis Scheduler)
- API配置: config.yaml中的AI配置
- 批量大小: 每次处理50条新闻

## API端点

### RSS管理 (Admin)
- `GET /admin/rss-sources` - 获取RSS源列表
- `POST /admin/rss-sources` - 创建RSS源
- `POST /admin/rss-sources/batch-analyze` - 触发批量分析 (调用AI Handler)

### AI分析
- `POST /ai/analyze` - 分析单条新闻
- `POST /ai/batch-analyze-unprocessed` - 批量分析未处理新闻
- `GET /ai/stats` - 获取AI分析统计

## 启动日志
```
RSS scheduler is running
News AI Analysis scheduler is running (every 15 minutes)
AI Event generation service is running (every 30 minutes)
```

这样的架构确保了RSS和AI功能的完全解耦，每个组件都有明确的职责边界。 