# 事件生成功能说明

## 概述

EasyPeek 后端现在支持从新闻数据自动生成事件功能。该功能会分析现有的新闻数据，通过智能聚类算法将相似的新闻归类，并自动生成对应的事件，同时建立新闻与事件的关联关系。

**架构设计**: 事件生成的核心逻辑完全在服务层 (`services/event_service.go`) 中实现，确保了业务逻辑的复用性和可测试性。

## 功能特性

- **智能聚类**: 基于新闻标题相似度进行聚类，支持地域和主题关键词匹配
- **地域分类**: 自动识别并聚合地域相关新闻（如"中东形势"、"俄乌冲突"等）
- **主题分类**: 按经济、科技、政治、军事等主题进行智能分类
- **自动关联**: 生成事件的同时自动更新新闻的 `belonged_event_id` 字段
- **热度计算**: 基于新闻的浏览量、点赞数、评论数、分享数计算事件热度
- **分类处理**: 按新闻分类分组处理，提高聚类效果
- **内容生成**: 自动生成事件的详细内容，包含相关新闻信息
- **服务层架构**: 核心逻辑在服务层，支持多种调用方式
- **高聚合度**: 放宽相似性标准，提高事件聚合程度

## 使用方法

### 1. API 调用

通过 HTTP API 调用事件生成功能：

```bash
POST /api/v1/events/generate
Authorization: Bearer <admin_token>
```

响应示例：

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "generated_events": [
      {
        "id": 123,
        "title": "中东地区最新形势发展",
        "category": "国际",
        "hotness_score": 85.5,
        // ... 其他事件字段
      }
    ],
    "total_events": 15,
    "processed_news": 120,
    "generation_time": "2024-01-01T12:00:00Z",
    "elapsed_time": "2.5s",
    "category_breakdown": {
      "国际": 5,
      "科技": 8,
      "社会": 2
    }
  }
}
```

### 2. 脚本工具

使用 `scripts` 目录下的脚本工具：

```bash
# 进入项目目录
cd EasyPeek-backend

# 运行事件生成脚本
go run scripts/generate_events_from_news.go
```

脚本会调用服务层的核心逻辑，并提供详细的执行日志和统计信息。

### 3. 服务层调用

在代码中直接调用服务层方法：

```go
package main

import (
    "github.com/EasyPeek/EasyPeek-backend/internal/services"
)

func main() {
    // 核心逻辑都在服务层
    eventService := services.NewEventService()
    
    result, err := eventService.GenerateEventsFromNews()
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("生成了 %d 个事件\n", result.TotalEvents)
}
```

## 核心组件

### 1. EventService.GenerateEventsFromNews()

**位置**: `internal/services/event_service.go`

主要的事件生成方法，包含以下步骤：

1. 获取所有新闻数据
2. 按分类分组新闻
3. 对每个分类进行聚类分析
4. 创建事件并保存到数据库
5. 更新新闻的事件关联

### 2. 增强聚类算法

**实现位置**: 服务层中的辅助方法

#### 地域关键词匹配
支持按地域自动聚合相关新闻，包括：
- **中东**: 以色列、巴勒斯坦、伊朗、叙利亚等
- **俄乌**: 俄罗斯、乌克兰、普京、泽连斯基等  
- **朝鲜半岛**: 朝鲜、韩国、金正恩等
- **南海**: 台海、台湾、南沙、西沙等
- **欧洲**: 欧盟、英国、法国、德国等
- **美洲**: 美国、加拿大、墨西哥、巴西等
- **其他地域**: 非洲、东南亚、南亚等

#### 主题关键词匹配
支持按主题自动聚合相关新闻，包括：
- **经济**: 经济、GDP、通胀、利率、股市等
- **科技**: 科技、AI、人工智能、5G、芯片等
- **政治**: 政治、选举、总统、首相、政府等
- **军事**: 军事、军队、武器、导弹、战机等
- **能源**: 能源、石油、天然气、电力、核能等
- **环境**: 环境、气候、碳排放、全球变暖等
- **其他主题**: 健康、教育、体育、文化等

#### 相似度算法
- **降低阈值**: 相似度阈值降至20%，提高聚合程度
- **关键词匹配**: 支持包含关系和编辑距离相似
- **时间因素**: 考虑新闻发布时间，确定事件的开始和结束时间
- **热度计算**: 综合考虑浏览量、点赞、评论、分享等指标

### 3. 新闻服务扩展

**位置**: `internal/services/news_service.go`

新增了以下方法处理新闻与事件的关联：

- `UpdateNewsEventAssociation()`: 批量更新新闻的事件关联
- `GetNewsByEventID()`: 根据事件ID获取相关新闻
- `GetUnlinkedNews()`: 获取未关联事件的新闻

## 聚类效果示例

### 地域聚合示例
```
事件标题: "中东地区最新形势发展"
包含新闻:
- 以色列与巴勒斯坦最新谈判进展
- 伊朗核协议谈判取得突破
- 沙特阿拉伯宣布新经济政策
- 黎巴嫩政局最新变化
```

### 主题聚合示例  
```
事件标题: "人工智能技术发展动态"
包含新闻:
- OpenAI发布最新AI模型
- 科技巨头投资人工智能研究
- AI技术在医疗领域的应用突破
- 人工智能监管政策最新进展
```

## 数据库变更

确保 `news` 表包含 `belonged_event_id` 字段：

```sql
ALTER TABLE news ADD COLUMN belonged_event_id INTEGER REFERENCES events(id);
CREATE INDEX idx_news_belonged_event_id ON news(belonged_event_id);
```

## 权限要求

- API 调用需要管理员权限
- 脚本工具需要数据库访问权限
- 服务层调用需要正确的数据库配置

## 注意事项

1. **性能考虑**: 大量新闻数据可能需要较长处理时间
2. **重复执行**: 多次执行可能产生重复事件，建议在执行前清理或检查
3. **数据质量**: 新闻数据的质量直接影响生成事件的质量
4. **聚类参数**: 已优化相似度阈值，提高聚合程度
5. **服务层优势**: 核心逻辑在服务层，便于单元测试和功能扩展
6. **地域识别**: 算法能够自动识别地域相关性，提高聚合效果

## 扩展功能

### 定制聚类算法

可以通过修改服务层中的相关方法来调整聚类算法：

```go
// 在 internal/services/event_service.go 中修改地域关键词
func (s *EventService) hasRegionalMatch(title1, title2 string) bool {
    // 添加新的地域关键词组
    regionalGroups := map[string][]string{
        "新地域": {"关键词1", "关键词2", "关键词3"},
        // ... 其他地域
    }
    // ... 其他逻辑
}

// 修改主题关键词
func (s *EventService) hasTopicMatch(title1, title2 string) bool {
    // 添加新的主题关键词组
    topicGroups := map[string][]string{
        "新主题": {"关键词1", "关键词2", "关键词3"},
        // ... 其他主题
    }
    // ... 其他逻辑
}

// 调整相似度阈值
func (s *EventService) areTitlesSimilar(title1, title2 string) bool {
    // 调整相似度阈值（当前为20%）
    minKeywords := int(math.Max(1, float64(totalKeywords)*0.1)) // 降低到10%
    return matches >= minKeywords
}
```

### 添加过滤条件

可以在服务层的生成方法中添加新闻过滤条件：

```go
// 在 GenerateEventsFromNews 方法中添加过滤逻辑
// 只处理特定时间范围的新闻
// 只处理特定分类的新闻
// 排除已关联事件的新闻
```

### 定时任务集成

由于核心逻辑在服务层，可以轻松集成到定时任务中：

```go
// 定时任务示例
func scheduleEventGeneration() {
    eventService := services.NewEventService()
    
    // 每天凌晨2点执行
    c := cron.New()
    c.AddFunc("0 2 * * *", func() {
        result, err := eventService.GenerateEventsFromNews()
        if err != nil {
            log.Printf("定时生成事件失败: %v", err)
            return
        }
        log.Printf("定时生成了 %d 个事件", result.TotalEvents)
    })
    c.Start()
}
```

## 监控和日志

系统会记录以下信息：

- 处理的新闻数量
- 生成的事件数量  
- 处理时间
- 分类统计
- 地域匹配统计
- 主题匹配统计
- 错误信息

**日志位置**:
- API调用: HTTP访问日志
- 脚本执行: 控制台输出
- 服务层: 应用程序日志

## 故障排除

### 常见问题

1. **数据库连接失败**: 检查数据库配置和连接
2. **权限不足**: 确保用户具有管理员权限
3. **内存不足**: 大量数据处理可能需要更多内存
4. **生成事件为空**: 检查新闻数据是否存在且格式正确
5. **聚合效果不佳**: 检查关键词库是否需要更新

### 调试技巧

1. 启用详细日志
2. 检查数据库中的新闻数据
3. 验证聚类算法的效果
4. 监控服务器资源使用情况
5. 使用脚本工具进行调试（输出更详细）
6. 检查地域和主题关键词匹配情况

## 文件结构

```
EasyPeek-backend/
├── internal/
│   └── services/
│       ├── event_service.go      # 核心业务逻辑（含增强聚类算法）
│       └── news_service.go       # 新闻服务扩展
├── internal/api/
│   ├── event_handler.go         # API端点处理
│   └── router.go                # 路由配置
├── scripts/
│   └── generate_events_from_news.go  # 执行脚本
└── EVENT_GENERATION.md         # 本文档
``` 