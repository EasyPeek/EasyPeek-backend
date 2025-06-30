@echo off
echo 🔍 EasyPeek 数据库快速验证
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
