#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
将 localization.json 文件转换为 news 表格式的数据
自动补全缺失字段并生成随机数据
"""

import json
import random
import uuid
from datetime import datetime, timedelta
from typing import List, Dict, Any
import re
import hashlib

class NewsDataConverter:
    def __init__(self):
        self.status_options = ["published", "draft", "archived"]
        self.language_options = ["zh", "en", "zh-CN", "zh-TW"]
        self.source_types = ["manual", "rss"]
        self.categories = [
            "政治", "经济", "科技", "体育", "娱乐", "社会", "国际", 
            "军事", "教育", "健康", "环境", "文化", "旅游", "财经"
        ]
        
    def parse_datetime(self, date_str: str) -> str:
        """解析发布时间字符串并转换为标准格式"""
        try:
            # 处理特殊格式：varauthor='';if(author==""){document.getElementById('author').style.display='none';}中央广电总台国际在线2025年06月25日20:50
            if "document.getElementById" in date_str:
                # 提取日期部分
                date_match = re.search(r'(\d{4}年\d{2}月\d{2}日\d{2}:\d{2})', date_str)
                if date_match:
                    date_str = date_match.group(1)
            
            # 标准格式：2025年06月27日19:38:27
            if re.match(r'\d{4}年\d{2}月\d{2}日\d{2}:\d{2}:\d{2}', date_str):
                return datetime.strptime(date_str, '%Y年%m月%d日%H:%M:%S').strftime('%Y-%m-%d %H:%M:%S')
            
            # 简化格式：2025年06月25日20:50
            if re.match(r'\d{4}年\d{2}月\d{2}日\d{2}:\d{2}', date_str):
                return datetime.strptime(date_str, '%Y年%m月%d日%H:%M').strftime('%Y-%m-%d %H:%M:%S')
            
            # 如果无法解析，使用当前时间减去随机天数
            days_ago = random.randint(1, 30)
            return (datetime.now() - timedelta(days=days_ago)).strftime('%Y-%m-%d %H:%M:%S')
            
        except Exception as e:
            print(f"日期解析错误: {date_str}, 错误: {e}")
            # 生成随机时间
            days_ago = random.randint(1, 30)
            return (datetime.now() - timedelta(days=days_ago)).strftime('%Y-%m-%d %H:%M:%S')
    
    def generate_guid(self, title: str, source: str) -> str:
        """生成唯一的GUID"""
        content = f"{title}_{source}_{datetime.now().isoformat()}"
        return hashlib.md5(content.encode('utf-8')).hexdigest()
    
    def generate_random_stats(self) -> Dict[str, int]:
        """生成随机的统计数据"""
        base_views = random.randint(100, 10000)
        return {
            "view_count": base_views,
            "like_count": random.randint(int(base_views * 0.01), int(base_views * 0.1)),
            "comment_count": random.randint(0, int(base_views * 0.05)),
            "share_count": random.randint(0, int(base_views * 0.02)),
        }
    
    def calculate_hotness_score(self, stats: Dict[str, int], published_at: str) -> float:
        """计算热度分值"""
        try:
            pub_date = datetime.strptime(published_at, '%Y-%m-%d %H:%M:%S')
            days_old = (datetime.now() - pub_date).days
            
            # 基础分值计算
            base_score = (
                stats["view_count"] * 1.0 +
                stats["like_count"] * 5.0 +
                stats["comment_count"] * 3.0 +
                stats["share_count"] * 10.0
            )
            
            # 时间衰减因子
            time_factor = max(0.1, 1.0 - (days_old * 0.1))
            
            return round(base_score * time_factor, 2)
        except:
            return round(random.uniform(10.0, 1000.0), 2)
    
    def extract_tags(self, keyword: str, category: str) -> str:
        """提取并生成标签"""
        tags = []
        if keyword:
            tags.append(keyword)
        if category:
            tags.append(category)
        
        # 添加一些随机标签
        additional_tags = ["热点", "新闻", "重要", "关注"]
        tags.extend(random.sample(additional_tags, k=random.randint(0, 2)))
        
        return json.dumps(tags, ensure_ascii=False)
    
    def clean_text(self, text: str) -> str:
        """清理文本内容"""
        if not text:
            return ""
        
        # 移除多余的换行和空格
        text = re.sub(r'\r\n|\r|\n', '\n', text)
        text = re.sub(r'\n+', '\n', text)
        text = text.strip()
        
        return text
    
    def convert_news_item(self, item: Dict[str, Any]) -> Dict[str, Any]:
        """转换单个新闻条目"""
        # 获取原始数据
        keyword = item.get("关键词", "")
        title = item.get("标题", "")
        image_url = item.get("图片", "")
        summary = item.get("简介", "")
        source = item.get("来源", "")
        published_time = item.get("发布时间", "")
        content = item.get("正文", "")
        link = item.get("页面网址", "")
        
        # 生成统计数据
        stats = self.generate_random_stats()
        
        # 解析发布时间
        published_at = self.parse_datetime(published_time)
        
        # 计算热度分值
        hotness_score = self.calculate_hotness_score(stats, published_at)
        
        # 构建新闻对象
        news_item = {
            # 基础字段
            "title": self.clean_text(title),
            "content": self.clean_text(content),
            "summary": self.clean_text(summary),
            "description": self.clean_text(summary),  # 使用简介作为描述
            "source": source if source else "未知来源",
            "category": keyword if keyword in self.categories else random.choice(self.categories),
            "published_at": published_at,
            "created_by": None,  # 手动新闻可为空
            "is_active": True,
            
            # RSS相关字段
            "source_type": "manual",  # 标记为手动创建
            "rss_source_id": None,
            "link": link,
            "guid": self.generate_guid(title, source),
            "author": "",  # 原数据中没有作者信息
            "image_url": image_url,
            "tags": self.extract_tags(keyword, keyword),
            "language": "zh",
            
            # 统计字段
            "view_count": stats["view_count"],
            "like_count": stats["like_count"],
            "comment_count": stats["comment_count"],
            "share_count": stats["share_count"],
            "hotness_score": hotness_score,
            
            # 状态字段
            "status": "published",
            "is_processed": True,
        }
        
        return news_item
    
    def convert_all(self, input_file: str, output_file: str) -> None:
        """转换所有数据"""
        try:
            # 读取原始数据
            with open(input_file, 'r', encoding='utf-8') as f:
                data = json.load(f)
            
            news_items = []
            
            # 处理Sheet1中的数据
            if "Sheet1" in data and isinstance(data["Sheet1"], list):
                for item in data["Sheet1"]:
                    try:
                        news_item = self.convert_news_item(item)
                        news_items.append(news_item)
                    except Exception as e:
                        print(f"转换条目失败: {e}")
                        print(f"问题条目: {item}")
                        continue
            
            # 保存转换后的数据
            output_data = {
                "news_items": news_items,
                "total_count": len(news_items),
                "conversion_time": datetime.now().isoformat(),
                "source_file": input_file
            }
            
            with open(output_file, 'w', encoding='utf-8') as f:
                json.dump(output_data, f, ensure_ascii=False, indent=2)
            
            print(f"转换完成！")
            print(f"原始数据条数: {len(data.get('Sheet1', []))}")
            print(f"成功转换条数: {len(news_items)}")
            print(f"输出文件: {output_file}")
            
        except Exception as e:
            print(f"转换过程中出现错误: {e}")
            raise
    
    def generate_sql_inserts(self, json_file: str, sql_file: str) -> None:
        """生成SQL插入语句"""
        try:
            with open(json_file, 'r', encoding='utf-8') as f:
                data = json.load(f)
            
            news_items = data.get("news_items", [])
            
            with open(sql_file, 'w', encoding='utf-8') as f:
                f.write("-- 自动生成的新闻数据插入语句\n")
                f.write("-- 生成时间: " + datetime.now().isoformat() + "\n\n")
                f.write("BEGIN;\n\n")
                
                for item in news_items:
                    # 转义单引号
                    def escape_sql(value):
                        if value is None:
                            return "NULL"
                        if isinstance(value, (int, float, bool)):
                            return str(value).lower() if isinstance(value, bool) else str(value)
                        return "'" + str(value).replace("'", "''") + "'"
                    
                    sql = f"""INSERT INTO news (
    title, content, summary, description, source, category, published_at,
    created_by, is_active, source_type, rss_source_id, link, guid,
    author, image_url, tags, language, view_count, like_count,
    comment_count, share_count, hotness_score, status, is_processed,
    created_at, updated_at
) VALUES (
    {escape_sql(item['title'])},
    {escape_sql(item['content'])},
    {escape_sql(item['summary'])},
    {escape_sql(item['description'])},
    {escape_sql(item['source'])},
    {escape_sql(item['category'])},
    {escape_sql(item['published_at'])},
    {escape_sql(item['created_by'])},
    {escape_sql(item['is_active'])},
    {escape_sql(item['source_type'])},
    {escape_sql(item['rss_source_id'])},
    {escape_sql(item['link'])},
    {escape_sql(item['guid'])},
    {escape_sql(item['author'])},
    {escape_sql(item['image_url'])},
    {escape_sql(item['tags'])},
    {escape_sql(item['language'])},
    {item['view_count']},
    {item['like_count']},
    {item['comment_count']},
    {item['share_count']},
    {item['hotness_score']},
    {escape_sql(item['status'])},
    {escape_sql(item['is_processed'])},
    NOW(),
    NOW()
);\n\n"""
                    f.write(sql)
                
                f.write("COMMIT;\n")
            
            print(f"SQL文件生成完成: {sql_file}")
            print(f"包含 {len(news_items)} 条插入语句")
            
        except Exception as e:
            print(f"生成SQL文件时出现错误: {e}")
            raise

def main():
    converter = NewsDataConverter()
    
    # 输入和输出文件路径
    input_file = "localization.json"
    output_json = "converted_news_data.json"
    output_sql = "insert_news_data.sql"
    
    print("开始转换 localization.json 数据...")
    
    # 转换为JSON格式
    converter.convert_all(input_file, output_json)
    
    # 生成SQL插入语句
    converter.generate_sql_inserts(output_json, output_sql)
    
    print("\n转换完成！生成的文件:")
    print(f"1. JSON格式: {output_json}")
    print(f"2. SQL格式: {output_sql}")
    print("\n你可以直接使用SQL文件导入数据库，或使用JSON文件进行其他操作。")

if __name__ == "__main__":
    main()
