#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
新闻数据格式转换脚本
将源新闻格式转换为目标格式，支持AI主题分析
"""

import json
import random
import hashlib
import re
from datetime import datetime
from urllib.parse import urlparse
import requests
from typing import Dict, List, Any
source_file="sourcenews.json"
target_file="data/converted_news.json"
class NewsConverter:
    def __init__(self):
        """初始化转换器"""
        self.categories = ["政治", "经济", "社会", "科技", "体育", "娱乐", "国际", "军事", "教育", "健康"]
        self.default_authors = ["记者", "编辑", "通讯员", "特约记者"]
        
    def analyze_category_with_ai(self, title: str, content: str) -> str:
        """使用AI分析文章主题分类"""
        # 关键词映射到分类
        keyword_category_map = {
            "政治": ["政治", "政府", "政策", "领导", "党", "国家", "外交", "会议", "决策", "治理"],
            "经济": ["经济", "股市", "股指", "金融", "投资", "贸易", "GDP", "通胀", "银行", "企业", "上市", "板块", "涨幅", "交易"],
            "科技": ["科技", "AI", "人工智能", "互联网", "5G", "芯片", "创新", "技术", "数字", "智能"],
            "社会": ["社会", "民生", "教育", "医疗", "就业", "住房", "环境", "安全", "文化"],
            "国际": ["国际", "美国", "欧洲", "日本", "韩国", "俄罗斯", "全球", "世界"],
            "军事": ["军事", "国防", "军队", "战争", "武器", "安全"],
            "体育": ["体育", "比赛", "运动", "足球", "篮球", "奥运"],
            "娱乐": ["娱乐", "明星", "电影", "音乐", "综艺", "演出"],
            "健康": ["健康", "医疗", "疫情", "病毒", "治疗", "药物"],
            "教育": ["教育", "学校", "大学", "招生", "考试", "学生"]
        }
        
        text = title + " " + content[:500]  # 取标题和内容前500字分析
        
        # 统计每个分类的关键词出现次数
        category_scores = {}
        for category, keywords in keyword_category_map.items():
            score = 0
            for keyword in keywords:
                score += text.count(keyword)
            if score > 0:
                category_scores[category] = score
        
        # 返回得分最高的分类，如果没有匹配则返回"社会"
        if category_scores:
            return max(category_scores.items(), key=lambda x: x[1])[0]
        else:
            return "社会"
    
    def generate_summary(self, content: str, max_length: int = 200) -> str:
        """生成摘要"""
        # 清理内容
        clean_content = re.sub(r'\s+', '', content)
        clean_content = re.sub(r'【.*?】', '', clean_content)
        
        # 按句号分割，取前几句作为摘要
        sentences = re.split(r'[。！？]', clean_content)
        summary = ""
        for sentence in sentences:
            if len(summary + sentence) <= max_length and sentence.strip():
                summary += sentence + "。"
            else:
                break
        
        return summary.strip() if summary else content[:max_length] + "..."
    
    def generate_guid(self, url: str, title: str) -> str:
        """生成GUID"""
        unique_string = url + title + str(datetime.now().timestamp())
        return hashlib.md5(unique_string.encode('utf-8')).hexdigest()
    
    def parse_datetime(self, date_string: str) -> str:
        """解析日期时间字符串"""
        try:
            # 处理不同的日期格式
            if "年" in date_string and "月" in date_string and "日" in date_string:
                # 格式：2025年07月01日19:01
                date_part = re.search(r'(\d{4})年(\d{1,2})月(\d{1,2})日(\d{1,2}):(\d{1,2})', date_string)
                if date_part:
                    year, month, day, hour, minute = date_part.groups()
                    return f"{year}-{month.zfill(2)}-{day.zfill(2)} {hour.zfill(2)}:{minute.zfill(2)}:00"
            
            # 如果无法解析，返回当前时间
            return datetime.now().strftime("%Y-%m-%d %H:%M:%S")
        except:
            return datetime.now().strftime("%Y-%m-%d %H:%M:%S")
    
    def extract_domain_name(self, url: str) -> str:
        """从URL提取域名作为来源"""
        try:
            parsed = urlparse(url)
            domain = parsed.netloc
            # 简化域名显示
            if domain.startswith('www.'):
                domain = domain[4:]
            return domain
        except:
            return "未知来源"
    
    def generate_random_stats(self) -> Dict[str, int]:
        """生成随机的统计数据"""
        return {
            "view_count": random.randint(100, 10000),
            "like_count": random.randint(10, 1000),
            "comment_count": random.randint(5, 500),
            "share_count": random.randint(0, 200)
        }
    
    def calculate_hotness_score(self, stats: Dict[str, int]) -> float:
        """计算热度分数"""
        # 简单的热度计算公式
        score = (stats["view_count"] * 0.1 + 
                stats["like_count"] * 2 + 
                stats["comment_count"] * 3 + 
                stats["share_count"] * 5)
        return round(score, 1)
    
    def convert_news_item(self, source_item: Dict[str, Any]) -> Dict[str, Any]:
        """转换单个新闻项"""
        # 提取源数据
        title = source_item.get("标题", "")
        content = source_item.get("正文", "")
        source_name = source_item.get("来源", "")
        url = source_item.get("页面网址", "")
        publish_time = source_item.get("发布时间", "")
        
        # 检查正文是否为空，如果为空则跳过
        if not content or content.strip() == "":
            raise ValueError("正文为空，跳过此条新闻")
        
        # AI分析分类
        category = self.analyze_category_with_ai(title, content)
        
        # 生成摘要
        summary = self.generate_summary(content)
        description = summary  # 描述和摘要相同
        
        # 处理来源
        if not source_name:
            source_name = self.extract_domain_name(url)
        
        # 生成随机统计数据
        stats = self.generate_random_stats()
        hotness_score = self.calculate_hotness_score(stats)
        
        # 构建目标格式
        converted_item = {
            "title": title,
            "content": content,
            "summary": summary,
            "description": description,
            "source": source_name,
            "category": category,
            "published_at": self.parse_datetime(publish_time),
            "created_by": None,
            "is_active": True,
            "source_type": "manual",
            "rss_source_id": None,
            "link": url,
            "guid": self.generate_guid(url, title),
            "author": random.choice(self.default_authors),
            "image_url": "",  # 默认为空，可以后续添加图片提取逻辑
            "tags": json.dumps([category, category, "热点"], ensure_ascii=False),
            "language": "zh",
            "view_count": stats["view_count"],
            "like_count": stats["like_count"],
            "comment_count": stats["comment_count"],
            "share_count": stats["share_count"],
            "hotness_score": hotness_score,
            "status": "published",
            "is_processed": True
        }
        
        return converted_item
    
    def convert_file(self, source_file: str, target_file: str):
        """转换整个文件"""
        try:
            # 读取源文件
            with open(source_file, 'r', encoding='utf-8') as f:
                source_data = json.load(f)
            
            # 确保源数据是列表格式
            if isinstance(source_data, dict):
                source_data = [source_data]
            elif not isinstance(source_data, list):
                raise ValueError("源数据格式不正确，应该是JSON对象或数组")
            
            # 转换数据
            converted_data = []
            skipped_count = 0
            for i, item in enumerate(source_data):
                try:
                    converted_item = self.convert_news_item(item)
                    converted_data.append(converted_item)
                    print(f"✓ 转换成功: {converted_item['title']}")
                except ValueError as e:
                    if "正文为空" in str(e):
                        title = item.get("标题", f"第{i+1}条新闻")
                        print(f"⚠ 跳过: {title} (正文为空)")
                        skipped_count += 1
                    else:
                        print(f"✗ 转换失败: {e}")
                    continue
                except Exception as e:
                    title = item.get("标题", f"第{i+1}条新闻")
                    print(f"✗ 转换失败 [{title}]: {e}")
                    continue
            
            # 保存转换后的数据
            with open(target_file, 'w', encoding='utf-8') as f:
                json.dump(converted_data, f, ensure_ascii=False, indent=4)
            
            print(f"\n转换完成！")
            print(f"源文件: {source_file}")
            print(f"目标文件: {target_file}")
            print(f"总计处理: {len(source_data)} 条新闻")
            print(f"成功转换: {len(converted_data)} 条新闻")
            if skipped_count > 0:
                print(f"跳过空文: {skipped_count} 条新闻")
            if len(source_data) - len(converted_data) - skipped_count > 0:
                print(f"转换失败: {len(source_data) - len(converted_data) - skipped_count} 条新闻")
            
        except FileNotFoundError:
            print(f"错误: 找不到源文件 {source_file}")
        except json.JSONDecodeError:
            print(f"错误: 源文件 {source_file} 不是有效的JSON格式")
        except Exception as e:
            print(f"转换过程中发生错误: {e}")

def main():
    """主函数"""
    converter = NewsConverter()
    
    # 默认文件路径
    source_file = "sourcenews.json"
    target_file = "data/converted_news.json"
    
    print("=== 新闻数据格式转换工具 ===")
    print(f"源文件: {source_file}")
    print(f"目标文件: {target_file}")
    print("开始转换...")
    
    converter.convert_file(source_file, target_file)

if __name__ == "__main__":
    main() 