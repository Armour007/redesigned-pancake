param(
  [int]$AppPort = 8081
)

$ErrorActionPreference = 'Stop'

function Wait-Port {
  param(
    [string]$ComputerName = 'localhost',
    [int]$Port = 5432,
    [int]$TimeoutSec = 60
  )
  $deadline = (Get-Date).AddSeconds($TimeoutSec)
  while ((Get-Date) -lt $deadline) {
    try {
      $client = New-Object System.Net.Sockets.TcpClient
      $iar = $client.BeginConnect($ComputerName, $Port, $null, $null)
      $ok = $iar.AsyncWaitHandle.WaitOne(1000, $false)
      if ($ok -and $client.Connected) { $client.Close(); return $true }
      $client.Close()
    } catch { }
  }
  return $false
}

# Ensure DB is reachable first
Write-Host "[" (Get-Date -Format o) "] Waiting for Postgres at localhost:5432..."
if (-not (Wait-Port -ComputerName 'localhost' -Port 5432 -TimeoutSec 60)) {
  Write-Error "Postgres not reachable on localhost:5432. Start Docker Desktop and run the 'DB: up (local)' task first."
  exit 1
}

# Free application port if currently held by a previous run
try {
  $listeners = Get-NetTCPConnection -LocalPort $AppPort -State Listen -ErrorAction SilentlyContinue
  if ($listeners) {
    $pids = $listeners.OwningProcess | Select-Object -Unique
    Write-Host "[" (Get-Date -Format o) "] Port $AppPort busy, terminating PIDs: $($pids -join ',')"
    foreach ($procId in $pids) { try { Stop-Process -Id $procId -Force -ErrorAction Stop } catch { Write-Warning $_.Exception.Message } }
    Start-Sleep -Seconds 1
  }
} catch {}

# Set backend env
Set-Location "$PSScriptRoot/../backend"
$env:DB_HOST = 'localhost'
$env:DB_PORT = '5432'
$env:DB_USER = 'aura_user'
$env:DB_PASSWORD = 'your_strong_password'
$env:DB_NAME = 'aura_db'
$env:DB_SSLMODE = 'disable'
$env:JWT_SECRET = 'dev_jwt_secret_123'
$env:PORT = "$AppPort"
$env:AURA_FRONTEND_BASE_URL = 'http://localhost:5173'
$env:AURA_API_BASE_URL = 'http://localhost:8081'

Write-Host "[" (Get-Date -Format o) "] Starting backend..."
go run ./cmd/server
