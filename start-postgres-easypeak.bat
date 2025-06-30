@echo off
echo ================================================
echo EasyPeek PostgreSQL 容器管理脚本
echo 容器名称: postgres_easypeak
echo ================================================
echo.

echo 步骤1: 检查Docker状态...
docker version >nul 2>&1
if %ERRORLEVEL% neq 0 (
    echo ❌ Docker未运行，请先启动Docker Desktop
    pause
    exit /b 1
)
echo ✅ Docker运行正常

echo.
echo 步骤2: 检查容器状态...
docker ps | grep postgres_easypeak >nul 2>&1
if %ERRORLEVEL% == 0 (
    echo ✅ 容器 postgres_easypeak 已运行
    goto :test_connection
)

echo ⚠️ 容器未运行，检查是否存在...
docker ps -a | grep postgres_easypeak >nul 2>&1
if %ERRORLEVEL% == 0 (
    echo 📦 容器存在但已停止，正在启动...
    docker start postgres_easypeak
    if %ERRORLEVEL% neq 0 (
        echo ❌ 启动失败
        pause
        exit /b 1
    )
    echo ✅ 容器启动成功
) else (
    echo 📦 容器不存在，正在创建...
    docker run -d ^
      --name postgres_easypeak ^
      -e POSTGRES_USER=postgres ^
      -e POSTGRES_PASSWORD=password ^
      -e POSTGRES_DB=easypeek ^
      -p 5432:5432 ^
      postgres:15
    
    if %ERRORLEVEL% neq 0 (
        echo ❌ 容器创建失败
        pause
        exit /b 1
    )
    echo ✅ 容器创建成功
)

echo.
echo 🕒 等待数据库完全启动...
timeout /t 10 /nobreak >nul

:test_connection
echo.
echo 步骤3: 测试数据库连接...
docker exec postgres_easypeak pg_isready -U postgres >nul 2>&1
if %ERRORLEVEL% neq 0 (
    echo ❌ 数据库连接测试失败
    echo 查看容器日志:
    docker logs postgres_easypeak
    pause
    exit /b 1
)
echo ✅ 数据库连接正常

echo.
echo 步骤4: 设置环境变量...
set DB_HOST=localhost
set DB_PORT=5432
set DB_USER=postgres
set DB_PASSWORD=password
set DB_NAME=easypeek
set DB_SSLMODE=disable
echo ✅ 环境变量设置完成

echo.
echo ================================================
echo 🎉 PostgreSQL 容器准备就绪！
echo ================================================
echo.
echo 容器信息:
echo   名称: postgres_easypeak
echo   主机: localhost:5432
echo   用户: postgres
echo   密码: password
echo   数据库: easypeek
echo.
echo 现在可以运行:
echo 1. go run scripts/migrate.go migrations/001_create_news_tables.sql
echo 2. go run cmd/import-news/main.go
echo 3. go run cmd/verify/main.go
echo.
echo 容器管理命令:
echo   停止: docker stop postgres_easypeak
echo   启动: docker start postgres_easypeak
echo   日志: docker logs postgres_easypeak
echo   进入: docker exec -it postgres_easypeak psql -U postgres -d easypeek
echo.
pause
