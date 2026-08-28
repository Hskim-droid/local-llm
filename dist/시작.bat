@echo off
chcp 65001 >nul
cd /d "%~dp0"
if exist "%~dp0로컬LLM보고서.exe" (
  "%~dp0로컬LLM보고서.exe" %*
) else (
  "%~dp0local-llm-report.exe" %*
)
if errorlevel 1 pause
