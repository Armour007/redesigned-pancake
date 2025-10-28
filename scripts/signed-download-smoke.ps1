$ErrorActionPreference = 'Stop'
$base = 'http://localhost:8081'
$key = 'dev_signing_key_123' # Must match AURA_DOWNLOAD_SIGNING_KEY on server

function Get-Base64Url([byte[]] $bytes) {
  $s = [Convert]::ToBase64String($bytes)
  $s = $s.TrimEnd('=')
  $s = $s.Replace('+','-').Replace('/','_')
  return $s
}

# Register and login
$rand = Get-Random -Maximum 1000000
$email = "signed$rand@example.com"
$register = @{ full_name = 'Signed User'; email = $email; password = 'P@ssw0rd12345!' } | ConvertTo-Json
Invoke-RestMethod -Method Post -Uri "$base/auth/register" -ContentType 'application/json' -Body $register | Out-Null
$loginBody = @{ email = $email; password = 'P@ssw0rd12345!' } | ConvertTo-Json
$loginRes = Invoke-RestMethod -Method Post -Uri "$base/auth/login" -ContentType 'application/json' -Body $loginBody
$token = $loginRes.token

# Get org
$orgs = Invoke-RestMethod -Method Get -Uri "$base/organizations/mine" -Headers @{ Authorization = "Bearer $token" }
$orgId = $orgs[0].id

# Create agent
$agentBody = @{ name = 'signed-agent'; description = 'signed test' } | ConvertTo-Json
$agentRes = Invoke-RestMethod -Method Post -Uri "$base/organizations/$orgId/agents" -Headers @{ Authorization = "Bearer $token" } -ContentType 'application/json' -Body $agentBody
$agentId = $agentRes.id

# Request generation for java
$payload = @{ lang = 'java'; agent_id = $agentId; organization_id = $orgId } | ConvertTo-Json
$genRes = Invoke-RestMethod -Method Post -Uri "$base/sdk/generate" -Headers @{ Authorization = "Bearer $token" } -ContentType 'application/json' -Body $payload
$job = $genRes.job_id
Write-Host "Job:" $job

# Poll status
for ($i=0; $i -lt 30; $i++) {
  Start-Sleep -Seconds 2
  $st = Invoke-RestMethod -Method Get -Uri "$base/sdk/generate/$job" -Headers @{ Authorization = "Bearer $token" }
  Write-Host "Status:" $st.status
  if ($st.status -eq 'ready') { break }
  if ($st.status -eq 'error') { throw "Job failed: $($st.error)" }
}

# Build signed URL
$exp = [DateTimeOffset]::UtcNow.ToUnixTimeSeconds() + 3600
$msg = [System.Text.Encoding]::UTF8.GetBytes("$job.$exp")
$keyBytes = [Text.Encoding]::UTF8.GetBytes($key)
$hmac = New-Object System.Security.Cryptography.HMACSHA256 (,$keyBytes)
$sigBytes = $hmac.ComputeHash($msg)
$sig = Get-Base64Url $sigBytes

$signedUrl = "$base/sdk/public/download-generated/$($job)?exp=$exp&sig=$sig"
Write-Host "Signed URL:" $signedUrl

# Download publicly (no auth)
$out = Join-Path $env:TEMP "aura_signed_$job.zip"
$response = Invoke-WebRequest -UseBasicParsing -Uri $signedUrl -OutFile $out -PassThru
Write-Host "HTTP:" $response.StatusCode
Write-Host "ZIP size:" ((Get-Item $out).Length)
