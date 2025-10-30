# Backend smoke test script
$ErrorActionPreference = 'Stop'
$base = 'http://localhost:8081'

# Begin transcript logging to capture intermittent failures with timestamps
try {
	$logsDir = Join-Path $PSScriptRoot '..' | Join-Path -ChildPath 'logs'
	if (-not (Test-Path $logsDir)) { New-Item -ItemType Directory -Path $logsDir | Out-Null }
	$ts = Get-Date -Format 'yyyyMMdd-HHmmss'
	$logFile = Join-Path $logsDir "backend-smoke-$ts.log"
	Start-Transcript -Path $logFile -Append | Out-Null
} catch { }

# Helper: wait for backend readiness
$waitSec = [int]([Environment]::GetEnvironmentVariable('AURA_SMOKE_WAIT_SEC') | ForEach-Object { if ($_ -and $_ -match '^[0-9]+$') { $_ } else { '60' } })
function Wait-UrlOk {
	param(
		[string]$Url,
		[int]$TimeoutSec = 60
	)
	$deadline = (Get-Date).AddSeconds($TimeoutSec)
	while ((Get-Date) -lt $deadline) {
		try {
			$resp = Invoke-WebRequest -Uri $Url -Method Get -TimeoutSec 3
			if ($resp.StatusCode -ge 200 -and $resp.StatusCode -lt 500) { return $true }
		} catch {
			# swallow and retry
		}
		Start-Sleep -Seconds 1
	}
	return $false
}

# Wait for /healthz before proceeding (handles intermittent backend start timing)
Write-Host "[" (Get-Date -Format o) "] Waiting for backend at $base/healthz (timeout ${waitSec}s)..."
if (-not (Wait-UrlOk -Url "$base/healthz" -TimeoutSec $waitSec)) {
	Write-Host "[" (Get-Date -Format o) "] Backend did not become ready within ${waitSec}s."
	try { Stop-Transcript | Out-Null } catch { }
	exit 1
}

# Health checks
Write-Host "[" (Get-Date -Format o) "] Healthz:"; (Invoke-WebRequest "$base/healthz").StatusCode
Write-Host "[" (Get-Date -Format o) "] OpenAPI:"; (Invoke-WebRequest "$base/openapi.json").StatusCode

# Register
$rand = Get-Random -Maximum 1000000
$email = "dev$rand@example.com"
$register = @{ full_name = 'Dev User'; email = $email; password = 'P@ssw0rd12345!' } | ConvertTo-Json
$regRes = Invoke-RestMethod -Method Post -Uri "$base/auth/register" -ContentType 'application/json' -Body $register
Write-Host "[" (Get-Date -Format o) "] Registered:" ($regRes.email)

# Login
$loginBody = @{ email = $email; password = 'P@ssw0rd12345!' } | ConvertTo-Json
$loginRes = Invoke-RestMethod -Method Post -Uri "$base/auth/login" -ContentType 'application/json' -Body $loginBody
$token = $loginRes.token
Write-Host "[" (Get-Date -Format o) "] Token acquired: " ($token.Substring(0,16) + '...')

# Get org
$orgs = Invoke-RestMethod -Method Get -Uri "$base/organizations/mine" -Headers @{ Authorization = "Bearer $token" }
$orgId = $orgs[0].id
Write-Host "[" (Get-Date -Format o) "] Org:" $orgId

# Create agent
$agentBody = @{ name = 'demo-agent'; description = 'demo agent' } | ConvertTo-Json
$agentRes = Invoke-RestMethod -Method Post -Uri "$base/organizations/$orgId/agents" -Headers @{ Authorization = "Bearer $token" } -ContentType 'application/json' -Body $agentBody
$agentId = $agentRes.id
Write-Host "[" (Get-Date -Format o) "] Agent:" $agentId

# Add allow rule (as nested JSON object, not a JSON string)
$ruleObj = @{ action = 'deploy'; effect = 'allow'; context = @{ env = 'prod' } }
$ruleBody = @{ rule = $ruleObj } | ConvertTo-Json -Depth 5 -Compress
$ruleRes = Invoke-RestMethod -Method Post -Uri "$base/organizations/$orgId/agents/$agentId/permissions" -Headers @{ Authorization = "Bearer $token" } -ContentType 'application/json' -Body $ruleBody
Write-Host "[" (Get-Date -Format o) "] Rule:" $ruleRes.id

# Create API key
$keyBody = @{ name = 'dev key' } | ConvertTo-Json
$keyRes = Invoke-RestMethod -Method Post -Uri "$base/organizations/$orgId/apikeys" -Headers @{ Authorization = "Bearer $token" } -ContentType 'application/json' -Body $keyBody
$secret = $keyRes.secret_key
Write-Host "[" (Get-Date -Format o) "] API key prefix:" $keyRes.key_prefix

# Verify with API key
$ctxObj = @{ action = 'deploy'; env = 'prod' }
$verifyObj = @{ agent_id = "$agentId"; request_context = $ctxObj }
$verifyJson = $verifyObj | ConvertTo-Json -Depth 5 -Compress
$verifyRes = Invoke-RestMethod -Method Post -Uri "$base/v1/verify" -Headers @{ 'X-API-Key' = $secret; 'AURA-Version' = '2025-10-01' } -ContentType 'application/json' -Body $verifyJson
Write-Host "[" (Get-Date -Format o) "] Decision:" $verifyRes.decision "Reason:" $verifyRes.reason

# End transcript
try { Stop-Transcript | Out-Null } catch { }
