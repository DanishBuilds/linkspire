# Quick Cloudflare tunnels to localhost API (8081) and redirect (8080).
$ErrorActionPreference = "Stop"
$env:Path = [System.Environment]::GetEnvironmentVariable("Path","Machine") + ";" + [System.Environment]::GetEnvironmentVariable("Path","User")
$cf = "C:\Program Files (x86)\cloudflared\cloudflared.exe"
if (-not (Test-Path $cf)) { throw "Install cloudflared: winget install Cloudflare.cloudflared" }

$Root = Split-Path (Split-Path $PSScriptRoot -Parent) -Parent
$Hybrid = Join-Path $Root "deploy\hybrid"
New-Item -ItemType Directory -Force -Path $Hybrid | Out-Null

Get-Process cloudflared -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue
Start-Sleep -Seconds 1

Start-Process -FilePath $cf -ArgumentList @("tunnel","--url","http://localhost:8081") `
  -RedirectStandardError (Join-Path $Hybrid "tunnel-api.log") -WindowStyle Hidden
Start-Process -FilePath $cf -ArgumentList @("tunnel","--url","http://localhost:8080") `
  -RedirectStandardError (Join-Path $Hybrid "tunnel-redirect.log") -WindowStyle Hidden

Write-Host "Waiting for tunnel URLs..."
Start-Sleep -Seconds 10

function Get-TunnelUrl($logPath) {
  $m = Select-String -Path $logPath -Pattern "https://[a-z0-9-]+\.trycloudflare\.com" | Select-Object -First 1
  if (-not $m) { throw "No URL in $logPath — open the log and retry." }
  return $m.Matches[0].Value
}

$apiUrl = Get-TunnelUrl (Join-Path $Hybrid "tunnel-api.log")
$redirectUrl = Get-TunnelUrl (Join-Path $Hybrid "tunnel-redirect.log")
$redirectHost = ([Uri]$redirectUrl).Host

@{
  api = $apiUrl
  redirect = $redirectUrl
  startedAt = (Get-Date).ToUniversalTime().ToString("o")
} | ConvertTo-Json | Set-Content (Join-Path $Hybrid "tunnels.current.json") -Encoding utf8

$envFile = Join-Path $Root "deploy\docker\.env"
@"
LINKSPIRE_ENV=dev
PUBLIC_BASE_URL=$redirectUrl
DEFAULT_DOMAIN_HOST=$redirectHost
"@ | Set-Content $envFile -Encoding utf8

Set-Location $Root
docker compose -f deploy/docker/compose.backend.yaml --env-file $envFile up -d --force-recreate redirect workers api | Out-Null

Write-Host "API tunnel:       $apiUrl"
Write-Host "Redirect tunnel:  $redirectUrl"
Write-Host "Updated deploy/docker/.env and restarted api, redirect, workers."
