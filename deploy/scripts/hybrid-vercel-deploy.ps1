# Deploy web/ to Vercel with env from deploy/hybrid/tunnels.current.json
# Requires: vercel login (or VERCEL_TOKEN env var)
$ErrorActionPreference = "Stop"
$Root = Split-Path (Split-Path $PSScriptRoot -Parent) -Parent
$tunnelsPath = Join-Path $Root "deploy\hybrid\tunnels.current.json"
if (-not (Test-Path $tunnelsPath)) {
  throw "Run deploy/scripts/hybrid-tunnels-start.ps1 first (or create tunnels.current.json)."
}
$t = Get-Content $tunnelsPath -Raw | ConvertFrom-Json
$apiUrl = $t.api.TrimEnd("/")
$redirectUrl = $t.redirect.TrimEnd("/")

$env:Path = [System.Environment]::GetEnvironmentVariable("Path","Machine") + ";" + [System.Environment]::GetEnvironmentVariable("Path","User")
Set-Location (Join-Path $Root "web")

vercel whoami | Out-Null

function Set-VercelEnv($name, $value) {
  $value | vercel env add $name production --force 2>$null
  if ($LASTEXITCODE -ne 0) {
    $value | vercel env add $name production
  }
}

Set-VercelEnv "LINKSPIRE_API_URL" $apiUrl
Set-VercelEnv "NEXT_PUBLIC_SHORT_BASE_URL" $redirectUrl
Set-VercelEnv "SESSION_COOKIE_SECURE" "true"

vercel deploy --prod --yes
Write-Host "Done. Dashboard on Vercel; short links use $redirectUrl"
