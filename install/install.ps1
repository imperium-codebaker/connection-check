# Downloads Connection Check for this computer and starts it.
# Usage: & ([scriptblock]::Create((irm <release>/install.ps1))) -Site 'Name=host','Name2=host2'
param([string[]]$Site = @())

$ErrorActionPreference = 'Stop'
$repo = 'johncbrubaker/connection-check'
$dir = Join-Path $env:LOCALAPPDATA 'connection-check'
$bin = Join-Path $dir 'connection-check.exe'

New-Item -ItemType Directory -Force -Path $dir | Out-Null
Write-Host 'Downloading Connection Check...'
$ProgressPreference = 'SilentlyContinue'
Invoke-WebRequest -UseBasicParsing -OutFile "$bin.tmp" `
  -Uri "https://github.com/$repo/releases/latest/download/connection-check-windows-amd64.exe"
Move-Item -Force "$bin.tmp" $bin

$argList = @()
foreach ($s in $Site) { $argList += @('--site', $s) }
& $bin @argList
