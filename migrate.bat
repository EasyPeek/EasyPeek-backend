@echo off
echo EasyPeek Database Migration Tool
echo.

if "%1"=="" (
    echo Usage: migrate.bat ^<sql_file^>
    echo.
    echo Examples:
    echo   migrate.bat migrations/001_create_news_tables.sql
    echo   migrate.bat migrations/insert_sample_news.sql
    echo.
    echo Available SQL files:
    dir /b migrations\*.sql 2>nul
    exit /b 1
)

echo Executing: %1
echo.

REM 设置数据库环境变量（可根据需要修改）
set DB_HOST=localhost
set DB_PORT=5432
set DB_USER=postgres
set DB_PASSWORD=password
set DB_NAME=easypeek
set DB_SSLMODE=disable

REM 执行Go脚本
go run scripts/migrate.go %1

if %ERRORLEVEL% == 0 (
    echo.
    echo ✓ Migration completed successfully!
) else (
    echo.
    echo ✗ Migration failed with error code %ERRORLEVEL%
)

pause
