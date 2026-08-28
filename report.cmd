@echo off
chcp 65001 >nul
set PYTHONUTF8=1
set PYTHONIOENCODING=utf-8
set "HERE=%~dp0"
where py >nul 2>nul && (
  py -3 "%HERE%reportctl.py" %*
  exit /b %ERRORLEVEL%
)
where python >nul 2>nul && (
  python "%HERE%reportctl.py" %*
  exit /b %ERRORLEVEL%
)
echo Python not found. Install 3.11+ from python.org and add it to PATH.
exit /b 1
