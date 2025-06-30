@echo off
echo ================================================
echo EasyPeek 数据库工具集 (postgres_easypeak)
echo ================================================
echo.

echo 可用工具:
echo.
echo 1. 启动数据库容器
echo    start-postgres-easypeak.bat
echo.
echo 2. 数据库迁移
echo    migrate.bat migrations/simple_init.sql
echo.
echo 3. 数据库验证
echo    verify.bat
echo.
echo 4. 数据库诊断
echo    go run cmd/diagnose-postgres-easypeak/main.go
echo.
echo 5. 导入新闻数据
echo    go run cmd/import-news/main.go
echo.
echo 容器管理命令:
echo   启动: docker start postgres_easypeak
echo   停止: docker stop postgres_easypeak
echo   状态: docker ps | findstr postgres_easypeak
echo   日志: docker logs postgres_easypeak
echo   进入: docker exec -it postgres_easypeak psql -U postgres -d easypeek
echo.

set /p choice="选择操作 (1-5) 或按任意键退出: "

if "%choice%"=="1" (
    call start-postgres-easypeak.bat
) else if "%choice%"=="2" (
    call migrate.bat migrations/simple_init.sql
) else if "%choice%"=="3" (
    call verify.bat
) else if "%choice%"=="4" (
    go run cmd/diagnose-postgres-easypeak/main.go
    pause
) else if "%choice%"=="5" (
    go run cmd/import-news/main.go
    pause
) else (
    echo 退出工具集
)
