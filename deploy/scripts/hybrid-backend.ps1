# Start Linkspire backend (no web) for Vercel hybrid deploy.
$ErrorActionPreference = "Stop"
$Root = Split-Path (Split-Path $PSScriptRoot -Parent) -Parent
$EnvFile = Join-Path $Root "deploy\docker\.env"
$Example = Join-Path $Root "deploy\docker\env.hybrid.example"
$Compose = Join-Path $Root "deploy\docker\compose.backend.yaml"

if (-not (Test-Path $EnvFile)) {
  Copy-Item $Example $EnvFile
  Write-Host "Created deploy/docker/.env from example. Edit PUBLIC_BASE_URL after starting redirect tunnel."
}

Set-Location $Root
docker compose -f $Compose --env-file $EnvFile up --build -d
Write-Host ""
Write-Host "Backend up. API http://localhost:8081  Redirect http://localhost:8080"
Write-Host "Next: start tunnels (see deploy/hybrid/README.md) then vercel deploy from web/"
