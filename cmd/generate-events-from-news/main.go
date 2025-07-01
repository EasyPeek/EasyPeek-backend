package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/EasyPeek/EasyPeek-backend/internal/config"
	"github.com/EasyPeek/EasyPeek-backend/internal/database"
	"github.com/EasyPeek/EasyPeek-backend/internal/models"
)

// EventCluster 表示一个事件聚类
type EventCluster struct {
	Title        string
	Description  string
	Category     string
	StartTime    time.Time
	EndTime      time.Time
	Location     string
	Status       string
	Tags         []string
	Source       string
	NewsList     []models.News
	HotnessScore float64
}

// EventExport 用于导出的事件结构
type EventExport struct {
	Title        string    `json:"title"`
	Description  string    `json:"description"`
	Content      string    `json:"content"`
	StartTime    time.Time `json:"start_time"`
	EndTime      time.Time `json:"end_time"`
	Location     string    `json:"location"`
	Status       string    `json:"status"`
	Category     string    `json:"category"`
	Tags         string    `json:"tags"`
	Source       string    `json:"source"`
	RelatedLinks string    `json:"related_links"`
	ViewCount    int64     `json:"view_count"`
	LikeCount    int64     `json:"like_count"`
	CommentCount int64     `json:"comment_count"`
	ShareCount   int64     `json:"share_count"`
	HotnessScore float64   `json:"hotness_score"`
	CreatedBy    uint      `json:"created_by"`
	NewsIDs      []uint    `json:"news_ids"`
}

func main() {
	startTime := time.Now()
	fmt.Println("🔄 EasyPeek 从新闻自动生成事件工具")
	fmt.Println("====================================")

	// 加载配置
	cfg, err := config.LoadConfig("internal/config/config.yaml")
	if err != nil {
		log.Fatalf("❌ 无法加载配置: %v", err)
	}

	// 初始化数据库
	if err := database.Initialize(cfg); err != nil {
		log.Fatalf("❌ 无法初始化数据库: %v", err)
	}
	defer database.CloseDatabase()

	db := database.GetDB()

	// 1. 获取所有新闻
	var allNews []models.News
	if err := db.Find(&allNews).Error; err != nil {
		log.Fatalf("❌ 获取新闻列表失败: %v", err)
	}
	fmt.Printf("✅ 获取到 %d 条新闻\n", len(allNews))

	if len(allNews) == 0 {
		log.Fatalf("❌ 数据库中没有新闻，无法生成事件")
	}

	// 2. 根据分类将新闻分组
	categoryNews := make(map[string][]models.News)
	for _, news := range allNews {
		if news.Category == "" {
			news.Category = "未分类"
		}
		categoryNews[news.Category] = append(categoryNews[news.Category], news)
	}
	fmt.Printf("✅ 新闻已按 %d 个分类分组\n", len(categoryNews))

	// 3. 为每个分类生成事件聚类
	allEventClusters := make([]*EventCluster, 0)

	for category, newsList := range categoryNews {
		fmt.Printf("🔄 处理分类 [%s] 的 %d 条新闻...\n", category, len(newsList))

		// 对每个分类内的新闻进行简单聚类
		clusters := clusterNewsByTitle(newsList)
		fmt.Printf("  ✅ 在 [%s] 分类中生成了 %d 个事件聚类\n", category, len(clusters))

		allEventClusters = append(allEventClusters, clusters...)
	}

	// 4. 按热度对事件聚类排序
	sort.Slice(allEventClusters, func(i, j int) bool {
		return allEventClusters[i].HotnessScore > allEventClusters[j].HotnessScore
	})

	// 5. 转换聚类为事件导出格式
	events := make([]EventExport, 0, len(allEventClusters))

	for _, cluster := range allEventClusters {
		event := convertClusterToEvent(cluster)
		events = append(events, event)
	}

	// 6. 将事件导出为JSON文件
	exportEventsToJSON(events)

	elapsed := time.Since(startTime)
	fmt.Println("====================================")
	fmt.Printf("✅ 操作完成! 成功生成 %d 个事件 (耗时: %.2f秒)\n", len(events), elapsed.Seconds())
	fmt.Println("现在您可以运行 import-generated-events 脚本导入生成的事件数据")
}

// clusterNewsByTitle 根据标题相似度简单聚类新闻
func clusterNewsByTitle(newsList []models.News) []*EventCluster {
	// 按发布时间排序
	sort.Slice(newsList, func(i, j int) bool {
		return newsList[i].PublishedAt.Before(newsList[j].PublishedAt)
	})

	clusters := make([]*EventCluster, 0)
	processed := make(map[uint]bool)

	// 对每个未处理的新闻，尝试创建或加入聚类
	for i, news := range newsList {
		if processed[news.ID] {
			continue
		}

		// 创建新聚类
		cluster := &EventCluster{
			Title:        news.Title,
			Description:  news.Summary,
			Category:     news.Category,
			StartTime:    news.PublishedAt,
			EndTime:      news.PublishedAt.Add(24 * time.Hour), // 默认事件持续一天
			Location:     "全国",                                 // 默认位置
			Status:       "进行中",
			Tags:         extractTags(news),
			Source:       news.Source,
			NewsList:     []models.News{news},
			HotnessScore: float64(news.ViewCount + news.LikeCount*2 + news.CommentCount*3 + news.ShareCount*5),
		}

		// 查找相似的新闻加入此聚类
		for j := i + 1; j < len(newsList); j++ {
			if processed[newsList[j].ID] {
				continue
			}

			// 如果标题相似度高，加入聚类
			if areTitlesSimilar(news.Title, newsList[j].Title) {
				cluster.NewsList = append(cluster.NewsList, newsList[j])
				processed[newsList[j].ID] = true

				// 更新聚类信息
				if newsList[j].PublishedAt.Before(cluster.StartTime) {
					cluster.StartTime = newsList[j].PublishedAt
				}
				if newsList[j].PublishedAt.After(cluster.EndTime) {
					cluster.EndTime = newsList[j].PublishedAt
				}

				// 合并标签
				newsTags := extractTags(newsList[j])
				for _, tag := range newsTags {
					if !contains(cluster.Tags, tag) {
						cluster.Tags = append(cluster.Tags, tag)
					}
				}

				// 累加热度分数
				cluster.HotnessScore += float64(newsList[j].ViewCount + newsList[j].LikeCount*2 +
					newsList[j].CommentCount*3 + newsList[j].ShareCount*5)
			}
		}

		// 完善聚类信息
		if len(cluster.Description) == 0 && len(cluster.NewsList) > 0 {
			// 如果没有描述，使用第一条新闻的内容开头作为描述
			for _, n := range cluster.NewsList {
				if len(n.Content) > 10 {
					// 使用内容的前100个字符作为描述
					endPos := int(math.Min(100, float64(len(n.Content))))
					cluster.Description = n.Content[:endPos] + "..."
					break
				}
			}
		}

		// 更新状态
		now := time.Now()
		if cluster.EndTime.Before(now) {
			cluster.Status = "已结束"
		} else if cluster.StartTime.After(now) {
			cluster.Status = "未开始"
		}

		// 将此聚类加入结果
		clusters = append(clusters, cluster)
		processed[news.ID] = true
	}

	return clusters
}

// areTitlesSimilar 检查两个标题是否相似
func areTitlesSimilar(title1, title2 string) bool {
	// 实现简单的标题相似度判断
	title1 = strings.ToLower(title1)
	title2 = strings.ToLower(title2)

	// 如果标题中包含对方的关键词，认为相似
	keywords1 := extractKeywords(title1)
	keywords2 := extractKeywords(title2)

	var matches int
	for _, kw1 := range keywords1 {
		if len(kw1) < 2 {
			continue // 跳过太短的词
		}
		for _, kw2 := range keywords2 {
			if len(kw2) < 2 {
				continue
			}
			if strings.Contains(kw1, kw2) || strings.Contains(kw2, kw1) {
				matches++
			}
		}
	}

	// 根据匹配的关键词数判断相似度
	// 如果有至少2个关键词匹配或匹配率超过40%，认为相似
	threshold := int(math.Min(float64(len(keywords1)), float64(len(keywords2))) * 0.4)
	return matches >= int(math.Max(2, float64(threshold)))
}

// extractKeywords 从标题中提取关键词
func extractKeywords(title string) []string {
	// 简单实现：按空格和标点分割
	title = strings.ToLower(title)
	for _, r := range []string{"，", "。", "？", "！", "、", "：", "；", ",", ".", "?", "!", ":", ";", "'", "\"", "(", ")", "（", "）"} {
		title = strings.ReplaceAll(title, r, " ")
	}

	// 分割并过滤空字符串
	words := strings.Split(title, " ")
	result := make([]string, 0)

	// 过滤掉常见的停用词
	stopWords := map[string]bool{
		"的": true, "了": true, "是": true, "在": true, "有": true, "和": true, "与": true, "为": true,
		"a": true, "an": true, "the": true, "to": true, "of": true, "for": true, "and": true,
		"or": true, "in": true, "on": true, "at": true,
	}

	for _, word := range words {
		word = strings.TrimSpace(word)
		if word != "" && !stopWords[word] {
			result = append(result, word)
		}
	}

	return result
}

// extractTags 从新闻中提取标签
func extractTags(news models.News) []string {
	tags := make([]string, 0)

	// 1. 从新闻的tags字段提取
	if news.Tags != "" {
		// 如果是JSON格式，解析
		if strings.HasPrefix(news.Tags, "[") && strings.HasSuffix(news.Tags, "]") {
			var parsedTags []string
			if err := json.Unmarshal([]byte(news.Tags), &parsedTags); err == nil {
				tags = append(tags, parsedTags...)
			}
		} else {
			// 否则按逗号分割
			for _, tag := range strings.Split(news.Tags, ",") {
				tag = strings.TrimSpace(tag)
				if tag != "" {
					tags = append(tags, tag)
				}
			}
		}
	}

	// 2. 添加分类作为标签
	if news.Category != "" && !contains(tags, news.Category) {
		tags = append(tags, news.Category)
	}

	// 3. 添加来源作为标签
	if news.Source != "" && !contains(tags, news.Source) {
		tags = append(tags, news.Source)
	}

	return tags
}

// contains 检查切片是否包含指定元素
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// convertClusterToEvent 将聚类转换为导出格式的事件
func convertClusterToEvent(cluster *EventCluster) EventExport {
	// 准备相关链接
	links := make([]string, 0)
	newsIDs := make([]uint, 0)
	viewCount := int64(0)
	likeCount := int64(0)
	commentCount := int64(0)
	shareCount := int64(0)

	// 生成内容
	content := fmt.Sprintf("# %s\n\n## 事件概述\n\n%s\n\n## 相关新闻\n\n",
		cluster.Title, cluster.Description)

	for _, news := range cluster.NewsList {
		// 累加统计数据
		viewCount += news.ViewCount
		likeCount += news.LikeCount
		commentCount += news.CommentCount
		shareCount += news.ShareCount

		// 收集新闻ID
		newsIDs = append(newsIDs, news.ID)

		// 添加链接
		if news.Link != "" {
			links = append(links, news.Link)
		}

		// 添加新闻到内容
		content += fmt.Sprintf("### %s\n\n", news.Title)
		if news.Source != "" {
			content += fmt.Sprintf("来源: %s   ", news.Source)
		}
		if !news.PublishedAt.IsZero() {
			content += fmt.Sprintf("发布时间: %s\n\n", news.PublishedAt.Format("2006-01-02 15:04:05"))
		} else {
			content += "\n\n"
		}

		// 添加摘要或内容片段
		if news.Summary != "" {
			content += news.Summary + "\n\n"
		} else if news.Content != "" {
			// 使用内容的前200个字符作为摘要
			endPos := int(math.Min(200, float64(len(news.Content))))
			content += news.Content[:endPos]
			if len(news.Content) > 200 {
				content += "..."
			}
			content += "\n\n"
		}
	}

	// 将标签数组转换为JSON字符串
	tagsJSON, _ := json.Marshal(cluster.Tags)

	// 将链接数组转换为JSON字符串
	linksJSON, _ := json.Marshal(links)

	return EventExport{
		Title:        cluster.Title,
		Description:  cluster.Description,
		Content:      content,
		StartTime:    cluster.StartTime,
		EndTime:      cluster.EndTime,
		Location:     cluster.Location,
		Status:       cluster.Status,
		Category:     cluster.Category,
		Tags:         string(tagsJSON),
		Source:       cluster.Source,
		RelatedLinks: string(linksJSON),
		ViewCount:    viewCount,
		LikeCount:    likeCount,
		CommentCount: commentCount,
		ShareCount:   shareCount,
		HotnessScore: cluster.HotnessScore,
		CreatedBy:    1, // 默认系统管理员ID
		NewsIDs:      newsIDs,
	}
}

// exportEventsToJSON 将事件导出为JSON文件
func exportEventsToJSON(events []EventExport) {
	// 创建导出目录
	exportDir := "exports"
	if err := os.MkdirAll(exportDir, 0755); err != nil {
		log.Fatalf("❌ 创建导出目录失败: %v", err)
	}

	// 创建带时间戳的文件名
	timestamp := time.Now().Format("20060102_150405")
	filename := filepath.Join(exportDir, fmt.Sprintf("generated_events_%s.json", timestamp))

	// 导出对象
	exportData := struct {
		Events        []EventExport `json:"events"`
		TotalCount    int           `json:"total_count"`
		GeneratedTime string        `json:"generated_time"`
	}{
		Events:        events,
		TotalCount:    len(events),
		GeneratedTime: time.Now().Format("2006-01-02 15:04:05"),
	}

	// 转换为JSON
	jsonData, err := json.MarshalIndent(exportData, "", "  ")
	if err != nil {
		log.Fatalf("❌ JSON序列化失败: %v", err)
	}

	// 写入文件
	if err := os.WriteFile(filename, jsonData, 0644); err != nil {
		log.Fatalf("❌ 写入文件失败: %v", err)
	}

	fmt.Printf("✅ 已将 %d 个事件导出到: %s\n", len(events), filename)
}
