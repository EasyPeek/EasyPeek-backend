@echo off
echo ================================================
echo EasyPeek 数据库迁移工具 (postgres_easypeak)
echo ================================================
echo.

if "%1"=="" (
    echo Usage: migrate.bat ^<sql_file^>
    echo.
    echo Examples:
    echo   migrate.bat migrations/simple_init.sql
    echo.
    echo 可用的迁移文件:
    dir /b migrations\*.sql 2>nul
    echo.
    pause
    exit /b 1
)

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
echo 执行迁移: %1

REM 设置数据库环境变量
set DB_HOST=localhost
set DB_PORT=5432
set DB_USER=postgres
set DB_PASSWORD=PostgresPassword
set DB_NAME=easypeekdb
set DB_SSLMODE=disable

REM 执行Go脚本
go run scripts/migrate.go %1

if %ERRORLEVEL% == 0 (
    echo.
    echo ✅ 迁移完成成功！
) else (
    echo.
    echo ❌ 迁移失败，错误代码 %ERRORLEVEL%
)

pause
