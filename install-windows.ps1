# One-time: put `report` on your user PATH
$ErrorActionPreference = "Stop"
$src = Split-Path -Parent $MyInvocation.MyCommand.Path
$dest = Join-Path $env:USERPROFILE "bin"
New-Item -ItemType Directory -Force -Path $dest | Out-Null
$files = @(
    "report.ps1", "report.cmd", "reportctl.py",
    "extract.py", "ollama_client.py", "structure.py", "render.py",
    "hardware.py", "config.json", "schema.json"
)
foreach ($f in $files) {
    Copy-Item -Force (Join-Path $src $f) (Join-Path $dest $f)
}
Copy-Item -Force (Join-Path $dest "report.ps1") (Join-Path $dest "report.ps1")
$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($userPath -notlike "*$dest*") {
    [Environment]::SetEnvironmentVariable("Path", "$dest;$userPath", "User")
    $env:Path = "$dest;$env:Path"
}
try { Set-ExecutionPolicy -Scope CurrentUser RemoteSigned -Force } catch {}
Write-Host "Installed. Open a new PowerShell and type:  report"
Write-Host "Location: $dest"
