@echo off
echo ================================================
echo EasyPeek 数据库验证工具 (postgres_easypeak)
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
echo 设置数据库环境变量...
set DB_HOST=localhost
set DB_PORT=5432
set DB_USER=postgres
set DB_PASSWORD=password
set DB_NAME=easypeek
set DB_SSLMODE=disable

echo.
echo 运行数据库诊断...
go run cmd/diagnose-postgres-easypeak/main.go

if %ERRORLEVEL% == 0 (
    echo.
    echo 运行详细验证...
    go run cmd/verify/main.go
)

echo.
pause EasyPeek 数据库快速验证
echo.

REM 设置数据库环境变量
set DB_HOST=localhost
set DB_PORT=5432
set DB_USER=postgres
set DB_PASSWORD=password
set DB_NAME=easypeek
set DB_SSLMODE=disable

echo 正在验证数据库...
go run cmd/verify/main.go

echo.
pause
