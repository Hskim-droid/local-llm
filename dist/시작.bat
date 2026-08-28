@echo off
chcp 65001 >nul
title 로컬LLM보고서
cd /d "%~dp0"
echo.
echo  로컬LLM보고서
echo  이 창을 닫지 마세요. 안내는 프로그램이 한글로 합니다.
echo  파일은 이 노트북 밖으로 나가지 않습니다.
echo.
if exist "%~dp0로컬LLM보고서.exe" (
  "%~dp0로컬LLM보고서.exe" %*
) else if exist "%~dp0local-llm-report.exe" (
  "%~dp0local-llm-report.exe" %*
) else (
  echo  같은 폴더에 로컬LLM보고서.exe 가 없습니다.
  echo  GitHub dist 폴더에서 exe와 이 bat을 같이 받아 주세요.
  pause
  exit /b 1
)
if errorlevel 1 pause
