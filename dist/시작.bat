@echo off
chcp 65001 >nul
cd /d "%~dp0"
if exist "%~dp0그램보고서.exe" (
  "%~dp0그램보고서.exe" %*
) else (
  "%~dp0gram-report.exe" %*
)
if errorlevel 1 pause
