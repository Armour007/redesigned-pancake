$ErrorActionPreference = 'Stop'
$base = 'http://localhost:8081'

# Register and login
$rand = Get-Random -Maximum 1000000
$email = "g$rand@example.com"
$register = @{ full_name = 'Gen User'; email = $email; password = 'P@ssw0rd12345!' } | ConvertTo-Json
Invoke-RestMethod -Method Post -Uri "$base/auth/register" -ContentType 'application/json' -Body $register | Out-Null
$loginBody = @{ email = $email; password = 'P@ssw0rd12345!' } | ConvertTo-Json
$loginRes = Invoke-RestMethod -Method Post -Uri "$base/auth/login" -ContentType 'application/json' -Body $loginBody
$token = $loginRes.token

# Get org
$orgs = Invoke-RestMethod -Method Get -Uri "$base/organizations/mine" -Headers @{ Authorization = "Bearer $token" }
$orgId = $orgs[0].id

# Create an agent for context
$agentBody = @{ name = 'gen-agent'; description = 'codegen test' } | ConvertTo-Json
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

# Download generated zip
$out = Join-Path $env:TEMP "aura_gen_$job.zip"
Invoke-WebRequest -UseBasicParsing -Uri "$base/sdk/download-generated/$job" -Headers @{ Authorization = "Bearer $token" } -OutFile $out
Write-Host "Generated ZIP size:" ((Get-Item $out).Length)
