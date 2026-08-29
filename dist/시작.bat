@echo off
chcp 65001 >nul
title local-llm
cd /d "%~dp0"
echo.
echo  local-llm
echo  Do not close this window.  이 창을 닫지 마세요.
echo  Files stay on this PC.  파일은 이 노트북 밖으로 나가지 않습니다.
echo.
if exist "%~dp0로컬LLM.exe" (
  "%~dp0로컬LLM.exe" %*
) else if exist "%~dp0local-llm.exe" (
  "%~dp0local-llm.exe" %*
) else (
  echo  같은 폴더에 로컬LLM.exe 가 없습니다.
  echo  GitHub dist 폴더에서 exe와 이 bat을 같이 받아 주세요.
  pause
  exit /b 1
)
if errorlevel 1 pause
