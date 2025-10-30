$ErrorActionPreference = 'Stop'
$ProgressPreference = 'SilentlyContinue'
$base = 'http://localhost:18081'
$key = 'dev_signing_key_123' # Must match AURA_DOWNLOAD_SIGNING_KEY on server

function Invoke-RestJson {
  param(
    [Parameter(Mandatory=$true)][string]$Method,
    [Parameter(Mandatory=$true)][string]$Uri,
    [hashtable]$Headers,
    $Body
  )
  try {
    # Ensure we can read plain-text error bodies by forcing identity encoding
    $hdrs = @{ 'Accept-Encoding' = 'identity' }
    if ($Headers) { foreach ($k in $Headers.Keys) { $hdrs[$k] = $Headers[$k] } }
    if ($PSBoundParameters.ContainsKey('Body')) {
      return Invoke-RestMethod -Method $Method -Uri $Uri -Headers $hdrs -ContentType 'application/json' -Body $Body
    } else {
      return Invoke-RestMethod -Method $Method -Uri $Uri -Headers $hdrs
    }
  } catch {
    if ($_.Exception.Response) {
      try { $sr = New-Object IO.StreamReader($_.Exception.Response.GetResponseStream()); $txt = $sr.ReadToEnd(); Write-Host "HTTP ERROR BODY:"; Write-Host $txt } catch {}
    }
    throw
  }
}

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
Invoke-RestJson -Method Post -Uri "$base/auth/register" -Body $register | Out-Null
$loginBody = @{ email = $email; password = 'P@ssw0rd12345!' } | ConvertTo-Json
$loginRes = Invoke-RestJson -Method Post -Uri "$base/auth/login" -Body $loginBody
$token = $loginRes.token

# Get org
$orgs = Invoke-RestJson -Method Get -Uri "$base/organizations/mine" -Headers @{ Authorization = "Bearer $token" }
$orgId = $orgs[0].id

# Create agent
$agentBody = @{ name = 'signed-agent'; description = 'signed test' } | ConvertTo-Json
$agentRes = Invoke-RestJson -Method Post -Uri "$base/organizations/$orgId/agents" -Headers @{ Authorization = "Bearer $token" } -Body $agentBody
$agentId = $agentRes.id

# Request generation for java
$payload = @{ lang = 'java'; agent_id = $agentId; organization_id = $orgId } | ConvertTo-Json
$genRes = Invoke-RestJson -Method Post -Uri "$base/sdk/generate" -Headers @{ Authorization = "Bearer $token" } -Body $payload
$job = $genRes.job_id
Write-Host "Job:" $job

# Poll status
for ($i=0; $i -lt 30; $i++) {
  Start-Sleep -Seconds 2
  $st = Invoke-RestJson -Method Get -Uri "$base/sdk/generate/$job" -Headers @{ Authorization = "Bearer $token" }
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
$response = Invoke-WebRequest -UseBasicParsing -Headers @{ 'Accept-Encoding' = 'identity' } -Uri $signedUrl -OutFile $out -PassThru
Write-Host "HTTP:" $response.StatusCode
Write-Host "ZIP size:" ((Get-Item $out).Length)
