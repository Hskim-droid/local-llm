# PATH에 report 등록. 모델 다운로드는 첫 report 실행이 안내합니다.
$ErrorActionPreference = "Stop"
try {
    [Console]::OutputEncoding = [System.Text.UTF8Encoding]::new()
    chcp 65001 | Out-Null
} catch {}
$src = Split-Path -Parent $MyInvocation.MyCommand.Path
$dest = Join-Path $env:USERPROFILE "bin"
New-Item -ItemType Directory -Force -Path $dest | Out-Null
$files = @(
    "report.ps1", "report.cmd", "reportctl.py", "bootstrap.py",
    "extract.py", "ollama_client.py", "structure.py", "render.py",
    "hardware.py", "config.json", "schema.json", "requirements.txt"
)
foreach ($f in $files) {
    Copy-Item -Force (Join-Path $src $f) (Join-Path $dest $f)
}
$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($userPath -notlike "*$dest*") {
    [Environment]::SetEnvironmentVariable("Path", "$dest;$userPath", "User")
    $env:Path = "$dest;$env:Path"
}
try { Set-ExecutionPolicy -Scope CurrentUser RemoteSigned -Force } catch {}
Write-Host "등록했습니다. 새 파워셸을 여세요."
Write-Host "  report"
Write-Host "위치: $dest"
Write-Host "첫 실행이 사양을 보고 모델을 순서대로 받습니다. 창을 닫지 마세요."
