@echo off
chcp 65001 >nul
set PYTHONUTF8=1
set PYTHONIOENCODING=utf-8
set PYTHONUNBUFFERED=1
set "HERE=%~dp0"
where powershell >nul 2>nul && (
  powershell -NoProfile -ExecutionPolicy Bypass -File "%HERE%report.ps1" %*
  exit /b %ERRORLEVEL%
)
echo 파워셸을 찾을 수 없습니다.
exit /b 1
