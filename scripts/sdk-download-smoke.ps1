$ErrorActionPreference = 'Stop'
$base = 'http://localhost:8081'

# Register and login to get JWT
$rand = Get-Random -Maximum 1000000
$email = "e$rand@example.com"
$register = @{ full_name = 'Smoke User'; email = $email; password = 'P@ssw0rd12345!' } | ConvertTo-Json
$regRes = Invoke-RestMethod -Method Post -Uri "$base/auth/register" -ContentType 'application/json' -Body $register
$loginBody = @{ email = $email; password = 'P@ssw0rd12345!' } | ConvertTo-Json
$loginRes = Invoke-RestMethod -Method Post -Uri "$base/auth/login" -ContentType 'application/json' -Body $loginBody
$token = $loginRes.token

# Create an agent
$orgs = Invoke-RestMethod -Method Get -Uri "$base/organizations/mine" -Headers @{ Authorization = "Bearer $token" }
$orgId = $orgs[0].id
$agentBody = @{ name = 'zip-agent'; description = 'zip test' } | ConvertTo-Json
$agentRes = Invoke-RestMethod -Method Post -Uri "$base/organizations/$orgId/agents" -Headers @{ Authorization = "Bearer $token" } -ContentType 'application/json' -Body $agentBody
$agentId = $agentRes.id

# Download curated Node SDK
$zip = Join-Path $env:TEMP 'aura_node.zip'
$u = "$base/sdk/download?lang=node&agent_id=$agentId&action=deploy"
Invoke-WebRequest -UseBasicParsing -Uri $u -Headers @{ Authorization = "Bearer $token" } -OutFile $zip
if (Test-Path $zip) {
  $len = (Get-Item $zip).Length
  Write-Host "ZIP size:" $len
} else {
  Write-Host "ZIP not found"
}
