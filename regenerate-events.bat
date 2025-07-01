@echo off
echo ================================================
echo EasyPeek 事件自动生成流程脚本
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
echo 步骤2: 检查PostgreSQL容器状态...
docker ps | findstr postgres_easypeak >nul 2>&1
if %ERRORLEVEL% neq 0 (
    echo ⚠️ PostgreSQL容器未运行，正在启动...
    docker start postgres_easypeak >nul 2>&1
    if %ERRORLEVEL% neq 0 (
        echo ❌ 容器启动失败，尝试创建新容器...
        docker run -d --name postgres_easypeak ^
          -e POSTGRES_USER=postgres ^
          -e POSTGRES_PASSWORD=PostgresPassword ^
          -e POSTGRES_DB=easypeekdb ^
          -p 5432:5432 ^
          postgres:15 >nul 2>&1
        if %ERRORLEVEL% neq 0 (
            echo ❌ 无法创建PostgreSQL容器，请检查Docker状态
            pause
            exit /b 1
        )
        echo ✅ 已创建并启动新容器
    ) else (
        echo ✅ 已启动现有容器
    )
) else (
    echo ✅ PostgreSQL容器运行正常
)

echo.
echo 步骤3: 清空现有事件数据...
echo ⚠️ 此操作将清空所有事件数据和新闻-事件关联
echo 是否继续? (y/n)
set /p confirm=

if /i "%confirm%" neq "y" (
    echo ❌ 操作已取消
    pause
    exit /b 0
)

echo 🔄 开始清空事件数据...
go run cmd/clear-events/main.go
if %ERRORLEVEL% neq 0 (
    echo ❌ 清空事件数据失败
    pause
    exit /b 1
)

echo.
echo 步骤4: 根据新闻数据自动生成事件...
echo 🔄 开始生成事件数据...
go run cmd/generate-events-from-news/main.go
if %ERRORLEVEL% neq 0 (
    echo ❌ 生成事件数据失败
    pause
    exit /b 1
)

echo.
echo 步骤5: 导入生成的事件数据并关联新闻...
echo 🔄 开始导入事件数据...
go run cmd/import-generated-events/main.go
if %ERRORLEVEL% neq 0 (
    echo ❌ 导入事件数据失败
    pause
    exit /b 1
)

echo.
echo ================================================
echo ✅ 整个流程已成功完成!
echo 清空事件数据 ✓
echo 生成事件JSON ✓
echo 导入事件并关联新闻 ✓
echo ================================================

pause
