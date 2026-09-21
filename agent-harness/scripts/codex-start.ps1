$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)
Set-Location $Root

python harness.py init
if ($LASTEXITCODE -ne 0) { throw "harness init failed ($LASTEXITCODE)" }

python harness.py doctor
if ($LASTEXITCODE -ne 0) { throw "harness doctor failed ($LASTEXITCODE)" }

python harness.py session-start --agent codex
if ($LASTEXITCODE -ne 0) { throw "harness session-start failed ($LASTEXITCODE)" }
