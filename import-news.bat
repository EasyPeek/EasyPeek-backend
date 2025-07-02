@echo off
echo ================================================
echo EasyPeek 新闻数据导入工具 (postgres_easypeak)
echo ================================================
echo.

echo 检查Docker容器状态...
docker ps | findstr postgres_easypeak >nul 2>&1
if %ERRORLEVEL% neq 0 (
    echo ❌ 容器 postgres_easypeak 未运行
    echo 启动命令: docker start postgres_easypeak
    echo 或运行: start-postgres-easypeak.bat
    pause
    exit /b 1
)
echo ✅ 容器正在运行

echo.
echo 设置环境变量...
set DB_HOST=localhost
set DB_PORT=5432
set DB_USER=postgres
set DB_PASSWORD=password
set DB_NAME=easypeek
set DB_SSLMODE=disable

echo.
echo 检查数据文件...
if exist "converted_news_data.json" (
    echo ✅ 找到数据文件: converted_news_data.json
) else if exist "news_converted.json" (
    echo ✅ 找到数据文件: news_converted.json
) else if exist "localization.json" (
    echo ⚠️ 只找到 localization.json，需要先转换
    echo 正在运行数据转换...
    python scripts/convert_localization_to_news.py
    if %ERRORLEVEL% neq 0 (
        echo ❌ 数据转换失败
        pause
        exit /b 1
    )
) else (
    echo ❌ 找不到数据文件
    echo 请确保以下文件之一存在:
    echo   - converted_news_data.json
    echo   - news_converted.json
    echo   - localization.json
    pause
    exit /b 1
)

echo.
echo 运行数据导入...
go run cmd/import-news/main.go

if %ERRORLEVEL% == 0 (
    echo.
    echo ✅ 导入完成！
    echo.
    echo 现在可以运行:
    echo   1. verify.bat (验证数据)
    echo   2. go run cmd/main.go (启动服务)
) else (
    echo.
    echo ❌ 导入失败
)

echo.
pause
