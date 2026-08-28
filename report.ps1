# 한국어. 한 번 실행 → 사양 조사 → 패키지·모델 순차 다운로드 → 파일 드롭.
$ErrorActionPreference = "Stop"
try {
    [Console]::OutputEncoding = [System.Text.UTF8Encoding]::new()
    [Console]::InputEncoding  = [System.Text.UTF8Encoding]::new()
    $OutputEncoding = [System.Text.UTF8Encoding]::new()
    chcp 65001 | Out-Null
} catch {}
$env:PYTHONUTF8 = "1"
$env:PYTHONIOENCODING = "utf-8"
$env:PYTHONUNBUFFERED = "1"

$here = Split-Path -Parent $MyInvocation.MyCommand.Path

$py = $null
foreach ($name in @("py", "python", "python3")) {
    $cmd = Get-Command $name -ErrorAction SilentlyContinue
    if ($cmd) { $py = $cmd.Source; break }
}
if (-not $py) {
    Write-Host ""
    Write-Host "Python이 없습니다. 한 번만 설치하면 됩니다." -ForegroundColor Yellow
    Write-Host "  1) 브라우저:  https://www.python.org/downloads/windows/"
    Write-Host "  2) 설치 화면 맨 아래 'Add python.exe to PATH' 를 반드시 체크"
    Write-Host "  3) 설치 후 이 창을 닫고, 새 파워셸에서 다시 .\report.ps1"
    Write-Host ""
    exit 1
}

$pyPrefix = @()
if ((Split-Path $py -Leaf) -eq "py.exe") { $pyPrefix = @("-3") }

$skipSetup = $false
$forceSetup = $false
$ctlArgs = @()
for ($i = 0; $i -lt $args.Count; $i++) {
    if ($args[$i] -eq "--skip-setup") { $skipSetup = $true; continue }
    if ($args[$i] -eq "--force") { $forceSetup = $true; continue }
    $ctlArgs += $args[$i]
}
if ($ctlArgs -contains "--status") { $skipSetup = $true }

if (-not $skipSetup) {
    $boot = $pyPrefix + @((Join-Path $here "bootstrap.py"))
    for ($i = 0; $i -lt $args.Count; $i++) {
        if ($args[$i] -eq "--profile" -and ($i + 1) -lt $args.Count) {
            $boot += @("--profile", $args[$i + 1])
        }
    }
    if ($forceSetup) { $boot += "--force" }
    & $py @boot
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
}

$run = $pyPrefix + @((Join-Path $here "reportctl.py")) + $ctlArgs
& $py @run
exit $LASTEXITCODE
