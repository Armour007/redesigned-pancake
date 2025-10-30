param(
    [switch]$KeepBackend
)

$ErrorActionPreference = 'Stop'

# Resolve repo root and logs dir
$repoRoot = Split-Path -Parent $PSScriptRoot
$logsDir = Join-Path $repoRoot 'logs'
if (!(Test-Path $logsDir)) { New-Item -ItemType Directory -Path $logsDir | Out-Null }
$serverOut = Join-Path $logsDir 'backend-smokes-server.out.log'
$serverErr = Join-Path $logsDir 'backend-smokes-server.err.log'
foreach ($p in @($serverOut,$serverErr)) { if (Test-Path $p) { try { Remove-Item $p -Force } catch {} } }
$migrateLog = Join-Path $logsDir 'migrate.log'
$migrateErr = Join-Path $logsDir 'migrate.err.log'
foreach ($p in @($migrateLog,$migrateErr)) { if (Test-Path $p) { Remove-Item $p -Force } }

function Wait-For-Health {
    param(
        [string]$Url = 'http://localhost:18081/healthz',
        [int]$TimeoutSec = 60
    )
    $deadline = (Get-Date).AddSeconds($TimeoutSec)
    while ((Get-Date) -lt $deadline) {
        try {
            $resp = Invoke-WebRequest -UseBasicParsing -Uri $Url -TimeoutSec 3
            if ($resp.StatusCode -eq 200) { return $true }
        } catch { }
        Start-Sleep -Seconds 1
    }
    return $false
}

# Ensure MailHog SMTP sink is running
Write-Host "[run-smokes] Ensuring MailHog (SMTP sink) is running..."
try { docker pull mailhog/mailhog *> $null } catch {}
try { docker rm -f aura-mailhog *> $null } catch {}
try { docker run -d --name aura-mailhog -p 1025:1025 -p 8025:8025 mailhog/mailhog *> $null } catch {}

# Free port 18081 if needed (kill all listeners)
try {
    $conns = Get-NetTCPConnection -LocalPort 18081 -State Listen -ErrorAction SilentlyContinue
    if ($conns) {
        $pids = $conns | Select-Object -ExpandProperty OwningProcess | Sort-Object -Unique
        foreach ($pid in $pids) {
            try { Stop-Process -Id $pid -Force -ErrorAction SilentlyContinue } catch {}
        }
        # Wait briefly until port is free
        $deadline = (Get-Date).AddSeconds(5)
        while ((Get-Date) -lt $deadline) {
            $still = Get-NetTCPConnection -LocalPort 18081 -State Listen -ErrorAction SilentlyContinue
            if (-not $still) { break }
            Start-Sleep -Milliseconds 200
        }
    }
} catch {}

# Export env vars in parent so children inherit
$env:DB_HOST='localhost'
$env:DB_PORT='5432'
$env:DB_USER='aura_user'
$env:DB_PASSWORD='your_strong_password'
$env:DB_NAME='aura_db'
$env:DB_SSLMODE='disable'
$env:PORT='18081'
$env:AURA_FRONTEND_BASE_URL='http://localhost:5173'
$env:AURA_API_BASE_URL='http://localhost:18081'
$env:JWT_SECRET='dev_jwt_secret_123'
$env:AURA_DOWNLOAD_SIGNING_KEY='dev_signing_key_123'
$env:SMTP_HOST='localhost'
$env:SMTP_PORT='1025'
$env:SMTP_USER=''
$env:SMTP_PASS=''
$env:SMTP_FROM='noreply@localhost'

# Run DB migrations first
Write-Host "[run-smokes] Applying database migrations..."
$migProc = Start-Process -FilePath "go" -WorkingDirectory (Join-Path $repoRoot 'backend') -ArgumentList @("run","./cmd/migrate") -RedirectStandardOutput $migrateLog -RedirectStandardError $migrateErr -PassThru -WindowStyle Hidden
Wait-Process -Id $migProc.Id
if ($migProc.ExitCode -ne 0) { Write-Warning "[run-smokes] Migration process exited with $($migProc.ExitCode). Continuing. See $migrateLog and $migrateErr" }

Write-Host "[run-smokes] Starting backend (with signing + SMTP) as a background process..."
$backendProc = Start-Process -FilePath "go" -WorkingDirectory (Join-Path $repoRoot 'backend') -ArgumentList @("run","./cmd/server") -RedirectStandardOutput $serverOut -RedirectStandardError $serverErr -PassThru -WindowStyle Hidden

if (-not (Wait-For-Health -TimeoutSec 60)) {
    try { Stop-Process -Id $backendProc.Id -Force -ErrorAction SilentlyContinue } catch {}
    throw "Backend failed to become healthy on http://localhost:18081/healthz. See $serverOut and $serverErr"
}

# Capture admin health snapshot (authenticated) to include DB/queue details
try {
    $rnd = Get-Random -Maximum 10000000
    $email = "health$rnd@example.com"
    $regBody = @{ full_name='Health Probe'; email=$email; password='P@ssw0rd12345!' } | ConvertTo-Json
    $loginBody = @{ email=$email; password='P@ssw0rd12345!' } | ConvertTo-Json
    Invoke-RestMethod -Headers @{ 'Accept-Encoding'='identity' } -Method Post -Uri 'http://localhost:18081/auth/register' -ContentType 'application/json' -Body $regBody | Out-Null
    $loginRes = Invoke-RestMethod -Headers @{ 'Accept-Encoding'='identity' } -Method Post -Uri 'http://localhost:18081/auth/login' -ContentType 'application/json' -Body $loginBody
    $tok = $loginRes.token
    Invoke-WebRequest -UseBasicParsing -Headers @{ 'Authorization' = "Bearer $tok"; 'Accept-Encoding' = 'identity' } -Uri 'http://localhost:18081/admin/health' -OutFile (Join-Path $logsDir 'admin-health.json') -TimeoutSec 10 | Out-Null
} catch {
    "admin/health failed: $($_.Exception.Message)" | Out-File -FilePath (Join-Path $logsDir 'admin-health.json') -Encoding utf8
}

Write-Host "[run-smokes] Backend is healthy. Running smokes..."

$results = @()

function Run-Smoke {
    param(
        [string]$Name,
        [string]$ScriptPath,
        [string]$LogPath
    )
    Write-Host "[run-smokes] Running $Name ..."
    $exitCode = 0
    try {
        & $ScriptPath *> $LogPath
        if (-not $?) { $exitCode = 1 }
    } catch {
        $exitCode = 1
        "ERROR: $($_.Exception.Message)" | Out-File -FilePath $LogPath -Append -Encoding utf8
    }
    $results += [PSCustomObject]@{ Name = $Name; Log = $LogPath; ExitCode = $exitCode }
}

Run-Smoke -Name 'signed-download-smoke' -ScriptPath (Join-Path $repoRoot 'scripts/signed-download-smoke.ps1') -LogPath (Join-Path $logsDir 'signed-download-smoke.log')
Run-Smoke -Name 'public-download-smoke' -ScriptPath (Join-Path $repoRoot 'scripts\test-public-download.ps1') -LogPath (Join-Path $logsDir 'public-download-smoke.log')
Run-Smoke -Name 'smtp-test-smoke' -ScriptPath (Join-Path $repoRoot 'scripts\smtp-test-smoke.ps1') -LogPath (Join-Path $logsDir 'smtp-test-smoke.log')

if (-not $KeepBackend) {
    Write-Host "[run-smokes] Stopping backend process..."
    try { Stop-Process -Id $backendProc.Id -Force -ErrorAction SilentlyContinue } catch {}
}

Write-Host "[run-smokes] Results:"
$results | ForEach-Object { Write-Host (" - {0}: Exit={1} Log={2}" -f $_.Name, $_.ExitCode, $_.Log) }

# Exit non-zero if any failed
if ($results | Where-Object { $_.ExitCode -ne 0 }) { exit 1 } else { exit 0 }
