#!/usr/bin/env pwsh

param(
    [Parameter(Mandatory=$true)]
    [string]$SqlFile
)

Write-Host "EasyPeek Database Migration Tool" -ForegroundColor Green
Write-Host ""

if (-not (Test-Path $SqlFile)) {
    Write-Host "Error: SQL file '$SqlFile' not found!" -ForegroundColor Red
    Write-Host ""
    Write-Host "Available SQL files in migrations/:" -ForegroundColor Yellow
    Get-ChildItem -Path "migrations/*.sql" -Name 2>$null
    exit 1
}

Write-Host "Executing: $SqlFile" -ForegroundColor Cyan
Write-Host ""

# 设置数据库环境变量（可根据需要修改）
$env:DB_HOST = "localhost"
$env:DB_PORT = "5432"
$env:DB_USER = "postgres"
$env:DB_PASSWORD = "password"
$env:DB_NAME = "easypeek"
$env:DB_SSLMODE = "disable"

# 执行Go脚本
$result = & go run scripts/migrate.go $SqlFile

if ($LASTEXITCODE -eq 0) {
    Write-Host ""
    Write-Host "✓ Migration completed successfully!" -ForegroundColor Green
} else {
    Write-Host ""
    Write-Host "✗ Migration failed with error code $LASTEXITCODE" -ForegroundColor Red
}

Read-Host "Press Enter to continue"
