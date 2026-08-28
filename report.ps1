# PowerShell: `.\report.ps1` then drop a video, PPTX, and PDF
$ErrorActionPreference = "Stop"
try {
    [Console]::OutputEncoding = [System.Text.UTF8Encoding]::new()
} catch {}
$env:PYTHONUTF8 = "1"
$env:PYTHONIOENCODING = "utf-8"

$here = Split-Path -Parent $MyInvocation.MyCommand.Path
$script = Join-Path $here "reportctl.py"

$py = $null
foreach ($name in @("py", "python", "python3")) {
    $cmd = Get-Command $name -ErrorAction SilentlyContinue
    if ($cmd) { $py = $cmd.Source; break }
}
if (-not $py) {
    Write-Error "Python not found. Install 3.11+ from python.org and tick 'Add python.exe to PATH'."
    exit 1
}

$pyArgs = @()
if ((Split-Path $py -Leaf) -eq "py.exe") { $pyArgs += "-3" }
$pyArgs += @($script) + $args
& $py @pyArgs
exit $LASTEXITCODE
