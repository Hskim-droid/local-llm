[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
$ErrorActionPreference = 'Stop'
$ko = ([System.Globalization.CultureInfo]::CurrentUICulture.TwoLetterISOLanguageName -eq 'ko')
$dest = Join-Path ([Environment]::GetFolderPath('Desktop')) 'local-llm'
New-Item -ItemType Directory -Force -Path $dest | Out-Null
$zip = Join-Path $env:TEMP ('local-llm-' + [guid]::NewGuid().ToString() + '.zip')
if ($ko) { Write-Host '받는 중… 창을 닫지 마세요.' } else { Write-Host 'Downloading. Do not close this window.' }
Invoke-WebRequest -UseBasicParsing -Uri 'https://github.com/Hskim-droid/local-llm/releases/latest/download/local-llm-windows.zip' -OutFile $zip
if ($ko) { Write-Host '압축 푸는 중…' } else { Write-Host 'Extracting…' }
Expand-Archive -LiteralPath $zip -DestinationPath $dest -Force
Remove-Item $zip -ErrorAction SilentlyContinue
Start-Process explorer.exe $dest
Write-Host ''
if ($ko) {
  Write-Host '폴더가 열립니다. 시작.bat 을 더블클릭하세요.'
  Write-Host 'Windows가 막으면: 추가 정보 → 실행'
  Write-Host '검은 창을 닫지 마세요. 첫 실행은 모델을 받습니다.'
} else {
  Write-Host 'The folder is open. Double-click 시작.bat'
  Write-Host 'If Windows blocks it: More info → Run anyway'
  Write-Host 'Do not close the black window. First run downloads the model.'
}
