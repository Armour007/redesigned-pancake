# Dev: Start All (local)
# - Kills any existing processes on ports 8081 (backend) and 5173 (Svelte dev)
# - Ensures Postgres DB is running via docker compose
# - Starts backend and frontend with correct environment
# - Waits for backend /healthz and prints a concise status

$ErrorActionPreference = 'Stop'

function Stop-Port {
  param([int]$Port)
  Write-Host "[dev] Ensuring port $Port is free..."
  try {
    $conns = Get-NetTCPConnection -LocalPort $Port -ErrorAction SilentlyContinue
    if ($conns) {
      $pids = $conns | Select-Object -ExpandProperty OwningProcess -Unique
      foreach ($pid in $pids) {
        try { Stop-Process -Id $pid -Force -ErrorAction SilentlyContinue } catch {}
      }
    }
  } catch {
    # Fallback to netstat parsing
    $lines = netstat -ano | Select-String ":$Port\s" | ForEach-Object { $_.ToString() }
    foreach ($line in $lines) { if ($line -match '\\s+(\\d+)$') { $pid = [int]$Matches[1]; try { Stop-Process -Id $pid -Force -ErrorAction SilentlyContinue } catch {} } }
  }
}

function Wait-UrlOk {
  param([string]$Url, [int]$TimeoutSec = 60)
  $deadline = (Get-Date).AddSeconds($TimeoutSec)
  while ((Get-Date) -lt $deadline) {
    try {
      $resp = Invoke-WebRequest -Uri $Url -Method Get -TimeoutSec 3
      if ($resp.StatusCode -ge 200 -and $resp.StatusCode -lt 500) { return $true }
    } catch {}
    Start-Sleep -Seconds 1
  }
  return $false
}

$root = Resolve-Path "$PSScriptRoot/.."
Write-Host "[dev] Root = $root"

# 1) Stop existing listeners
Stop-Port 8081
Stop-Port 5173

# 2) DB up
Write-Host "[dev] Starting Postgres (docker compose) ..."
Push-Location $root
& docker compose -f docker-compose.yml up -d db | Out-Null
Pop-Location

# 3) Start backend (background job)
Write-Host "[dev] Starting backend on :8081 ..."
$backendJob = Start-Job -ScriptBlock {
  $ErrorActionPreference = 'Stop'
  $root = Resolve-Path "$using:root"
  Set-Location (Join-Path $root 'backend')
  $env:PORT = '8081'
  $env:AURA_FRONTEND_BASE_URL = 'http://localhost:5173'
  $env:AURA_API_BASE_URL = 'http://localhost:8081'
  $env:JWT_SECRET = 'dev_jwt_secret_123'
  $env:DB_HOST = 'localhost'
  $env:DB_PORT = '5432'
  $env:DB_USER = 'aura_user'
  $env:DB_PASSWORD = 'your_strong_password'
  $env:DB_NAME = 'aura_db'
  $env:DB_SSLMODE = 'disable'
  go run ./cmd/server
}

# 4) Start frontend (background job)
Write-Host "[dev] Starting frontend (Svelte dev server) ..."
$frontendJob = Start-Job -ScriptBlock {
  $ErrorActionPreference = 'Stop'
  $root = Resolve-Path "$using:root"
  Set-Location (Join-Path $root 'frontend')
  $env:PUBLIC_API_BASE = 'http://localhost:8081'
  npm run dev
}

# 5) Health check
$ok = Wait-UrlOk -Url 'http://localhost:8081/healthz' -TimeoutSec 60
if (-not $ok) {
  Write-Host "[dev] Backend did not become ready within 60s" -ForegroundColor Yellow
} else {
  try {
    $h = Invoke-WebRequest -Uri 'http://localhost:8081/healthz' -Method Get -TimeoutSec 3
    Write-Host "[dev] Backend healthz: $($h.StatusCode)" -ForegroundColor Green
  } catch {
    Write-Host "[dev] Backend healthz: ERROR" -ForegroundColor Yellow
  }
}

Write-Host "[dev] Frontend: http://localhost:5173"
Write-Host "[dev] Backend:  http://localhost:8081"
Write-Host "[dev] Jobs -> backend: $($backendJob.Id), frontend: $($frontendJob.Id)"
Write-Host "[dev] Tip: Use 'Get-Job' / 'Receive-Job' / 'Stop-Job' to manage background jobs if needed."
