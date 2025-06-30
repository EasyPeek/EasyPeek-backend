# EasyPeek 数据库迁移脚本 (PowerShell)
# 使用方法: .\migrate.ps1 up 或 .\migrate.ps1 down

param(
    [Parameter(Mandatory=$true)]
    [ValidateSet("up", "down")]
    [string]$Command
)

Write-Host "🚀 EasyPeek 数据库迁移工具" -ForegroundColor Green
Write-Host "==========================" -ForegroundColor Green

# 检查是否在正确的目录
if (-not (Test-Path "migrations\001_create_news_tables.sql")) {
    Write-Host "❌ 错误: 请在项目根目录运行此脚本" -ForegroundColor Red
    Write-Host "当前目录: $(Get-Location)" -ForegroundColor Yellow
    exit 1
}

# 设置数据库连接参数（请根据实际情况修改）
$DB_HOST = "localhost"
$DB_PORT = "5432"
$DB_NAME = "easypeek_db"
$DB_USER = "postgres"

Write-Host "📋 数据库连接信息:" -ForegroundColor Cyan
Write-Host "   主机: $DB_HOST" -ForegroundColor Gray
Write-Host "   端口: $DB_PORT" -ForegroundColor Gray
Write-Host "   数据库: $DB_NAME" -ForegroundColor Gray
Write-Host "   用户: $DB_USER" -ForegroundColor Gray

switch ($Command) {
    "up" {
        Write-Host "⬆️  执行数据库迁移..." -ForegroundColor Yellow
        
        try {
            # 使用psql执行迁移脚本
            $env:PGPASSWORD = Read-Host "请输入数据库密码" -AsSecureString | ConvertFrom-SecureString -AsPlainText
            
            psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -f "migrations\001_create_news_tables.sql"
            
            if ($LASTEXITCODE -eq 0) {
                Write-Host "✅ 迁移执行成功!" -ForegroundColor Green
                Write-Host "📊 创建的表: rss_sources, news, event_news_relations" -ForegroundColor Cyan
                Write-Host "🔧 创建的函数: calculate_news_hotness, update_news_hotness" -ForegroundColor Cyan
                Write-Host "👁️  创建的视图: news_with_stats, news_stats_summary" -ForegroundColor Cyan
            } else {
                Write-Host "❌ 迁移执行失败!" -ForegroundColor Red
                exit 1
            }
        }
        catch {
            Write-Host "❌ 执行错误: $($_.Exception.Message)" -ForegroundColor Red
            exit 1
        }
    }
    
    "down" {
        Write-Host "⚠️  警告: 此操作将删除所有新闻数据!" -ForegroundColor Red
        $confirm = Read-Host "确认继续? (y/N)"
        
        if ($confirm -ne "y" -and $confirm -ne "Y") {
            Write-Host "🚫 操作已取消" -ForegroundColor Yellow
            return
        }
        
        Write-Host "⬇️  执行数据库回滚..." -ForegroundColor Yellow
        
        try {
            if (-not $env:PGPASSWORD) {
                $env:PGPASSWORD = Read-Host "请输入数据库密码" -AsSecureString | ConvertFrom-SecureString -AsPlainText
            }
            
            psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -f "migrations\001_rollback_news_tables.sql"
            
            if ($LASTEXITCODE -eq 0) {
                Write-Host "✅ 回滚执行成功!" -ForegroundColor Green
                Write-Host "🗑️  已删除所有新闻相关表和数据" -ForegroundColor Cyan
            } else {
                Write-Host "❌ 回滚执行失败!" -ForegroundColor Red
                exit 1
            }
        }
        catch {
            Write-Host "❌ 执行错误: $($_.Exception.Message)" -ForegroundColor Red
            exit 1
        }
    }
}

Write-Host "`n🎉 操作完成!" -ForegroundColor Green
Write-Host "💡 提示: 可以使用以下命令验证结果:" -ForegroundColor Cyan
Write-Host "   psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c '\dt'" -ForegroundColor Gray
