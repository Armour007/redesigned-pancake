$ErrorActionPreference = 'Stop'
$ProgressPreference = 'SilentlyContinue'
$base = 'http://localhost:18081'
# If signing is enabled on the server, unauthenticated public download without a signature will be rejected.
# In that case, skip this smoke (covered by signed-download-smoke).
if ($env:AURA_DOWNLOAD_SIGNING_KEY) {
  Write-Host "Signing key detected; skipping unsigned public download smoke."
  exit 0
}
# Helper to print response bodies on errors
function Invoke-RestJson {
  param(
    [Parameter(Mandatory=$true)][string]$Method,
    [Parameter(Mandatory=$true)][string]$Uri,
    [hashtable]$Headers,
    $Body
  )
  try {
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

# Register and login to create a real job
$rand = Get-Random -Maximum 1000000
$email = "public$rand@example.com"
$register = @{ full_name = 'Public User'; email = $email; password = 'P@ssw0rd12345!' } | ConvertTo-Json
Invoke-RestJson -Method Post -Uri "$base/auth/register" -Body $register | Out-Null
$loginBody = @{ email = $email; password = 'P@ssw0rd12345!' } | ConvertTo-Json
$loginRes = Invoke-RestJson -Method Post -Uri "$base/auth/login" -Body $loginBody
$token = $loginRes.token

# Get org and create agent
$orgs = Invoke-RestJson -Method Get -Uri "$base/organizations/mine" -Headers @{ Authorization = "Bearer $token" }
$orgId = $orgs[0].id
$agentBody = @{ name = 'public-agent'; description = 'public test' } | ConvertTo-Json
$agentRes = Invoke-RestJson -Method Post -Uri "$base/organizations/$orgId/agents" -Headers @{ Authorization = "Bearer $token" } -Body $agentBody
$agentId = $agentRes.id

# Request generation
$payload = @{ lang = 'java'; agent_id = $agentId; organization_id = $orgId } | ConvertTo-Json
$genRes = Invoke-RestJson -Method Post -Uri "$base/sdk/generate" -Headers @{ Authorization = "Bearer $token" } -Body $payload
$job = $genRes.job_id
Write-Host "Job:" $job

# Poll until ready
for ($i=0; $i -lt 30; $i++) {
  Start-Sleep -Seconds 2
  $st = Invoke-RestJson -Method Get -Uri "$base/sdk/generate/$job" -Headers @{ Authorization = "Bearer $token" }
  Write-Host "Status:" $st.status
  if ($st.status -eq 'ready') { break }
  if ($st.status -eq 'error') { throw "Job failed: $($st.error)" }
}

# Attempt public download (no auth)
$out = Join-Path $env:TEMP "aura_public_$job.zip"
$response = Invoke-WebRequest -UseBasicParsing -Headers @{ 'Accept-Encoding' = 'identity' } -Uri "$base/sdk/public/download-generated/$job" -OutFile $out -PassThru
Write-Host "HTTP:" $response.StatusCode
Write-Host "ZIP size:" ((Get-Item $out).Length)
